package wgpu

type Shareable interface {
	markShared()
}

// Share marks the given wgpu object as shared. A call to Release() will become a
// noop and the object will only be released once the garbage collector collects
// the object.
//
// Check the package documentation for more details on memory management.
func Share[T Shareable](s T) T {
	s.markShared()
	return s
}
