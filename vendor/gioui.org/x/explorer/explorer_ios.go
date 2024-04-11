// SPDX-License-Identifier: Unlicense OR MIT

//go:build ios
// +build ios

package explorer

/*
#cgo CFLAGS: -Werror -xobjective-c -fmodules -fobjc-arc

#include <UIKit/UIKit.h>
#include <stdint.h>

// Defined on explorer_ios.m file (implements UIDocumentPickerDelegate).
@interface explorer_picker:NSObject<UIDocumentPickerDelegate>
@property (strong) UIDocumentPickerViewController * picker;
@property (strong) UIViewController * controller;
@property uint64_t mode;
@property uint32_t id;
@end

static const uint64_t IMPORT_MODE = 1;
static const uint64_t EXPORT_MODE = 2;

extern CFTypeRef createPicker(CFTypeRef controllerRef, int32_t id);
extern bool exportFile(CFTypeRef expl, char * name);
extern bool importFile(CFTypeRef expl, char * ext);
*/
import "C"
import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/io/event"
)

type explorer struct {
	window *app.Window
	picker C.CFTypeRef
	result chan result
}

func newExplorer(w *app.Window) *explorer {
	return &explorer{window: w, result: make(chan result)}
}

func (e *Explorer) listenEvents(evt event.Event) {
	switch evt := evt.(type) {
	case app.UIKitViewEvent:
		e.explorer.picker = C.createPicker(C.CFTypeRef(evt.ViewController), C.int32_t(e.id))
	}
}

func (e *Explorer) exportFile(name string) (io.WriteCloser, error) {
	name = filepath.Join(os.TempDir(), name)

	f, err := os.Create(name)
	if err != nil {
		return nil, nil
	}
	f.Close()

	name = "file://" + name

	go e.window.Run(func() {
		if ok := bool(C.exportFile(e.explorer.picker, C.CString(name))); !ok {
			e.result <- result{error: ErrNotAvailable}
		}
	})

	file := <-e.result
	if file.error != nil {
		return nil, file.error
	}
	return file.file.(io.WriteCloser), nil
}

func (e *Explorer) importFile(extensions ...string) (io.ReadCloser, error) {
	for i, ext := range extensions {
		extensions[i] = strings.TrimPrefix(ext, ".")
	}

	cextensions := C.CString(strings.Join(extensions, ","))
	go e.window.Run(func() {
		if ok := bool(C.importFile(e.explorer.picker, cextensions)); !ok {
			e.result <- result{error: ErrNotAvailable}
		}
	})

	file := <-e.result
	if file.error != nil {
		return nil, file.error
	}
	return file.file.(io.ReadCloser), nil
}

func (e *Explorer) importFiles(_ ...string) ([]io.ReadCloser, error) {
	return nil, ErrNotAvailable
}

//export importCallback
func importCallback(u C.CFTypeRef, id C.int32_t) {
	fileCallback(u, id)
}

//export exportCallback
func exportCallback(u C.CFTypeRef, id C.int32_t) {
	fileCallback(u, id)
}

func fileCallback(u C.CFTypeRef, id C.int32_t) {
	var res result
	if v, ok := active.Load(int32(id)); ok {
		if u == 0 {
			res.error = ErrUserDecline
		} else {
			res.file, res.error = newFile(u)
		}
		v.(*explorer).result <- res
	}
}
