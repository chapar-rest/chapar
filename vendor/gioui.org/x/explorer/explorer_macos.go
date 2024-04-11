// SPDX-License-Identifier: Unlicense OR MIT

//go:build darwin && !ios
// +build darwin,!ios

package explorer

/*
#cgo CFLAGS: -Werror -xobjective-c -fmodules -fobjc-arc

#import <Appkit/AppKit.h>

// Defined on explorer_macos.m file.
extern void exportFile(CFTypeRef viewRef, char * name, int32_t id);
extern void importFile(CFTypeRef viewRef, char * ext, int32_t id);
*/
import "C"
import (
	"io"
	"net/url"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/io/event"
)

type explorer struct {
	window *app.Window
	view   C.CFTypeRef
	result chan result
}

func newExplorer(w *app.Window) *explorer {
	return &explorer{window: w, result: make(chan result)}
}

func (e *Explorer) listenEvents(evt event.Event) {
	switch evt := evt.(type) {
	case app.AppKitViewEvent:
		e.view = C.CFTypeRef(evt.View)
	}
}

func (e *Explorer) exportFile(name string) (io.WriteCloser, error) {
	cname := C.CString(name)
	e.window.Run(func() { C.exportFile(e.view, cname, C.int32_t(e.id)) })

	resp := <-e.result
	if resp.error != nil {
		return nil, resp.error
	}
	return resp.file.(io.WriteCloser), resp.error

}

func (e *Explorer) importFile(extensions ...string) (io.ReadCloser, error) {
	for i, ext := range extensions {
		extensions[i] = strings.TrimPrefix(ext, ".")
	}

	cextensions := C.CString(strings.Join(extensions, ","))
	e.window.Run(func() { C.importFile(e.view, cextensions, C.int32_t(e.id)) })

	resp := <-e.result
	if resp.error != nil {
		return nil, resp.error
	}
	return resp.file.(io.ReadCloser), resp.error
}

func (e *Explorer) importFiles(_ ...string) ([]io.ReadCloser, error) {
	return nil, ErrNotAvailable
}

//export importCallback
func importCallback(u *C.char, id int32) {
	if v, ok := active.Load(id); ok {
		v.(*explorer).result <- newOSFile(u, os.Open)
	}
}

//export exportCallback
func exportCallback(u *C.char, id int32) {
	if v, ok := active.Load(id); ok {
		v.(*explorer).result <- newOSFile(u, os.Create)
	}
}

func newOSFile(u *C.char, action func(s string) (*os.File, error)) result {
	name := C.GoString(u)
	if name == "" {
		return result{error: ErrUserDecline, file: nil}
	}

	uri, err := url.Parse(name)
	if err != nil {
		return result{error: err, file: nil}
	}

	path, err := url.PathUnescape(uri.Path)
	if err != nil {
		return result{error: err, file: nil}
	}

	f, err := action(path)
	return result{error: err, file: f}
}
