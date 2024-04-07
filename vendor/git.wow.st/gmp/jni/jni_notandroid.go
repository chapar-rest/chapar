// +build !android

package jni

/*
#include <stdlib.h>
#include <jni.h>
#include "gojni.h"
*/
import "C"

// CreateJavaVM creates a new Java VM with the options specified (if any).
// This should not be called more than once as it can result in an error if
// a JVM already exists for a given process.
//
// This is not implemented on Android and is therefore excluded on that
// platform by build flags and C.
func CreateJavaVM() JVM {
	jvm := C._jni_CreateJavaVM((**C.char)(nil), 0)
	return JVM{ jvm: jvm }
}
