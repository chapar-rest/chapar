// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jni implements various helper functions for communicating with the
// Android JVM though JNI.
package jni

/*
#cgo CFLAGS: -Wall

#include <stdlib.h>
#include <jni.h>
#include "gojni.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"unicode/utf16"
	"unsafe"
)

type JVM struct {
	jvm *C.JavaVM
}

type Env struct {
	env *C.JNIEnv
}

type (
	Class       C.jclass
	Object      C.jobject
	MethodID    C.jmethodID
	FieldID     C.jfieldID
	String      C.jstring
	ByteArray   C.jbyteArray
	ObjectArray C.jobjectArray
	Size        C.jsize
	Value       uint64 // All JNI types fit into 64-bits.
)

const (
	TRUE  = C.JNI_TRUE
	FALSE = C.JNI_FALSE
)

// JVMFor creates a JVM object, interpreting the given uintptr as a pointer
// to a C.JavaVM object.
func JVMFor(jvmPtr uintptr) JVM {
	return JVM{
		jvm: (*C.JavaVM)(unsafe.Pointer(jvmPtr)),
	}
}

// EnvFor creates an Env object, interpreting the given uintptr as a pointer
// to a C.JNIEnv object.
func EnvFor(envPtr uintptr) Env {
	return Env{
		env: (*C.JNIEnv)(unsafe.Pointer(envPtr)),
	}
}

// Do invokes a function with a temporary JVM environment. The
// environment is not valid after the function returns.
func Do(vm JVM, f func(env Env) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var env *C.JNIEnv
	if res := C._jni_GetEnv(vm.jvm, &env, C.JNI_VERSION_1_6); res != C.JNI_OK {
		if res != C.JNI_EDETACHED {
			panic(fmt.Errorf("JNI GetEnv failed with error %d", res))
		}
		if C._jni_AttachCurrentThread(vm.jvm, &env, nil) != C.JNI_OK {
			panic(errors.New("runInJVM: AttachCurrentThread failed"))
		}
		defer C._jni_DetachCurrentThread(vm.jvm)
	}

	return f(Env{env})
}

func varArgs(args []Value) *C.jvalue {
	if len(args) == 0 {
		return nil
	}
	return (*C.jvalue)(unsafe.Pointer(&args[0]))
}

// IsSameObject returns true if the two given objects refer to the same
// Java object.
func IsSameObject(e Env, ref1, ref2 Object) bool {
	same := C._jni_IsSameObject(e.env, C.jobject(ref1), C.jobject(ref2))
	return same == TRUE
}

// CallStaticIntMethod calls a static method on a Java class, returning an int.
func CallStaticIntMethod(e Env, cls Class, method MethodID, args ...Value) (int, error) {
	res := C._jni_CallStaticIntMethodA(e.env, C.jclass(cls), C.jmethodID(method), varArgs(args))
	return int(res), exception(e)
}

// CallStaticBooleanMethod calls a static method on a Java class, returning an int.
func CallStaticBooleanMethod(e Env, cls Class, method MethodID, args ...Value) (bool, error) {
	res := C._jni_CallStaticBooleanMethodA(e.env, C.jclass(cls), C.jmethodID(method), varArgs(args))
	return res == TRUE, exception(e)
}

// FindClass returns a reference to a Java class with a given name, using the
// JVM's default class loader. Any exceptions caused by the underlying JNI call
// (for example if the class is not found) will result in a panic.
func FindClass(e Env, name string) Class {
	mname := C.CString(name)
	defer C.free(unsafe.Pointer(mname))
	res := C._jni_FindClass(e.env, mname)
	if err := exception(e); err != nil {
		panic(err)
	}
	return Class(res)
}

// NewObject creates a new object given a class, initializer method, and
// initializer arguments (if any).
func NewObject(e Env, cls Class, method MethodID, args ...Value) (Object, error) {
	res := C._jni_NewObjectA(e.env, C.jclass(cls), C.jmethodID(method), varArgs(args))
	return Object(res), exception(e)
}

// CallStaticVoidMethod calls a static method on a Java class, returning
// nothing.
func CallStaticVoidMethod(e Env, cls Class, method MethodID, args ...Value) error {
	C._jni_CallStaticVoidMethodA(e.env, C.jclass(cls), C.jmethodID(method), varArgs(args))
	return exception(e)
}

// CallVoidMethod calls a method on an object, returning nothing.
func CallVoidMethod(e Env, obj Object, method MethodID, args ...Value) error {
	C._jni_CallVoidMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return exception(e)
}

// CallStaticObjectMethod calls a static method on a class, returning a
// Java object
func CallStaticObjectMethod(e Env, cls Class, method MethodID, args ...Value) (Object, error) {
	res := C._jni_CallStaticObjectMethodA(e.env, C.jclass(cls), C.jmethodID(method), varArgs(args))
	return Object(res), exception(e)
}

// CallObjectMethod calls a method on an object, returning a Java object.
func CallObjectMethod(e Env, obj Object, method MethodID, args ...Value) (Object, error) {
	res := C._jni_CallObjectMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return Object(res), exception(e)
}

// CallIntMethod calls a method on an object, returning an int32.
func CallIntMethod(e Env, obj Object, method MethodID, args ...Value) (int32, error) {
	res := C._jni_CallIntMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return int32(res), exception(e)
}

// CallBooleanMethod calls a method on an object, returning a bool.
func CallBooleanMethod(e Env, obj Object, method MethodID, args ...Value) (bool, error) {
	res := C._jni_CallBooleanMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return res == TRUE, exception(e)
}

// CallByteMethod calls a method on an object, returning a byte.
func CallByteMethod(e Env, obj Object, method MethodID, args ...Value) (byte, error) {
	res := C._jni_CallByteMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return byte(res), exception(e)
}

// CallCharMethod calls a method on an object, returning a rune.
func CallCharMethod(e Env, obj Object, method MethodID, args ...Value) (rune, error) {
	res := C._jni_CallCharMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return rune(res), exception(e)
}

// CallShortMethod calls a method on an object, returning an int32.
func CallShortMethod(e Env, obj Object, method MethodID, args ...Value) (int16, error) {
	res := C._jni_CallShortMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return int16(res), exception(e)
}

// CallLongMethod calls a method on an object, returning an int64.
func CallLongMethod(e Env, obj Object, method MethodID, args ...Value) (int64, error) {
	res := C._jni_CallLongMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return int64(res), exception(e)
}

// CallFloatMethod calls a method on an object, returning a float32.
func CallFloatMethod(e Env, obj Object, method MethodID, args ...Value) (float32, error) {
	res := C._jni_CallFloatMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return float32(res), exception(e)
}

// CallDoubleMethod calls a method on an object, returning a float64.
func CallDoubleMethod(e Env, obj Object, method MethodID, args ...Value) (float64, error) {
	res := C._jni_CallDoubleMethodA(e.env, C.jobject(obj), C.jmethodID(method), varArgs(args))
	return float64(res), exception(e)
}

// GetByteArrayElements returns the contents of the array.
func GetByteArrayElements(e Env, jarr ByteArray) []byte {
	size := C._jni_GetArrayLength(e.env, C.jarray(jarr))
	elems := C._jni_GetByteArrayElements(e.env, C.jbyteArray(jarr))
	defer C._jni_ReleaseByteArrayElements(e.env, C.jbyteArray(jarr), elems, 0)
	backing := (*(*[1 << 30]byte)(unsafe.Pointer(elems)))[:size:size]
	s := make([]byte, len(backing))
	copy(s, backing)
	return s
}

// NewByteArray allocates a Java byte array with the content. It
// panics if the allocation fails.
func NewByteArray(e Env, content []byte) ByteArray {
	jarr := C._jni_NewByteArray(e.env, C.jsize(len(content)))
	if jarr == 0 {
		panic(fmt.Errorf("jni: NewByteArray(%d) failed", len(content)))
	}
	elems := C._jni_GetByteArrayElements(e.env, jarr)
	defer C._jni_ReleaseByteArrayElements(e.env, jarr, elems, 0)
	backing := (*(*[1 << 30]byte)(unsafe.Pointer(elems)))[:len(content):len(content)]
	copy(backing, content)
	return ByteArray(jarr)
}

func NewObjectArray(e Env, len Size, class Class, elem Object) ObjectArray {
	jarr := C._jni_NewObjectArray(e.env, C.jsize(len), C.jclass(class), C.jobject(elem))
	if jarr == 0 {
		panic(fmt.Errorf("jni: NewObjectArray failed"))
	}
	return ObjectArray(jarr)
}

func GetObjectArrayElement(e Env, jarr ObjectArray, index Size) (Object, error) {
	jobj := C._jni_GetObjectArrayElement(e.env, C.jobjectArray(jarr), C.jsize(index))
	return Object(jobj), exception(e)
}

func SetObjectArrayElement(e Env, jarr ObjectArray, index Size, value Object) error {
	C._jni_SetObjectArrayElement(
		e.env,
		C.jobjectArray(jarr),
		C.jsize(index),
		C.jobject(value))
	return exception(e)
}

// ClassLoader returns a reference to the Java ClassLoader associated
// with obj.
func ClassLoaderFor(e Env, obj Object) Object {
	cls := GetObjectClass(e, obj)
	getClassLoader := GetMethodID(e, cls, "getClassLoader", "()Ljava/lang/ClassLoader;")
	clsLoader, err := CallObjectMethod(e, Object(obj), getClassLoader)
	if err != nil {
		// Class.getClassLoader should never fail.
		panic(err)
	}
	return Object(clsLoader)
}

// LoadClass invokes the underlying ClassLoader's loadClass method and
// returns the class.
func LoadClass(e Env, loader Object, class string) (Class, error) {
	cls := GetObjectClass(e, loader)
	loadClass := GetMethodID(e, cls, "loadClass", "(Ljava/lang/String;)Ljava/lang/Class;")
	name := JavaString(e, class)
	loaded, err := CallObjectMethod(e, loader, loadClass, Value(name))
	if err != nil {
		return 0, err
	}
	return Class(loaded), exception(e)
}

// exception returns an error corresponding to the pending
// exception, and clears it. exceptionError returns nil if no
// exception is pending.
func exception(e Env) error {
	thr := C._jni_ExceptionOccurred(e.env)
	if thr == 0 {
		return nil
	}
	C._jni_ExceptionClear(e.env)
	cls := GetObjectClass(e, Object(thr))
	toString := GetMethodID(e, cls, "toString", "()Ljava/lang/String;")
	msg, err := CallObjectMethod(e, Object(thr), toString)
	if err != nil {
		return err
	}
	return errors.New(GoString(e, String(msg)))
}

// GetObjectClass returns the Java Class for an Object.
func GetObjectClass(e Env, obj Object) Class {
	if obj == 0 {
		panic("null object")
	}
	// GetObjectClass does not throw any exceptions
	cls := C._jni_GetObjectClass(e.env, C.jobject(obj))
	return Class(cls)
}

// IsInstanceOf returns true if the given object is an instance of the
// given class.
func IsInstanceOf(e Env, obj Object, cls Class) bool {
	if obj == 0 {
		panic("null object")
	}
	if cls == 0 {
		panic("null class")
	}
	// Note: does not throw any exceptions
	res := C._jni_IsInstanceOf(e.env, C.jobject(obj), C.jclass(cls))
	return res == TRUE
}

// GetStaticMethodID returns the id for a static method. It panics if the method
// wasn't found.
func GetStaticMethodID(e Env, cls Class, name, signature string) MethodID {
	mname := C.CString(name)
	defer C.free(unsafe.Pointer(mname))
	msig := C.CString(signature)
	defer C.free(unsafe.Pointer(msig))
	m := C._jni_GetStaticMethodID(e.env, C.jclass(cls), mname, msig)
	if err := exception(e); err != nil {
		panic(err)
	}
	return MethodID(m)
}

// GetFieldID returns the id for a field. It panics if the field wasn't found.
func GetFieldID(e Env, cls Class, name, signature string) FieldID {
	mname := C.CString(name)
	defer C.free(unsafe.Pointer(mname))
	msig := C.CString(signature)
	defer C.free(unsafe.Pointer(msig))
	m := C._jni_GetFieldID(e.env, C.jclass(cls), mname, msig)
	if err := exception(e); err != nil {
		panic(err)
	}
	return FieldID(m)
}

// GetStaticFieldID returns the id for a static field. It panics if the field
// wasn't found.
func GetStaticFieldID(e Env, cls Class, name, signature string) FieldID {
	mname := C.CString(name)
	defer C.free(unsafe.Pointer(mname))
	msig := C.CString(signature)
	defer C.free(unsafe.Pointer(msig))
	m := C._jni_GetStaticFieldID(e.env, C.jclass(cls), mname, msig)
	if err := exception(e); err != nil {
		panic(err)
	}
	return FieldID(m)
}

// GetMethodID returns the id for a method. It panics if the method
// wasn't found.
func GetMethodID(e Env, cls Class, name, signature string) MethodID {
	mname := C.CString(name)
	defer C.free(unsafe.Pointer(mname))
	msig := C.CString(signature)
	defer C.free(unsafe.Pointer(msig))
	m := C._jni_GetMethodID(e.env, C.jclass(cls), mname, msig)
	if err := exception(e); err != nil {
		panic(err)
	}
	return MethodID(m)
}

// NewGlobalRef creates a new global reference.
func NewGlobalRef(e Env, obj Object) Object {
	return Object(C._jni_NewGlobalRef(e.env, C.jobject(obj)))
}

// DeleteGlobalRef delets a global reference.
func DeleteGlobalRef(e Env, obj Object) {
	C._jni_DeleteGlobalRef(e.env, C.jobject(obj))
}

// NewLocalRef creates a new local reference to the given object.
func NewLocalRef(e Env, obj Object) Object {
	return Object(C._jni_NewLocalRef(e.env, C.jobject(obj)))
}

// DeleteLocalRef delets a local reference.
func DeleteLocalRef(e Env, obj Object) {
	C._jni_DeleteLocalRef(e.env, C.jobject(obj))
}

// JavaString converts the string to a JVM jstring.
func JavaString(e Env, str string) String {
	if str == "" {
		return 0
	}
	utf16Chars := utf16.Encode([]rune(str))
	res := C._jni_NewString(e.env, (*C.jchar)(unsafe.Pointer(&utf16Chars[0])), C.int(len(utf16Chars)))
	return String(res)
}

// GoString converts the JVM jstring to a Go string.
func GoString(e Env, str String) string {
	if str == 0 {
		return ""
	}
	strlen := C._jni_GetStringLength(e.env, C.jstring(str))
	chars := C._jni_GetStringChars(e.env, C.jstring(str))
	var utf16Chars []uint16
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&utf16Chars))
	hdr.Data = uintptr(unsafe.Pointer(chars))
	hdr.Cap = int(strlen)
	hdr.Len = int(strlen)
	utf8 := utf16.Decode(utf16Chars)
	return string(utf8)
}

// GetStaticObjectField looks up the value of a static field of type Object.
// This should never throw an exception.
func GetStaticObjectField(env Env, clazz Class, fieldID FieldID) Object {
	value := C._jni_GetStaticObjectField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return Object(value)
}

// GetStaticBooleanField looks up the value of a static field of type boolean.
// This should never throw an exception.
func GetStaticBooleanField(env Env, clazz Class, fieldID FieldID) bool {
	value := C._jni_GetStaticBooleanField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return value != 0
}

// GetStaticByteField looks up the value of a static field of type byte.
// This should never throw an exception.
func GetStaticByteField(env Env, clazz Class, fieldID FieldID) byte {
	value := C._jni_GetStaticByteField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return byte(value)
}

// GetStaticCharField looks up the value of a static field of type char.
// This should never throw an exception.
func GetStaticCharField(env Env, clazz Class, fieldID FieldID) byte {
	value := C._jni_GetStaticCharField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return byte(value)
}

// GetStaticShortField looks up the value of a static field of type short.
// This should never throw an exception.
func GetStaticShortField(env Env, clazz Class, fieldID FieldID) int16 {
	value := C._jni_GetStaticShortField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return int16(value)
}

// GetStaticIntField looks up the value of a static field of type int.
// This should never throw an exception.
func GetStaticIntField(env Env, clazz Class, fieldID FieldID) int32 {
	value := C._jni_GetStaticIntField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return int32(value)
}

// GetStaticLongField looks up the value of a static field of type long.
// This should never throw an exception.
func GetStaticLongField(env Env, clazz Class, fieldID FieldID) int64 {
	value := C._jni_GetStaticLongField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return int64(value)
}

// GetStaticFloatField looks up the value of a static field of type float.
// This should never throw an exception.
func GetStaticFloatField(env Env, clazz Class, fieldID FieldID) float32 {
	value := C._jni_GetStaticFloatField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return float32(value)
}

// GetStaticDoubleField looks up the value of a static field of type double.
// This should never throw an exception.
func GetStaticDoubleField(env Env, clazz Class, fieldID FieldID) float64 {
	value := C._jni_GetStaticDoubleField(env.env, C.jclass(clazz), C.jfieldID(fieldID))
	return float64(value)
}

// GetObjectField looks up the value of a static field of type Object.
// This should never throw an exception.
func GetObjectField(env Env, obj Object, fieldID FieldID) Object {
	value := C._jni_GetObjectField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return Object(value)
}

// GetBooleanField looks up the value of a static field of type boolean.
// This should never throw an exception.
func GetBooleanField(env Env, obj Object, fieldID FieldID) bool {
	value := C._jni_GetBooleanField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return value != 0
}

// GetByteField looks up the value of a static field of type byte.
// This should never throw an exception.
func GetByteField(env Env, obj Object, fieldID FieldID) byte {
	value := C._jni_GetByteField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return byte(value)
}

// GetCharField looks up the value of a static field of type char.
// This should never throw an exception.
func GetCharField(env Env, obj Object, fieldID FieldID) byte {
	value := C._jni_GetCharField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return byte(value)
}

// GetShortField looks up the value of a static field of type short.
// This should never throw an exception.
func GetShortField(env Env, obj Object, fieldID FieldID) int16 {
	value := C._jni_GetShortField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return int16(value)
}

// GetIntField looks up the value of a static field of type int.
// This should never throw an exception.
func GetIntField(env Env, obj Object, fieldID FieldID) int32 {
	value := C._jni_GetIntField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return int32(value)
}

// GetLongField looks up the value of a static field of type long.
// This should never throw an exception.
func GetLongField(env Env, obj Object, fieldID FieldID) int64 {
	value := C._jni_GetLongField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return int64(value)
}

// GetFloatField looks up the value of a static field of type float.
// This should never throw an exception.
func GetFloatField(env Env, obj Object, fieldID FieldID) float32 {
	value := C._jni_GetFloatField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return float32(value)
}

// GetDoubleField looks up the value of a static field of type double.
// This should never throw an exception.
func GetDoubleField(env Env, obj Object, fieldID FieldID) float64 {
	value := C._jni_GetDoubleField(env.env, C.jobject(obj), C.jfieldID(fieldID))
	return float64(value)
}
