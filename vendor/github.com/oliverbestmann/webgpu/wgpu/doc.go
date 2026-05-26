// Package wgpu provides WebGPU bindings for golang
//
// ## Memory Management
// The wgpu package offers two ways of memory management that complement each other.
// In good go-fashion garbage collection works as expected. Every wgpu object returned
// by the api will be released automatically once it is garbage collected.
//
// There is a small caviat when using garbage collection with externally allocated objects.
// The golang garbage collector does not know about the size of the wgpu object you've
// allocated. E.g. for a large texture it only sees the ~64byte allocation of wgpu.Texture,
// not the 20mb of texture data in gpu memory. Allocating and keeping around 100 textures
// is not a problem for golangs garbage collector, it might be for your system.
//
// To handle this case, this package also offers a way to manually release resources.
// Each wgpu object has a Release() method. Once called, it will immediately release
// the underlaying wgpu object. It is okay to call Release() on an already released
// instance.
//
// Sometimes you want to hand out wgpu objects from your code that the caller is not supposed
// to release, e.g. for caching. You can mark those instances as "shared" by calling the
// wgpu.Share method on the object. If you do that, the object returned will only be released
// during garbage collection, and explicit calls to Release will be a noop.
package wgpu
