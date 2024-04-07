#include <stdlib.h>
#include <jni.h>

#ifndef __ANDROID_API__
JavaVM *_jni_CreateJavaVM(char **optstrings, int nOptions) {
	JavaVM *vm;
	JNIEnv *p_env;
	JavaVMInitArgs vm_args;
	JavaVMOption *options = (JavaVMOption *)malloc(sizeof(JavaVMOption)*nOptions);
	if (options != 0) {
		int i;
		for (i = 0; i < nOptions; i++) {
			options[i].optionString = optstrings[i];
		}
		vm_args.nOptions = nOptions;
		vm_args.options = options;
	}
	vm_args.version = JNI_VERSION_1_6;
	vm_args.ignoreUnrecognized = 0;
	jint res = JNI_CreateJavaVM(&vm, &p_env, &vm_args);
	if (res < 0) {
		printf("Can't create Java VM\n");
		return 0;
	}
	return vm;
}
#endif /* __ANDROID_API__ */

jint _jni_AttachCurrentThread(JavaVM *vm, JNIEnv **p_env, void *thr_args) {
	return (*vm)->AttachCurrentThread(vm, p_env, thr_args);
}

jint _jni_DetachCurrentThread(JavaVM *vm) {
	return (*vm)->DetachCurrentThread(vm);
}

jint _jni_GetEnv(JavaVM *vm, JNIEnv **env, jint version) {
	return (*vm)->GetEnv(vm, (void **)env, version);
}

jclass _jni_FindClass(JNIEnv *env, const char *name) {
	return (*env)->FindClass(env, name);
}

jthrowable _jni_ExceptionOccurred(JNIEnv *env) {
	return (*env)->ExceptionOccurred(env);
}

void _jni_ExceptionClear(JNIEnv *env) {
	(*env)->ExceptionClear(env);
}

jclass _jni_GetObjectClass(JNIEnv *env, jobject obj) {
	return (*env)->GetObjectClass(env, obj);
}

jmethodID _jni_GetMethodID(JNIEnv *env, jclass clazz, const char *name, const char *sig) {
	return (*env)->GetMethodID(env, clazz, name, sig);
}

jmethodID _jni_GetStaticMethodID(JNIEnv *env, jclass clazz, const char *name, const char *sig) {
	return (*env)->GetStaticMethodID(env, clazz, name, sig);
}

jfieldID _jni_GetFieldID(JNIEnv *env, jclass clazz, const char *name, const char *sig) {
    return (*env)->GetFieldID(env, clazz, name, sig);
}

jfieldID _jni_GetStaticFieldID(JNIEnv *env, jclass clazz, const char *name, const char *sig) {
    return (*env)->GetStaticFieldID(env, clazz, name, sig);
}

jsize _jni_GetStringLength(JNIEnv *env, jstring str) {
	return (*env)->GetStringLength(env, str);
}

const jchar *_jni_GetStringChars(JNIEnv *env, jstring str) {
	return (*env)->GetStringChars(env, str, NULL);
}

jstring _jni_NewString(JNIEnv *env, const jchar *unicodeChars, jsize len) {
	return (*env)->NewString(env, unicodeChars, len);
}

jboolean _jni_IsSameObject(JNIEnv *env, jobject ref1, jobject ref2) {
	return (*env)->IsSameObject(env, ref1, ref2);
}

jboolean _jni_IsInstanceOf(JNIEnv *env, jobject obj, jclass cls) {
	return (*env)->IsInstanceOf(env, obj, cls);
}

jobject _jni_NewGlobalRef(JNIEnv *env, jobject obj) {
	return (*env)->NewGlobalRef(env, obj);
}

void _jni_DeleteGlobalRef(JNIEnv *env, jobject obj) {
	(*env)->DeleteGlobalRef(env, obj);
}

jobject _jni_NewLocalRef(JNIEnv *env, jobject obj) {
	return (*env)->NewLocalRef(env, obj);
}

void _jni_DeleteLocalRef(JNIEnv *env, jobject obj) {
	(*env)->DeleteLocalRef(env, obj);
}

jobject _jni_NewObjectA(JNIEnv *env, jclass cls, jmethodID method, jvalue *args) {
	return (*env)->NewObject(env, cls, method, args);
}

void _jni_CallStaticVoidMethodA(JNIEnv *env, jclass cls, jmethodID method, jvalue *args) {
	(*env)->CallStaticVoidMethodA(env, cls, method, args);
}

