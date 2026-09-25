// Copyright 2023 The Knuth Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate make generate

// Package tex is the TEX program by D. E. Knuth, transpiled to Go.
//
//	http://mirrors.ctan.org/systems/knuth/dist/tex/tex.web
package tex // modernc.org/knuth/tex

import (
	// Required by go:embed
	_ "embed"
	"fmt"
	"io"
	"runtime/debug"
	"unsafe"

	"modernc.org/knuth"
)

//go:embed tex.pool
var pool string

// program TEX; {all file names are defined dynamically}

// Main executes the tex program using the supplied arguments.
func Main(stdin io.Reader, stdout, stderr io.Writer, options ...Option) (mainErr error) {
	defer func() {
		switch x := recover().(type) {
		case nil:
			// ok
		case signal:
			switch {
			case mainErr == nil:
				mainErr = fmt.Errorf("aborted")
			default:
				mainErr = fmt.Errorf("aborted: %v", mainErr)
			}
		case knuth.Error:
			mainErr = x
		default:
			mainErr = fmt.Errorf("PANIC %T: %[1]v, error: %v\n%s", x, mainErr, debug.Stack())
		}
	}()

	// Options are applied to a per-call config, never to package scope, so that
	// concurrent Main calls do not interfere and a WithInputFile from one call
	// cannot leak into the next. Every file is built afterwards, so all of them
	// see the same opener.
	cfg := &config{opener: defaultOpener}
	for _, v := range options {
		if err := v(cfg); err != nil {
			return err
		}
	}
	if cfg.dviFile == nil {
		cfg.dviFile = knuth.NewBinaryFile(nil, nil, 1, nil)
	}
	if cfg.logFile == nil {
		cfg.logFile = knuth.NewTextFile(nil, nil, nil)
	}

	prg := &prg{
		dviFile:  cfg.dviFile,
		fmtFile:  knuth.NewBinaryFile(nil, nil, int(unsafe.Sizeof(memoryWord{})), cfg.opener),
		logFile:  cfg.logFile,
		poolFile: knuth.NewPoolFile(pool),
		stderr:   knuth.NewTextFile(nil, stderr, nil),
		termIn:   knuth.NewTextFile(stdin, nil, nil),
		termOut:  knuth.NewTextFile(nil, stdout, nil),
		tfmFile:  knuth.NewBinaryFile(nil, nil, 1, cfg.opener),
	}
	for i := range prg.inputFile {
		prg.inputFile[i] = knuth.NewTextFile(nil, nil, cfg.opener)
	}
	for i := range prg.writeFile {
		prg.writeFile[i] = knuth.NewTextFile(nil, nil, nil)
	}
	for i := range prg.readFile {
		prg.readFile[i] = knuth.NewTextFile(nil, nil, nil)
	}
	prg.main()
	return nil
}

func defaultOpener(nm string) (io.Reader, error) {
	return knuth.Open(nm, []string{"."})
}

// config collects what the options set. It is per Main call: Main is safe to
// use concurrently and leaves no state behind.
type config struct {
	dviFile knuth.File
	logFile knuth.File
	opener  func(string) (io.Reader, error)
}

// Option adjusts program behavior.
type Option func(c *config) error

// WithInputFile replaces input file 'replace' with 'r'.
func WithInputFile(replace string, r io.Reader) Option {
	return func(c *config) error {
		prev := c.opener
		c.opener = func(nm string) (io.Reader, error) {
			if nm == replace {
				return r, nil
			}

			return prev(nm)
		}
		return nil
	}
}

// WithDVIFile sets the output DVI file to 'w'.
func WithDVIFile(w io.Writer) Option {
	return func(c *config) error {
		c.dviFile = &binaryWriter{w: w}
		return nil
	}
}

// WithLogFile sets the output log file to 'w'.
func WithLogFile(w io.Writer) Option {
	return func(c *config) error {
		c.logFile = &textWriter{w: w}
		return nil
	}
}

type binaryWriter struct {
	buf [1]byte
	w   io.Writer
}

func (w *binaryWriter) ByteP() *byte {
	panic("internal error")
}

func (w *binaryWriter) Close() {
	// nop
}

func (w *binaryWriter) CurPos() int32 {
	panic("internal error")
}

func (w *binaryWriter) Data4P() *[4]byte {
	panic("internal error")
}

func (w *binaryWriter) EOF() bool {
	panic("internal error")
}

func (w *binaryWriter) EOLN() bool {
	panic("internal error")
}

func (w *binaryWriter) ErStat() int32 {
	return 0
}

func (w *binaryWriter) Get() {
	panic("internal error")
}

func (w *binaryWriter) Put() {
	panic("internal error")
}

func (w *binaryWriter) Read(args ...interface{}) {
	panic("internal error")
}

func (w *binaryWriter) Readln(args ...interface{}) {
	panic("internal error")
}

func (w *binaryWriter) Reset(args ...interface{}) {
	panic("internal error")
}

func (w *binaryWriter) Rewrite(args ...interface{}) {
	// nop
}

func (w *binaryWriter) SetPos(int32) {
	panic("internal error")
}

func (w *binaryWriter) Write(args ...interface{}) {
	for _, v := range args {
		w.buf[0] = v.(byte)
		w.w.Write(w.buf[:])
	}
}

func (w *binaryWriter) Writeln(args ...interface{}) {
	panic("internal error")
}

type textWriter struct {
	w io.Writer
}

func (w *textWriter) ByteP() *byte {
	panic("internal error")
}

func (w *textWriter) Close() {
	// nop
}

func (w *textWriter) CurPos() int32 {
	panic("internal error")
}

func (w *textWriter) Data4P() *[4]byte {
	panic("internal error")
}

func (w *textWriter) EOF() bool {
	panic("internal error")
}

func (w *textWriter) EOLN() bool {
	panic("internal error")
}

func (w *textWriter) ErStat() int32 {
	return 0
}

func (w *textWriter) Get() {
	panic("internal error")
}

func (w *textWriter) Put() {
	panic("internal error")
}

func (w *textWriter) Read(args ...interface{}) {
	panic("internal error")
}

func (w *textWriter) Readln(args ...interface{}) {
	panic("internal error")
}

func (w *textWriter) Reset(args ...interface{}) {
	panic("internal error")
}

func (w *textWriter) Rewrite(args ...interface{}) {
	// nop
}

func (w *textWriter) SetPos(int32) {
	panic("internal error")
}

func (w *textWriter) Write(args ...interface{}) {
	for _, v := range args {
		fmt.Fprintf(w.w, "%v", v)
	}
}

func (w *textWriter) Writeln(args ...interface{}) {
	for _, v := range args {
		fmt.Fprintf(w.w, "%v", v)
	}
	fmt.Fprintln(w.w)
}
