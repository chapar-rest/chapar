//go:build !nogpu && darwin

package yoga

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>

static void yogaApplyMacTitleBar(void* nsWindowPtr, float titleBarHeight, float* outInset) {
	NSWindow* window = (__bridge NSWindow*)nsWindowPtr;
	if (window == nil) {
		if (outInset) *outInset = 78.0f;
		return;
	}
	window.titlebarAppearsTransparent = YES;
	window.titleVisibility = NSWindowTitleHidden;
	window.styleMask |= NSWindowStyleMaskFullSizeContentView;

	NSButton* closeBtn = [window standardWindowButton:NSWindowCloseButton];
	NSButton* zoomBtn = [window standardWindowButton:NSWindowZoomButton];
	if (closeBtn && zoomBtn) {
		NSRect closeFrame = [closeBtn frame];
		NSRect zoomFrame = [zoomBtn frame];
		float clusterH = MAX(closeFrame.size.height, zoomFrame.size.height);
		float yOffset = (titleBarHeight - clusterH) * 0.5f;
		if (yOffset < 0) yOffset = 0;
		for (NSUInteger type = NSWindowCloseButton; type <= NSWindowZoomButton; type++) {
			NSButton* btn = [window standardWindowButton:type];
			if (btn) {
				NSRect f = [btn frame];
				f.origin.y = yOffset;
				[btn setFrame:f];
			}
		}
		NSRect last = [zoomBtn frame];
		float inset = last.origin.x + last.size.width + 12.0f;
		if (outInset) *outInset = inset;
	} else if (outInset) {
		*outInset = 78.0f;
	}
}
*/
import "C"

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/mirzakhany/yoga/theme"
)

func applyCustomTitleBarChrome(window *glfw.Window, h *glfwWindowHost) {
	h.nativeControls = true
	h.undecorated = false
	nsWindow := window.GetCocoaWindow()
	var inset C.float
	titleBarH := float32(theme.DefaultComponentMetrics().TitleBarHeight)
	C.yogaApplyMacTitleBar(unsafe.Pointer(nsWindow), C.float(titleBarH), &inset)
	h.controlsInset = float32(inset)
}
