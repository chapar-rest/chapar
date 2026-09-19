// Copyright 2023 The Knuth Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package knuth // modernc.org/knuth

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"unsafe"
)

var (
	_ File  = (*binaryFile)(nil)
	_ File  = (*textFile)(nil)
	_ File  = (*poolFile)(nil)
	_ error = Error("")
)

// Error is a specific error implementation.
type Error string

func (e Error) Error() string { return string(e) }

// WriteWidth is the type of width in `writeln(foo: width);`.
type WriteWidth int

// File represents a Pascal file.
type File interface {
	ByteP() *byte
	Close()
	CurPos() int32
	Data4P() *[4]byte
	EOF() bool
	EOLN() bool
	ErStat() int32
	Get()
	Put()
	Read(args ...interface{})
	Readln(args ...interface{})
	Reset(args ...interface{})
	Rewrite(args ...interface{})
	SetPos(int32)
	Write(args ...interface{})
	Writeln(args ...interface{})
}

type file struct {
	buf   []byte
	name  string
	r     io.Reader
	r0    io.Reader
	reset func(string) (io.Reader, error)
	w     io.Writer
	w0    io.Writer

	atEOF  bool
	atEOLN bool
	erStat int32
	isText bool
}

func (f *file) ByteP() *byte {
	f.boot()
	return &f.buf[0]
}

func (f *file) Data4P() *[4]byte {
	f.boot()
	return (*[4]byte)(unsafe.Pointer(&f.buf[0]))
}

func (f *file) boot() {
	if len(f.buf) == 0 {
		f.buf = f.buf[:cap(f.buf)]
		f.Get()
	}
}

func (f *file) Put() {
	panic(todo(""))
}

func (f *file) EOLN() bool {
	if !f.isText {
		panic(todo(""))
	}

	f.boot()
	return f.atEOLN || f.atEOF
}

func (f *file) ErStat() int32 {
	return f.erStat
}

func (f *file) Close() {
	if f.r0 != nil {
		if x, ok := f.r0.(io.Closer); ok {
			x.Close()
		}
		f.r0 = nil
	}
	if f.r != nil {
		f.erStat = 0
		if x, ok := f.r.(io.Closer); ok {
			if err := x.Close(); err != nil {
				f.erStat = 1
			}
		}
		f.r = nil
	}
	if f.w0 != nil {
		if x, ok := f.w.(io.Closer); ok {
			x.Close()
		}
		f.w0 = nil
	}
	if f.w != nil {
		f.erStat = 0
		if x, ok := f.w.(io.Closer); ok {
			if err := x.Close(); err != nil {
				f.erStat = 1
			}
		}
		f.w = nil
	}
}

func (f *file) EOF() bool {
	f.boot()
	return f.atEOF
}

func (f *file) Read(args ...interface{}) {
	if f.r == nil && f.r0 != nil {
		f.r = f.r0
		f.r0 = nil
	}
	f.boot()
	switch len(f.buf) {
	case 1:
		for _, v := range args {
			switch x := v.(type) {
			case *byte:
				*x = f.buf[0]
			default:
				panic(todo("%T", x))
			}
			f.Get()
		}
	default:
		panic(todo("", len(f.buf)))
	}
}

func (f *file) Get() {
	if f.atEOF {
		panic(todo(""))
	}

	f.atEOLN = false
	if c, err := f.r.Read(f.buf[:]); c != len(f.buf) {
		if err != io.EOF {
			panic(todo(""))
		}

		f.atEOLN = true
		f.atEOF = true
		return
	}

	if f.isText && f.buf[0] == '\n' {
		f.atEOLN = true
		f.buf[0] = ' '
	}
}

func (f *file) Readln(args ...interface{}) {
	f.Read(args...)
	for !f.EOLN() {
		f.Get()
	}
	f.Get()
}

func (f *file) Reset(args ...interface{}) {
	if debug {
		defer func() {
			trc("RESET %p %q %v -> erStat %v (%v: %v:)", f, f.name, args, f.erStat, origin(3), origin(4))
		}()
	}
	f.atEOF = false
	f.atEOLN = false
	f.buf = f.buf[:0]
	switch len(args) {
	case 0:
		if f.r == nil && f.r0 != nil {
			f.r = f.r0
			f.r0 = nil
			break
		}

		switch x, ok := f.r.(io.Seeker); {
		case ok:
			if _, err := x.Seek(0, io.SeekStart); err != nil {
				panic(todo(""))
			}
		default:
			panic(todo(""))
		}

	case 1:
		switch x := args[0].(type) {
		case string:
			f.open(strings.TrimRight(x, " "))
		default:
			panic(todo("%T", x))
		}
	case 2:
		switch x := args[0].(type) {
		case string:
			switch y := args[1].(type) {
			case string:
				switch {
				case x == "TTY:" && y == "/O/I" && f.r0 != nil:
					f.name = x + y
					f.r = f.r0
					f.r0 = nil
				case y == "/O":
					f.open(strings.TrimRight(x, " "))
				default:
					panic(todo("%q %q %v", x, y, f.w != nil))
				}
			default:
				panic(todo("%T", y))
			}
		default:
			panic(todo("%T", x))
		}
	default:
		panic(todo("", args, len(args)))
	}
}

