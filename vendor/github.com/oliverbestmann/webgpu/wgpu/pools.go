package wgpu

import "C"

/*
#include <wgpu.h>
*/
import "C"
import "sync"

var allocErrorCallbackValue = makeTypedPool[errorCallback]()

var allocWGPUBindGroupDescriptor = makeTypedPool[C.WGPUBindGroupDescriptor]()
var allocWGPUCommandEncoderDescriptor = makeTypedPool[C.WGPUCommandEncoderDescriptor]()
var allocWGPURenderPassDescriptor = makeTypedPool[C.WGPURenderPassDescriptor]()
var allocWGPUComputePassDescriptor = makeTypedPool[C.WGPUComputePassDescriptor]()

type typedPool[T any] sync.Pool

func makeTypedPool[T any]() typedPool[T] {
	return typedPool[T](sync.Pool{
		New: func() any {
			var value T
			return &value
		},
	})
}

func (t *typedPool[T]) pool() *sync.Pool {
	return (*sync.Pool)(t)
}

func (t *typedPool[T]) Get() *T {
	return t.pool().Get().(*T)
}

func (t *typedPool[T]) GetZeroed() *T {
	value := t.pool().Get().(*T)

	// clear value before returning to the caller
	var tZero T
	*value = tZero

	return value
}

func (t *typedPool[T]) Put(value *T) {
	t.pool().Put(value)
}
