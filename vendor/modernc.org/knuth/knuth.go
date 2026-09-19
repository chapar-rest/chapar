// Copyright 2023 The Knuth Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package knuth collects utilities common to all other packages in this
// repository.
//
// To install all the included go* commands found in cmd/
//
//	$ go install modernc.org/knuth/cmd...@latest
//
// Documentation
//
//	http://godoc.org/modernc.org/knuth
//
// # Hacking
//
// Make sure you have these utilities from the Tex-live package(s) installed in
// your $PATH:
//
//	dvitype
//	gftopk
//	gftype
//	mf
//	mft
//	pooltype
//	tangle
//	tex
//	tftopl
//	vftovp
//	vptovf
//	weave
//
// These programs are used only to generate test data. Users of
// packages/commands in this repository do not need them installed.
//
// After modification of any sources, run '$ make' in the repository root. That
// will regenerate all applicable Go code and testdata, run tests of all
// packages in this repository and install all the commands found in ./cmd.
//
// If your local clone of the repository is private, you need to setup the
// GOPRIVATE environment variable properly for the tests to pass.
package knuth // modernc.org/knuth

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	mtoken "modernc.org/token"
)

var (
	// ASCII is a RuneValidator accepting runes '\x00' ... '\xff`.
	ASCII asciiValidator
	// Unicode is a RuneValidator accepting all valid unicode code points
	// except those in category Co and Cs.
	Unicode unicodeValidator

	//go:embed assets.tar.gz
	assets embed.FS

	// Assets provides some essential resources:
	//
	//  fonts/cm/mf/* from https://ctan.org/tex-archive/systems/knuth/dist/cm
	//  lib/* from https://ctan.org/tex-archive/systems/knuth/dist/lib
	Assets fs.FS = newCFS(assets)

	modTime = time.Now()

	_ RuneSource    = (*Changer)(nil)
	_ RuneSource    = (*runeSource)(nil)
	_ RuneValidator = asciiValidator{}
	_ RuneValidator = unicodeValidator{}
	_ fs.FS         = (*cfs)(nil)
	_ fs.File       = (*fsFile)(nil)
	_ fs.FileInfo   = (*fsFile)(nil)
	_ io.Seeker     = (*fsFile)(nil)
)

// RuneValidator validates runes.
type RuneValidator interface {
	// Validate returns true if its argument is in the accepted rune set.
	Validate(rune) bool
}

// asciiValidator is a RuneValidator accepting runes '\x00' ... '\xff`.
type asciiValidator struct{}

// Validate implements RuneValidator
func (asciiValidator) Validate(r rune) bool { return r >= 0 && r <= 0xff }

// unicodeValidator is a RuneValidator accepting all valid unicode code points
// except those in category Co and Cs.
type unicodeValidator struct{}

// Validate implements RuneValidator
func (unicodeValidator) Validate(r rune) bool {
	if r >= 0 && r <= 255 {
		return true
	}

	return r >= 0 && r <= unicode.MaxRune && !unicode.Is(unicode.Co, r) || !unicode.Is(unicode.Cs, r)
}

// RuneSource provides a finite stream of runes.
type RuneSource interface {
	// AddLineColumnInfo adds alternative file, line, and column number information
	// for a given file offset. The offset must be larger than the offset for the
	// previously added alternative line info and smaller than the file size;
	// otherwise the information is ignored.
	//
	// AddLineColumnInfo is typically used to register alternative position
	// information for line directives such as //line filename:line:column.
	AddLineColumnInfo(offset int, filename string, line, column int)
	// C returns the current rune or and error, if any.
	C() (rune, error)
	// Consume moves to the next rune, if any.
	Consume()
	// Position returns the current position.
	Position() token.Position
	// PositionFor returns the position for zero based offset.
	PositionFor(off int) token.Position
}

type runeSource struct {
	file      *mtoken.File
	name      string
	validator RuneValidator
	src       []byte

	off int
}

