//go:build !nogpu && !js && darwin

package yoga

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>
#include <stdlib.h>
#include <string.h>

// yogaClipboardPNG copies the pasteboard's image as PNG into a malloc'd
// buffer the caller frees, or returns NULL. Screenshots arrive as PNG;
// pictures copied from other apps are often TIFF only, and are converted.
static void* yogaClipboardPNG(size_t* outLen) {
	@autoreleasepool {
		NSPasteboard* pb = [NSPasteboard generalPasteboard];
		NSData* data = [pb dataForType:NSPasteboardTypePNG];
		if (data == nil) {
			NSData* tiff = [pb dataForType:NSPasteboardTypeTIFF];
			if (tiff == nil) {
				return NULL;
			}
			NSBitmapImageRep* rep = [NSBitmapImageRep imageRepWithData:tiff];
			if (rep == nil) {
				return NULL;
			}
			data = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
		}
		if (data == nil || data.length == 0) {
			return NULL;
		}
		void* buf = malloc(data.length);
		if (buf == NULL) {
			return NULL;
		}
		memcpy(buf, data.bytes, data.length);
		*outLen = data.length;
		return buf;
	}
}
*/
import "C"

// clipboardImage reads the pasteboard's image as PNG.
func clipboardImage() ([]byte, bool) {
	var n C.size_t
	p := C.yogaClipboardPNG(&n)
	if p == nil {
		return nil, false
	}
	defer C.free(p)
	return C.GoBytes(p, C.int(n)), true
}
