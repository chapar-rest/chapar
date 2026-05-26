package wgpu

import (
	"sync"
	"unsafe"
)

var pointers []byte

var handles = map[handle]any{}
var handlesMu sync.Mutex

type handle struct {
	pointer unsafe.Pointer
}

// NewHandle returns a handle for a given value.
//
// The handle is valid until the program calls Delete on it. The handle
// uses resources, and this package assumes that C code may hold on to
// the handle, so a program must explicitly call Delete when the handle
// is no longer needed in C code.
//
// The intended use is to pass the returned handle to C code, which
// passes it back to Go, which calls Value.
//
// Compared to cgo.Handle, this handle can be converted to an unsafe.Pointer
// and be used in C as userdata when a void* value is required.
func newHandle(value any) handle {
	handlesMu.Lock()
	defer handlesMu.Unlock()

	if len(pointers) == 0 {
		pointers = make([]byte, 1024*4)
	}

	// get a unique pointer as the identifier of the newly created handle
	h := handle{pointer: unsafe.Pointer(&pointers[0])}
	pointers = pointers[1:]

	handles[h] = value

	return h
}

// lookupHandle converts an unsafe.Pointer obtained by ToPointer back into
// a handle. It will not check if the handle is still valid.
func lookupHandle(p unsafe.Pointer) handle {
	return handle{pointer: p}
}

// Value returns the associated Go value for a valid handle.
//
// The method panics if the handle is invalid.
func (h handle) Value() any {
	handlesMu.Lock()
	defer handlesMu.Unlock()

	if !h.isValid() {
		panic("invalid handle")
	}

	return handles[h]
}

// Delete invalidates a handle. This method should only be called once
// the program no longer needs to pass the handle to C and the C code
// no longer has a copy of the handle value.
//
// The method panics if the handle is invalid.
func (h handle) Delete() {
	handlesMu.Lock()
	defer handlesMu.Unlock()

	if !h.isValid() {
		panic("invalid handle")
	}

	delete(handles, h)
}

// ToPointer converts the handle into a valid unsafe.Pointer
// that you can pass to C code.
func (h handle) ToPointer() unsafe.Pointer {
	return h.pointer
}

func (h handle) isValid() bool {
	_, valid := handles[h]
	return valid
}
