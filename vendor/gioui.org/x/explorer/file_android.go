package explorer

import (
	"errors"
	"io"

	"gioui.org/app"
	"git.wow.st/gmp/jni"
)

//go:generate javac -source 8 -target 8  -bootclasspath $ANDROID_HOME/platforms/android-30/android.jar -d $TEMP/explorer_file_android/classes file_android.java
//go:generate jar cf file_android.jar -C $TEMP/explorer_file_android/classes .

type File struct {
	stream    jni.Object
	libObject jni.Object
	libClass  jni.Class

	fileRead  jni.MethodID
	fileWrite jni.MethodID
	fileClose jni.MethodID
	getError  jni.MethodID

	sharedBuffer    jni.Object
	sharedBufferLen int
	isClosed        bool
}

func newFile(env jni.Env, stream jni.Object) (*File, error) {
	f := &File{stream: stream}

	class, err := jni.LoadClass(env, jni.ClassLoaderFor(env, jni.Object(app.AppContext())), "org/gioui/x/explorer/file_android")
	if err != nil {
		return nil, err
	}

	obj, err := jni.NewObject(env, class, jni.GetMethodID(env, class, "<init>", `()V`))
	if err != nil {
		return nil, err
	}

	// For some reason, using `f.stream` as argument for a constructor (`public file_android(Object j) {}`) doesn't work.
	if err := jni.CallVoidMethod(env, obj, jni.GetMethodID(env, class, "setHandle", `(Ljava/lang/Object;)V`), jni.Value(f.stream)); err != nil {
		return nil, err
	}

	f.libObject = jni.NewGlobalRef(env, obj)
	f.libClass = jni.Class(jni.NewGlobalRef(env, jni.Object(class)))
	f.fileRead = jni.GetMethodID(env, f.libClass, "fileRead", "([B)I")
	f.fileWrite = jni.GetMethodID(env, f.libClass, "fileWrite", "([B)Z")
	f.fileClose = jni.GetMethodID(env, f.libClass, "fileClose", "()Z")
	f.getError = jni.GetMethodID(env, f.libClass, "getError", "()Ljava/lang/String;")

	return f, nil

}

func (f *File) Read(b []byte) (n int, err error) {
	if f == nil || f.isClosed {
		return 0, io.ErrClosedPipe
	}
	if len(b) == 0 {
		return 0, nil // Avoid unnecessary call to JNI.
	}

	err = jni.Do(jni.JVMFor(app.JavaVM()), func(env jni.Env) error {
		if len(b) != f.sharedBufferLen {
			f.sharedBuffer = jni.Object(jni.NewGlobalRef(env, jni.Object(jni.NewByteArray(env, b))))
			f.sharedBufferLen = len(b)
		}

		size, err := jni.CallIntMethod(env, f.libObject, f.fileRead, jni.Value(f.sharedBuffer))
		if err != nil {
			return err
		}
		if size <= 0 {
			return f.lastError(env)
		}

		n = copy(b, jni.GetByteArrayElements(env, jni.ByteArray(f.sharedBuffer))[:int(size)])
		return nil
	})
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, err
}

func (f *File) Write(b []byte) (n int, err error) {
	if f == nil || f.isClosed {
		return 0, io.ErrClosedPipe
	}
	if len(b) == 0 {
		return 0, nil // Avoid unnecessary call to JNI.
	}

	err = jni.Do(jni.JVMFor(app.JavaVM()), func(env jni.Env) error {
		ok, err := jni.CallBooleanMethod(env, f.libObject, f.fileWrite, jni.Value(jni.NewByteArray(env, b)))
		if err != nil {
			return err
		}
		if !ok {
			return f.lastError(env)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(b), err
}

func (f *File) Close() error {
	if f == nil || f.isClosed {
		return io.ErrClosedPipe
	}

	return jni.Do(jni.JVMFor(app.JavaVM()), func(env jni.Env) error {
		ok, err := jni.CallBooleanMethod(env, f.libObject, f.fileClose)
		if err != nil {
			return err
		}
		if !ok {
			return f.lastError(env)
		}

		f.isClosed = true
		jni.DeleteGlobalRef(env, f.stream)
		jni.DeleteGlobalRef(env, f.libObject)
		jni.DeleteGlobalRef(env, jni.Object(f.libClass))
		if f.sharedBuffer != 0 {
			jni.DeleteGlobalRef(env, f.sharedBuffer)
		}

		return nil
	})
}

func (f *File) lastError(env jni.Env) error {
	message, err := jni.CallObjectMethod(env, f.libObject, f.getError)
	if err != nil {
		return err
	}
	if err := jni.GoString(env, jni.String(message)); len(err) > 0 {
		return errors.New(err)
	}
	return err
}