// NewRuneSource returns a newly created Source. Positions will be reported as
// coming from 'name'. 'src' is UTF-8 encoded. Decoded runes will be validated
// by 'validator'.
func NewRuneSource(name string, src []byte, validator RuneValidator) RuneSource {
	return &runeSource{
		file:      mtoken.NewFile(name, len(src)),
		name:      name,
		src:       src,
		validator: validator,
	}
}

// AddLineColumnInfo implements RuneSource.
func (s *runeSource) AddLineColumnInfo(offset int, filename string, line, column int) {
	s.file.AddLineColumnInfo(offset, filename, line, column)
}

func (s *runeSource) C() (rune, error) {
	r, sz := utf8.DecodeRune(s.src[s.off:])
	if r == utf8.RuneError {
		if sz == 0 {
			return 0, io.EOF
		}

		return 0, fmt.Errorf("%v: invalid rune", s.Position())
	}

	if !s.validator.Validate(r) {
		return 0, fmt.Errorf("%v: invalid rune", s.Position())
	}

	return r, nil
}

func (s *runeSource) Position() token.Position {
	return s.PositionFor(s.off)
}

func (s *runeSource) PositionFor(off int) token.Position {
	return token.Position(s.file.PositionFor(mtoken.Pos(s.file.Base()+off), true))
}

// Consume moves s past the current rune to the next one, if any.
func (s *runeSource) Consume() {
	r, sz := utf8.DecodeRune(s.src[s.off:])
	if r == '\n' {
		s.file.AddLine(s.file.Base() + s.off)
	}
	s.off += sz
}

// Line represents a source line and its position.
type Line struct {
	Position token.Position
	Src      string
}

// ReadLine reads from s up to and including the next newline, if any. If s is
// at EOF, (nil, io.EOF) is returned.
func ReadLine(s RuneSource) (line *Line, err error) {
	var a []rune

	pos := s.Position()
	for {
		c, err := s.C()
		if err != nil {
			if err == io.EOF {
				if len(a) != 0 {
					return &Line{pos, string(a)}, nil
				}

				return &Line{pos, string(a)}, err
			}

			return nil, fmt.Errorf("%v: invalid rune", s.Position())
		}

		a = append(a, c)
		s.Consume()
		if c == '\n' {
			return &Line{pos, string(a)}, nil
		}
	}
}

// ReadLine2 is like ReadLine but additionally trims trailing space. The final
// '\n' is preserved, if any.
func ReadLine2(s RuneSource) (line *Line, err error) {
	line, err = ReadLine(s)
	if line != nil {
		line.Src = rtrimLine(line.Src)
	}
	return line, err
}

func rtrimLine(s string) string {
	var nl string
	if strings.HasSuffix(s, "\n") {
		nl = "\n"
		s = s[:len(s)-1]
	}
	return strings.TrimRight(s, " \t") + nl
}

type changerSegment struct {
	src RuneSource
	b   []byte

	off0 int
	off  int
}

// Changer is a RuneSource implementing patching source using a change file.
type Changer struct {
	a  []*changerSegment
	ix int
}