jint _jni_CallStaticIntMethodA(JNIEnv *env, jclass cls, jmethodID method, jvalue *args) {
	return (*env)->CallStaticIntMethodA(env, cls, method, args);
}

jboolean _jni_CallStaticBooleanMethodA(JNIEnv *env, jclass cls, jmethodID method, jvalue *args) {
	return (*env)->CallStaticBooleanMethodA(env, cls, method, args);
}

jobject _jni_CallStaticObjectMethodA(JNIEnv *env, jclass cls, jmethodID method, jvalue *args) {
	return (*env)->CallStaticObjectMethodA(env, cls, method, args);
}

jobject _jni_CallObjectMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallObjectMethodA(env, obj, method, args);
}

jint _jni_CallIntMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallIntMethodA(env, obj, method, args);
}

void _jni_CallVoidMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	(*env)->CallVoidMethodA(env, obj, method, args);
}

jboolean _jni_CallBooleanMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallBooleanMethodA(env, obj, method, args);
}

jbyte _jni_CallByteMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallByteMethodA(env, obj, method, args);
}

jchar _jni_CallCharMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallCharMethodA(env, obj, method, args);
}

jshort _jni_CallShortMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallShortMethodA(env, obj, method, args);
}

jlong _jni_CallLongMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallLongMethodA(env, obj, method, args);
}

jfloat _jni_CallFloatMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallFloatMethodA(env, obj, method, args);
}

jdouble _jni_CallDoubleMethodA(JNIEnv *env, jobject obj, jmethodID method, jvalue *args) {
	return (*env)->CallDoubleMethodA(env, obj, method, args);
}

jbyteArray _jni_NewByteArray(JNIEnv *env, jsize length) {
	return (*env)->NewByteArray(env, length);
}

jbyte *_jni_GetByteArrayElements(JNIEnv *env, jbyteArray arr) {
	return (*env)->GetByteArrayElements(env, arr, NULL);
}

void _jni_ReleaseByteArrayElements(JNIEnv *env, jbyteArray arr, jbyte *elems, jint mode) {
	(*env)->ReleaseByteArrayElements(env, arr, elems, mode);
}

jsize _jni_GetArrayLength(JNIEnv *env, jarray arr) {
	return (*env)->GetArrayLength(env, arr);
}

jobjectArray _jni_NewObjectArray(JNIEnv *env, jsize length, jclass elementClass, jobject initialElement) {
	return (*env)->NewObjectArray(env, length, elementClass, initialElement);
}

jobject _jni_GetObjectArrayElement(JNIEnv *env, jobjectArray array, jsize index) {
  return (*env)->GetObjectArrayElement(env, array, index);
}

void _jni_SetObjectArrayElement(JNIEnv *env, jobjectArray array, jsize index, jobject value) {
  return (*env)->SetObjectArrayElement(env, array, index, value);
}

jobject _jni_GetStaticObjectField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticObjectField(env, clazz, fieldID);
}

jboolean _jni_GetStaticBooleanField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticBooleanField(env, clazz, fieldID);
}

jbyte _jni_GetStaticByteField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticByteField(env, clazz, fieldID);
}

jchar _jni_GetStaticCharField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticCharField(env, clazz, fieldID);
}

jshort _jni_GetStaticShortField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticShortField(env, clazz, fieldID);
}

jint _jni_GetStaticIntField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticIntField(env, clazz, fieldID);
}

jlong _jni_GetStaticLongField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticLongField(env, clazz, fieldID);
}

jfloat _jni_GetStaticFloatField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticFloatField(env, clazz, fieldID);
}

jdouble _jni_GetStaticDoubleField(JNIEnv *env, jclass clazz, jfieldID fieldID) {
    return (*env)->GetStaticDoubleField(env, clazz, fieldID);
}

jobject _jni_GetObjectField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetObjectField(env, obj, fieldID);
}

jboolean _jni_GetBooleanField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetBooleanField(env, obj, fieldID);
}

jbyte _jni_GetByteField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetByteField(env, obj, fieldID);
}

jchar _jni_GetCharField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetCharField(env, obj, fieldID);
}

jshort _jni_GetShortField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetShortField(env, obj, fieldID);
}

jint _jni_GetIntField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetIntField(env, obj, fieldID);
}

jlong _jni_GetLongField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetLongField(env, obj, fieldID);
}

jfloat _jni_GetFloatField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetFloatField(env, obj, fieldID);
}

jdouble _jni_GetDoubleField(JNIEnv *env, jobject obj, jfieldID fieldID) {
    return (*env)->GetDoubleField(env, obj, fieldID);
}

