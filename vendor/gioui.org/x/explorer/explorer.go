// SPDX-License-Identifier: Unlicense OR MIT

package explorer

import (
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/io/event"
)

var (
	// ErrUserDecline is returned when the user doesn't select the file.
	ErrUserDecline = errors.New("user exited the file selector without selecting a file")

	// ErrNotAvailable is return when the current OS isn't supported.
	ErrNotAvailable = errors.New("current OS not supported")
)

type result struct {
	file  interface{}
	error error
}

// Explorer facilitates opening OS-native dialogs to choose files and create files.
type Explorer struct {
	id    int32
	mutex sync.Mutex

	// explorer holds OS-Specific content, it varies for each OS.
	*explorer
}

// active holds all explorer currently active, that may necessary for callback functions.
//
// Some OSes (Android, iOS, macOS) may call Golang exported functions as callback, but we need
// someway to link that callback with the respective explorer, in order to give them a response.
//
// In that case, a construction like `callback(..., id int32)` is used. Then, it's possible to get the explorer
// by lookup the active using the callback id.
//
// To avoid hold dead/unnecessary explorer, the active will be removed using `runtime.SetFinalizer` on the related
// Explorer.
var (
	active  = sync.Map{} // map[int32]*explorer
	counter = new(int32)
)

// NewExplorer creates a new Explorer for the given *app.Window.
// The given app.Window must be unique and you should call NewExplorer
// once per new app.Window.
//
// It's mandatory to use Explorer.ListenEvents on the same *app.Window.
func NewExplorer(w *app.Window) (e *Explorer) {
	e = &Explorer{
		explorer: newExplorer(w),
		id:       atomic.AddInt32(counter, 1),
	}

	active.Store(e.id, e.explorer)
	runtime.SetFinalizer(e, func(e *Explorer) { active.Delete(e.id) })

	return e
}

// ListenEvents must get all the events from Gio, in order to get the GioView. You must
// include that function where you listen for Gio events.
//
// Similar as:
//
//	select {
//	case e := <-window.Events():
//
//		explorer.ListenEvents(e)
//		switch e := e.(type) {
//			(( ... your code ...  ))
//		}
//	}
func (e *Explorer) ListenEvents(evt event.Event) {
	if e == nil {
		return
	}
	e.listenEvents(evt)
}

// ChooseFile shows the file selector, allowing the user to select a single file.
// Optionally, it's possible to define which file extensions is supported to
// be selected (such as `.jpg`, `.png`).
//
// Example: ChooseFile(".jpg", ".png") will only accept the selection of files with
// .jpg or .png extensions.
//
// In some platforms the resulting `io.ReadCloser` is a `os.File`, but it's not
// a guarantee.
//
// In most known browsers, when user clicks cancel then this function never returns.
//
// It's a blocking call, you should call it on a separated goroutine. For most OSes, only one
// ChooseFile or CreateFile, can happen at the same time, for each app.Window/Explorer.
func (e *Explorer) ChooseFile(extensions ...string) (io.ReadCloser, error) {
	if e == nil {
		return nil, ErrNotAvailable
	}

	if runtime.GOOS != "js" {
		e.mutex.Lock()
		defer e.mutex.Unlock()
	}

	return e.importFile(extensions...)
}

// ChooseFiles shows the files selector, allowing the user to select multiple files.
// Optionally, it's possible to define which file extensions is supported to
// be selected (such as `.jpg`, `.png`).
//
// Example: ChooseFiles(".jpg", ".png") will only accept the selection of files with
// .jpg or .png extensions.
//
// In some platforms the resulting `io.ReadCloser` is a `os.File`, but it's not
// a guarantee.
//
// In most known browsers, when user clicks cancel then this function never returns.
//
// It's a blocking call, you should call it on a separated goroutine. For most OSes, only one
// ChooseFile{,s} or CreateFile, can happen at the same time, for each app.Window/Explorer.
func (e *Explorer) ChooseFiles(extensions ...string) ([]io.ReadCloser, error) {
	if e == nil {
		return nil, ErrNotAvailable
	}

	if runtime.GOOS != "js" {
		e.mutex.Lock()
		defer e.mutex.Unlock()
	}

	return e.importFiles(extensions...)
}

// CreateFile opens the file selector, and writes the given content into
// some file, which the use can choose the location.
//
// It's important to close the `io.WriteCloser`. In some platforms the
// file will be saved only when the writer is closer.
//
// In some platforms the resulting `io.WriteCloser` is a `os.File`, but it's not
// a guarantee.
//
// It's a blocking call, you should call it on a separated goroutine. For most OSes, only one
// ChooseFile or CreateFile, can happen at the same time, for each app.Window/Explorer.
func (e *Explorer) CreateFile(name string) (io.WriteCloser, error) {
	if e == nil {
		return nil, ErrNotAvailable
	}

	if runtime.GOOS != "js" {
		e.mutex.Lock()
		defer e.mutex.Unlock()
	}

	return e.exportFile(name)
}

var (
	DefaultExplorer *Explorer
)

// ListenEventsWindow calls Explorer.ListenEvents on DefaultExplorer,
// and creates a new Explorer, if needed.
//
// Deprecated: Use NewExplorer instead.
func ListenEventsWindow(win *app.Window, event event.Event) {
	if DefaultExplorer == nil {
		DefaultExplorer = NewExplorer(win)
	}
	DefaultExplorer.ListenEvents(event)
}

// ReadFile calls Explorer.ChooseFile on DefaultExplorer.
//
// Deprecated: Use NewExplorer and Explorer.ChooseFile instead.
func ReadFile(extensions ...string) (io.ReadCloser, error) {
	return DefaultExplorer.ChooseFile(extensions...)
}

// WriteFile calls Explorer.CreateFile on DefaultExplorer.
//
// Deprecated: Use NewExplorer and Explorer.CreateFile instead.
func WriteFile(name string) (io.WriteCloser, error) {
	return DefaultExplorer.CreateFile(name)
}