func (f *file) open(name string) {
	if f.reset != nil {
		if x, ok := f.r.(io.Closer); ok {
			x.Close()
			f.r = nil
		}
		f.name = name
		f.atEOF = false
		f.atEOLN = false
		f.erStat = 0
		var err error
		f.r, err = f.reset(name)
		if err == nil {
			return
		}

		f.r = nil
		f.erStat = 1
		return

	}

	f.name = name
	f.atEOF = false
	f.atEOLN = false
	f.erStat = 0
	var err error
	f.r, err = os.Open(name)
	if err == nil {
		return
	}

	f.r = nil
	f.erStat = 1
}

func (f *file) Rewrite(args ...interface{}) {
	if debug {
		defer func() {
			trc("REWRITE %p %q %v -> erStat %v ()", f, f.name, args, f.erStat)
		}()
	}
	f.atEOF = true
	f.atEOLN = false
	switch len(args) {
	case 0:
		if f.w == nil && f.w0 != nil {
			f.w = f.w0
			f.w0 = nil
			break
		}

		panic(todo(""))
	case 2:
		switch x := args[0].(type) {
		case string:
			switch y := args[1].(type) {
			case string:
				switch {
				case x == "TTY:" && y == "/O" && f.w0 != nil:
					f.name = x + y
					f.w = f.w0
					f.w0 = nil
				case y == "/O":
					f.erStat = 0
					if f.w != nil {
						panic(todo(""))
					}

					var err error
					f.name = strings.TrimRight(x, " ")
					if f.w, err = os.Create(f.name); err != nil {
						f.w = nil
						f.erStat = 1
						break
					}

				default:
					panic(todo("%q %q", x, y))
				}
			default:
				panic(todo("%T", y))
			}
		default:
			panic(todo("%T", x))
		}
	default:
		panic(todo("", args, len(args)))
	}
}

func (f *file) CurPos() int32 {
	switch {
	case f.r != nil:
		s, ok := f.r.(io.ReadSeeker)
		if !ok {
			panic(todo(""))
		}

		n, err := s.Seek(0, io.SeekCurrent)
		if err != nil || n > math.MaxInt32 {
			panic(todo(""))
		}

		return int32(n)
	case f.w != nil:
		panic(todo(""))
	default:
		panic(todo(""))
	}
}

func (f *file) SetPos(n int32) {
	switch {
	case f.r != nil:
		s, ok := f.r.(io.ReadSeeker)
		if !ok {
			panic(todo(""))
		}

		switch {
		case n < 0:
			if _, err := s.Seek(0, io.SeekEnd); err != nil {
				panic(todo(""))
			}

			f.atEOF = true
		default:
			if _, err := s.Seek(int64(n), io.SeekStart); err != nil {
				panic(todo(""))
			}

			f.atEOF = false
			f.atEOLN = false
			f.Get()
		}
	case f.w != nil:
		panic(todo(""))
	default:
		panic(todo(""))
	}
}

type textFile struct {
	*file
}

// NewTextFile returns File suitable for Pascal file type 'text'.
func NewTextFile(r io.Reader, w io.Writer, open func(string) (io.Reader, error)) File {
	return &textFile{
		&file{
			buf:    make([]byte, 1),
			isText: true,
			r0:     r,
			reset:  open,
			w0:     w,
		},
	}
}

