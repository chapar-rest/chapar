//go:build !nogpu && !js && darwin

package yoga

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework QuartzCore
#import <AppKit/AppKit.h>
#import <QuartzCore/QuartzCore.h>

// yogaSyncMetalLayer configures how Core Animation shows the last presented
// frame while the window's bounds run ahead of it (live resize, fullscreen
// transition). The default gravity (resize) stretches that frame to the new
// bounds, so every widget visibly scales and then snaps once the next frame
// lands. TopLeft keeps it at 1:1 pinned to the top-left corner; any newly
// exposed strip shows the layer background until the next frame fills it.
static void yogaSyncMetalLayer(void* nsWindowPtr, float r, float g, float b, float a) {
	NSWindow* window = (__bridge NSWindow*)nsWindowPtr;
	if (window == nil) {
		return;
	}
	CALayer* layer = window.contentView.layer;
	if (![layer isKindOfClass:[CAMetalLayer class]]) {
		return;
	}
	[CATransaction begin];
	[CATransaction setDisableActions:YES];
	layer.contentsGravity = kCAGravityTopLeft;
	// Without resize gravity the drawable is shown at contentsScale pixels per
	// point, so it must match the backing scale (also after display changes).
	layer.contentsScale = window.backingScaleFactor;
	CGColorRef bg = CGColorCreateSRGB(r, g, b, a);
	layer.backgroundColor = bg;
	CGColorRelease(bg);
	[CATransaction commit];
}
*/
import "C"

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/mirzakhany/yoga/render"
)

// syncSurfaceLayer keeps the window's CAMetalLayer scale and background in step
// with the window. Call after creating the surface, on framebuffer resize
// (backing scale may change), and when the clear color changes.
func syncSurfaceLayer(window *glfw.Window, bg render.Color) {
	ns := window.GetCocoaWindow()
	if ns == nil {
		return
	}
	C.yogaSyncMetalLayer(unsafe.Pointer(ns), C.float(bg.R), C.float(bg.G), C.float(bg.B), C.float(bg.A))
}