// NewChanger returns a newly created Changer or an error, if any.
func NewChanger(src, changes RuneSource) (*Changer, error) {
	var bb bytes.Buffer
out:
	for {
		line, err := ReadLine2(src)
		if err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("%v: reading source file: %v", line.Position, err)
			}
			break out
		}

		bb.WriteString(line.Src)
	}

	b := bb.Bytes()
	r := &Changer{}

	const (
		zero = iota
		stateX
		stateY
	)

	state := zero
	var orig, repl []byte
	var soff, coff int
	for {
		line, err := ReadLine2(changes)
		if err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("%v: reading change file: %v", line.Position, err)
			}

			if state != zero {
				return nil, fmt.Errorf("%v: reading change file: unexpected EOF", line.Position)
			}

			if len(b) != 0 {
				r.a = append(r.a, &changerSegment{src: src, off0: soff, b: b})
			}

			return r, nil
		}

		switch state {
		case zero:
			if !strings.HasPrefix(line.Src, "@x") {
				break
			}

			state = stateX
			orig = nil
			repl = nil
		case stateX:
			if strings.HasPrefix(line.Src, "@y") {
				coff = changes.Position().Offset
				state = stateY
				break
			}

			orig = append(orig, line.Src...)
		case stateY:
			if strings.HasPrefix(line.Src, "@z") {
				orig = bytes.TrimSpace(orig)
				repl = bytes.TrimSpace(repl)
				x := bytes.Index(b, orig)
				if x < 0 {
					return nil, fmt.Errorf("%v: change not found in src", line.Position)
				}

				r.a = append(r.a, &changerSegment{src: src, off0: soff, b: b[:x]})
				b = b[x+len(orig):]
				soff += x + len(orig)
				r.a = append(r.a, &changerSegment{src: changes, off0: coff, b: repl})
				state = zero
				break
			}

			repl = append(repl, line.Src...)
		default:
			panic(todo("%v: %q", state, line))
		}
	}
}

// AddLineColumnInfo implements RuneSource, but is a no-op.
func (c *Changer) AddLineColumnInfo(offset int, filename string, line, column int) {}

// C implements RuneSource.
func (c *Changer) C() (rune, error) {
	for {
		if c.ix < len(c.a) {
			s := c.a[c.ix]
			if s.off >= len(s.b) {
				c.ix++
				continue
			}

			r, _ := utf8.DecodeRune(s.b[s.off:])
			if r == utf8.RuneError {
				return 0, fmt.Errorf("%v: invalid rune", s.src.PositionFor(s.off))
			}

			return r, nil
		}

		return 0, io.EOF
	}
}

// Consume implements RuneSource.
func (c *Changer) Consume() {
	if c.ix < len(c.a) {
		s := c.a[c.ix]
		_, sz := utf8.DecodeRune(s.b[s.off:])
		s.off += sz
		if s.off >= len(s.b) {
			c.ix++
		}
	}
}

// Position implements RuneSource.
func (c *Changer) Position() (r token.Position) {
	if c.ix < len(c.a) {
		s := c.a[c.ix]
		return s.src.PositionFor(s.off0 + s.off)
	}

	if len(c.a) == 0 {
		return r
	}

	s := c.a[len(c.a)-1]
	return s.src.PositionFor(s.off0 + s.off)
}

// PositionFor implements RuneSource.
func (c *Changer) PositionFor(off int) token.Position {
	panic(todo(""))
}

type fsFile struct {
	b    []byte
	name string
	off  int64
}

func (f *fsFile) Close() error               { return nil }
func (f *fsFile) IsDir() bool                { return false }
func (f *fsFile) ModTime() time.Time         { return modTime }
func (f *fsFile) Mode() fs.FileMode          { return 0400 }
func (f *fsFile) Name() string               { return f.name }
func (f *fsFile) Size() int64                { return int64(len(f.b)) }
func (f *fsFile) Stat() (fs.FileInfo, error) { return f, nil }
func (f *fsFile) Sys() interface{}           { return nil }

func (f *fsFile) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		f.off += off
	case io.SeekStart:
		f.off = off
	case io.SeekEnd:
		f.off = int64(len(f.b)) + off
	}
	if f.off < 0 {
		f.off = 0
		return 0, fmt.Errorf("invalid seek")
	}

	if f.off > int64(len(f.b)) {
		f.off = int64(len(f.b))
		return 0, fmt.Errorf("invalid seek")
	}

	return f.off, nil
}

func (f *fsFile) Read(b []byte) (r int, err error) {
	r = copy(b, f.b[f.off:])
	f.off += int64(r)
	if r == 0 {
		err = io.EOF
	}
	return r, err
}

type cfs struct {
	m map[string][]byte
	r io.ReadSeeker
	sync.Mutex
}