func (f *textFile) Write(args ...interface{}) {
	if f.w == nil && f.w0 != nil {
		f.w = f.w0
		f.w0 = nil
	}
	var a [][]interface{}
	for i := 0; i < len(args); i++ {
		switch x := args[i].(type) {
		case WriteWidth:
			a[len(a)-1] = append(a[len(a)-1], int(x))
		default:
			a = append(a, []interface{}{x})
		}
	}
	for _, v := range a {
		switch x := v[0].(type) {
		case string:
			switch len(v) {
			case 1:
				if _, err := fmt.Fprintf(f.w, "%s", x); err != nil {
					panic(todo("", err))
				}
			case 2:
				if _, err := fmt.Fprintf(f.w, "%*s", v[1], v[0]); err != nil {
					panic(todo("", err))
				}
			default:
				panic(todo("", v))
			}
		case uint8:
			switch len(v) {
			case 1:
				if _, err := fmt.Fprintf(f.w, "%d", v[0]); err != nil {
					panic(todo("", err))
				}
			case 2:
				if _, err := fmt.Fprintf(f.w, "%*d", v[1], v[0]); err != nil {
					panic(todo("", err))
				}
			default:
				panic(todo("", v))
			}
		case uint16:
			switch len(v) {
			case 1:
				if _, err := fmt.Fprintf(f.w, "%d", v[0]); err != nil {
					panic(todo("", err))
				}
			case 2:
				if _, err := fmt.Fprintf(f.w, "%*d", v[1], v[0]); err != nil {
					panic(todo("", err))
				}
			default:
				panic(todo("", v))
			}
		case int32, int:
			switch len(v) {
			case 1:
				if _, err := fmt.Fprintf(f.w, "%d", v[0]); err != nil {
					panic(todo("", err))
				}
			case 2:
				if _, err := fmt.Fprintf(f.w, "%*d", v[1], v[0]); err != nil {
					panic(todo("", err))
				}
			default:
				panic(todo("", v))
			}
		case float32:
			switch len(v) {
			case 1:
				if _, err := fmt.Fprintf(f.w, "%g", v[0]); err != nil {
					panic(todo("", err))
				}
			case 2:
				if _, err := fmt.Fprintf(f.w, "%*e", v[1], v[0]); err != nil {
					panic(todo("", err))
				}
			case 3:
				mw := v[1].(int)
				nw := v[2].(int)
				if _, err := fmt.Fprintf(f.w, "%*.*f", mw, nw, v[0]); err != nil {
					panic(todo("", err))
				}
			default:
				panic(todo("", v))
			}
		case float64:
			switch len(v) {
			case 1:
				if _, err := fmt.Fprintf(f.w, "%g", v[0]); err != nil {
					panic(todo("", err))
				}
			case 2:
				if _, err := fmt.Fprintf(f.w, "%*e", v[1], v[0]); err != nil {
					panic(todo("", err))
				}
			case 3:
				mw := v[1].(int)
				nw := v[2].(int)
				if _, err := fmt.Fprintf(f.w, "%*.*f", mw, nw, v[0]); err != nil {
					panic(todo("", err))
				}
			default:
				panic(todo("", v))
			}
		default:
			panic(todo("%T %v", x, v))
		}
	}
}

func (f *textFile) Writeln(args ...interface{}) {
	f.Write(args...)
	if _, err := fmt.Fprintln(f.w); err != nil {
		panic(todo("", err))
	}
}

type binaryFile struct {
	*file
}

// NewBinaryFile returns a File suitable for Pascal file type 'file of T'.
func NewBinaryFile(r io.Reader, w io.Writer, sizeofT int, open func(string) (io.Reader, error)) File {
	return &binaryFile{
		&file{
			buf:   make([]byte, sizeofT),
			r0:    r,
			reset: open,
			w0:    w,
		},
	}
}

func (f *binaryFile) Write(args ...interface{}) {
	switch len(f.buf) {
	case 1:
		for _, v := range args {
			switch x := v.(type) {
			case int32:
				f.buf[0] = byte(x)
			case int:
				f.buf[0] = byte(x)
			case byte:
				f.buf[0] = x
			case int16:
				f.buf[0] = byte(x)
			case uint16:
				f.buf[0] = byte(x)
			default:
				panic(todo("%T", x))
			}
			f.Put()
		}
	default:
		panic(todo("", len(f.buf)))
	}
}

func (f *binaryFile) Put() {
	if c, err := f.w.Write(f.buf[:]); c != len(f.buf) {
		panic(todo("", err))
	}
}

func (f *binaryFile) Writeln(args ...interface{}) {
	panic(todo(""))
}

type poolFile struct {
	*file
}

// NewPoolFile returns a read only File with a string pool.
func NewPoolFile(pool string) File {
	return &poolFile{
		&file{
			buf:    make([]byte, 1),
			isText: true,
			r:      strings.NewReader(pool),
		},
	}
}

func (f *poolFile) Close() {
	f.atEOF = true
}

func (f *poolFile) ErStat() int32 {
	return 0
}

func (f *poolFile) Reset(args ...interface{}) {
	f.atEOF = false
	f.atEOLN = false
	switch len(args) {
	case 2: // eg. ["MFbases:MF.POOL", "/O"]
		f.r.(*strings.Reader).Seek(0, io.SeekStart)
		f.Get()
	default:
		panic(todo("%v", args))
	}
}

func (f *poolFile) Write(args ...interface{}) {
	panic(todo(""))
}

func (f *poolFile) Writeln(args ...interface{}) {
	panic(todo(""))
}
