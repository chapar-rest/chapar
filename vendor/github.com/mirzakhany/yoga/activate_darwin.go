//go:build darwin && !js

package yoga

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>

// yogaActivateInstance brings pid's windows to the front when it runs the
// same app as this process: the same bundle, or the same executable.
static int yogaActivateInstance(int pid) {
	NSRunningApplication* app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
	if (app == nil || app.terminated) {
		return 0;
	}
	NSRunningApplication* me = [NSRunningApplication currentApplication];
	BOOL same = (me.bundleIdentifier != nil && [me.bundleIdentifier isEqualToString:app.bundleIdentifier]) ||
		(me.executableURL != nil && [me.executableURL isEqual:app.executableURL]);
	if (!same) {
		return 0;
	}
	return [app activateWithOptions:NSApplicationActivateAllWindows] ? 1 : 0;
}
*/
import "C"

// ActivateInstance brings the windows of another instance of this app,
// running as pid, to the front, and reports whether pid is one. macOS keeps
// ⌘` for cycling one process's windows, so an app run once per window
// switches between its instances with this. It reports false on other
// platforms.
func ActivateInstance(pid int) bool { return C.yogaActivateInstance(C.int(pid)) == 1 }
