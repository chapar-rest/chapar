// SPDX-License-Identifier: Unlicense OR MIT

package explorer

import (
	"io"
	"strings"
	"syscall/js"

	"gioui.org/app"
	"gioui.org/io/event"
)

type explorer struct{}

func newExplorer(_ *app.Window) *explorer {
	return &explorer{}
}

func (e *Explorer) listenEvents(_ event.Event) {
	// NO-OP
}

func (e *Explorer) exportFile(name string) (io.WriteCloser, error) {
	return newFileWriter(name), nil
}

func (e *Explorer) importFile(extensions ...string) (io.ReadCloser, error) {
	// TODO: Replace with "File System Access API" when that becomes available on most browsers.
	// BUG: Not work on iOS/Safari.

	// It's not possible to know if the user closes the file-picker dialog, so an new channel is needed.
	r := make(chan result)

	document := js.Global().Get("document")
	input := document.Call("createElement", "input")
	input.Call("addEventListener", "change", openCallback(r))
	input.Call("addEventListener", "cancel", openCallback(r))
	input.Set("type", "file")
	input.Set("style", "display:none;")
	if len(extensions) > 0 {
		input.Set("accept", strings.Join(extensions, ","))
	}
	document.Get("body").Call("appendChild", input)
	input.Call("click")

	file := <-r
	if file.error != nil {
		return nil, file.error
	}
	return file.file.(io.ReadCloser), nil
}

func (e *Explorer) importFiles(_ ...string) ([]io.ReadCloser, error) {
	return nil, ErrNotAvailable
}

type FileReader struct {
	buffer                   js.Value
	isClosed                 bool
	index                    int
	callback                 chan js.Value
	successFunc, failureFunc js.Func
}

func newFileReader(v js.Value) *FileReader {
	f := &FileReader{
		buffer:   v,
		callback: make(chan js.Value, 1),
	}
	f.successFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		f.callback <- args[0]
		return nil
	})
	f.failureFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		f.callback <- js.Undefined()
		return nil
	})

	return f
}

func (f *FileReader) Read(b []byte) (n int, err error) {
	if f == nil || f.isClosed {
		return 0, io.ErrClosedPipe
	}

	go func() {
		fileSlice(f.index, f.index+len(b), f.buffer, f.successFunc, f.failureFunc)
	}()

	buffer := <-f.callback
	if !buffer.Truthy() {
		return 0, io.ErrUnexpectedEOF
	}

	n = fileRead(buffer, b)
	if n == 0 {
		return 0, io.EOF
	}
	f.index += n

	return n, err
}

func (f *FileReader) Close() error {
	if f == nil || f.isClosed {
		return io.ErrClosedPipe
	}

	f.failureFunc.Release()
	f.successFunc.Release()
	f.isClosed = true
	return nil
}

type FileWriter struct {
	buffers                  []js.Value
	isClosed                 bool
	name                     string
	successFunc, failureFunc js.Func
}

func newFileWriter(name string) *FileWriter {
	return &FileWriter{
		name: name,
	}
}

func (f *FileWriter) Write(b []byte) (n int, err error) {
	if f == nil || f.isClosed {
		return 0, io.ErrClosedPipe
	}
	if len(b) == 0 {
		return 0, nil
	}

	buff := js.Global().Get("Uint8Array").New(len(b))
	fileWrite(buff, b)
	f.buffers = append(f.buffers, buff)
	return len(b), err
}

func (f *FileWriter) Close() error {
	if f == nil || f.isClosed {
		return io.ErrClosedPipe
	}
	f.isClosed = true
	return f.saveFile()
}

func (f *FileWriter) saveFile() error {
	config := js.Global().Get("Object").New()
	config.Set("type", "octet/stream")

	buffs := js.Global().Get("Array").New(len(f.buffers))
	for idx, buf := range f.buffers {
		buffs.SetIndex(idx, js.ValueOf(buf))
	}
	blob := js.Global().Get("Blob").New(
		buffs,
		config,
	)

	document := js.Global().Get("document")
	anchor := document.Call("createElement", "a")
	anchor.Set("download", f.name)
	anchor.Set("href", js.Global().Get("URL").Call("createObjectURL", blob))
	document.Get("body").Call("appendChild", anchor)
	anchor.Call("click")

	return nil
}

func fileRead(value js.Value, b []byte) int {
	return js.CopyBytesToGo(b, js.Global().Get("Uint8Array").New(value))
}

func fileWrite(value js.Value, b []byte) int {
	return js.CopyBytesToJS(value, b)
}

func fileSlice(start, end int, value js.Value, success, failure js.Func) {
	value.Call("slice", start, end).Call("arrayBuffer").Call("then", success, failure)
}

func openCallback(r chan result) js.Func {
	// There's no way to detect when the dialog is closed, so we can't re-use the callback.
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		files := args[0].Get("target").Get("files")
		if files.Length() <= 0 {
			r <- result{error: ErrUserDecline}
			return nil
		}
		r <- result{file: newFileReader(files.Index(0))}
		return nil
	})
}

var (
	_ io.ReadCloser  = (*FileReader)(nil)
	_ io.WriteCloser = (*FileWriter)(nil)
)