func newCFS(fs fs.FS) *cfs {
	r, err := fs.Open("assets.tar.gz")
	if err != nil {
		panic(todo("", err))
	}

	return &cfs{
		m: map[string][]byte{},
		r: r.(io.ReadSeeker),
	}
}

func (f *cfs) Open(name string) (fs.File, error) {
	f.Lock()

	defer f.Unlock()

	b, ok := f.m[name]
	if !ok {
		f.r.Seek(0, io.SeekStart)
		gr, err := gzip.NewReader(f.r)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", name, err)
		}

		tr := tar.NewReader(gr)
		for {
			hdr, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					return nil, fmt.Errorf("%s: no such file", name)
				}

				return nil, fmt.Errorf("%s: %v", name, err)
			}

			if hdr.Name != name {
				continue
			}

			if b, err = io.ReadAll(tr); err != nil {
				return nil, fmt.Errorf("%s: %v", name, err)
			}

			f.m[name] = b
			break
		}
	}

	return &fsFile{b: b, name: name}, nil
}

// Open attempts to open 'name'. If not successful then it tries to open 'name'
// using 'search' paths. If still not found then Open may try to find the
// resource in Assets.
//
// If all letters of the base name of 'name' are upper case then the preceding
// steps may be extended by additionally looking for the lower case alternative
// of 'name'.
//
// The caller is responsible to properly .Close any returned non-nil fs.Files
// to avoid resource exhaustion.
func Open(name string, search []string) (f fs.File, err error) {
	if debug {
		defer func() {
			trc("Open(%q) %q -> %p %v", name, search, f, err)
		}()
	}

	if f, err := os.Open(name); err == nil {
		return f, nil
	}

	const (
		mfBasesArea    = "MFbases:"
		mfInputsArea   = "MFinputs:"
		texFontsArea   = "TeXfonts:"
		texFormatsArea = "TeXformats:"
		texInputsArea  = "TeXinputs:"
	)
	var area, dir, base, lcBase, ext string
	switch {
	case strings.HasPrefix(name, texFontsArea):
		area = texFontsArea
		base = name[len(texFontsArea):]
	case strings.HasPrefix(name, texInputsArea):
		area = texInputsArea
		base = name[len(texInputsArea):]
	case strings.HasPrefix(name, texFormatsArea):
		area = texFormatsArea
		base = name[len(texFormatsArea):]
	case strings.HasPrefix(name, mfBasesArea):
		area = mfBasesArea
		base = name[len(mfBasesArea):]
	case strings.HasPrefix(name, mfInputsArea):
		area = mfInputsArea
		base = name[len(mfInputsArea):]
	default:
		dir, base = filepath.Split(name)
	}
	if ucBase := strings.ToUpper(base); ucBase == base {
		lcBase = strings.ToLower(base)
	}

	if f, err := os.Open(filepath.Join(dir, base)); err == nil {
		return f, nil
	}

	if lcBase != "" {
		if f, err := os.Open(filepath.Join(dir, lcBase)); err == nil {
			return f, nil
		}
	}

	for _, path := range search {
		if f, err := os.Open(filepath.Join(path, base)); err == nil {
			return f, nil
		}

		if lcBase != "" {
			if f, err := os.Open(filepath.Join(path, lcBase)); err == nil {
				return f, nil
			}
		}
	}

	if area != "" {
		base = strings.ToLower(base)
	}
	ext = filepath.Ext(base)
	switch area {
	case texFontsArea:
		switch ext {
		case ".tfm":
			dir = "fonts/cm/tfm/"
		default:
			panic(todo("%q", name))
		}
	case
		mfInputsArea,
		texFormatsArea,
		texInputsArea:

		dir = "lib/"
	case mfBasesArea:
		dir = "mfbases/"
	default:
		return nil, fmt.Errorf("%s: no such file (searched %v)", name, search)
	}

	fn := dir + base
	if debug {
		trc("Open(%q) trying assets: %q", name, dir)
	}
	return Assets.Open(fn)
}
