// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package kpath provides tools to locate TeX related files.
//
// It loosely mimicks Kpathsea, as described in:
//   - https://texdoc.org/serve/kpathsea/0
package kpath // import "star-tex.org/x/tex/kpath"

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	stdpath "path"
	"path/filepath"
	"strings"
	"sync"

	"star-tex.org/x/tex/internal/tds"
)

var (
	once   sync.Once
	tdsCtx Context
)

// New returns a minimal kpath context initialized with the content of
// a minimal TeX Directory Structure.
func New() Context {
	once.Do(func() {
		tdsCtx, _ = NewFromFS(tds.FS)
	})
	return tdsCtx
}

// Context holds state to efficiently search for files in a TDS
// (TeX Directory Structure), as described in:
//   - https://tug.org/tds/tds.pdf
type Context struct {
	exts strset              // known common suffices
	db   map[string][]string // db of filename->dirs
	fs   fs.FS
}

func (ctx *Context) init(root fs.FS) {
	if ctx.exts.db == nil {
		ctx.exts = strsets["tex"]
	}
	if ctx.db == nil {
		ctx.db = make(map[string][]string)
	}
	ctx.fs = root
}

// // NewFromDB creates a kpath search from a TeX .cnf configuration file.
// func NewFromConfig(cfg io.Reader) (Context, error) {
// 	ctx, err := parseConfig(cfg)
// 	if err != nil {
// 		return Context{}, fmt.Errorf("kpath: could not parse config: %w", err)
// 	}
//
// 	ctx.init()
// 	return ctx, nil
// }

// NewFromDB creates a kpath search from a TeX ls-R db file.
func NewFromDB(r io.Reader) (Context, error) {
	return newFromDB(os.DirFS("/"), r)
}

func newFromDB(root fs.FS, r io.Reader) (Context, error) {
	dir := "/"
	if f, ok := r.(interface{ Name() string }); ok {
		dir = stdpath.Dir(filepath.ToSlash(f.Name()))
	}
	ctx, err := parseDB(dir, r)
	if err != nil {
		return Context{}, fmt.Errorf("kpath: could not parse db file: %w", err)
	}

	ctx.init(root)
	return ctx, nil
}

// NewFromFS creates a kpath search context from the provided filesystem.
//
// NewFromFS checks first whether an ls-R database exists at the root of the
// provided filesystem, and otherwise walks the whole fs.
func NewFromFS(fsys fs.FS) (Context, error) {
	var ctx Context
	ctx.init(fsys)

	if _, err := fs.Stat(fsys, "ls-R"); err == nil {
		db, err := fsys.Open("ls-R")
		if err != nil {
			return ctx, fmt.Errorf("kpath: could not open db file: %w", err)
		}
		defer db.Close()
		return newFromDB(fsys, db)
	}

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fname := stdpath.Base(path)
		ctx.db[fname] = append(ctx.db[fname], path)
		return nil
	})
	if err != nil {
		return ctx, fmt.Errorf("kpath: could not walk fs: %w", err)
	}

	return ctx, nil
}

// FS returns the underlying filesystem this context is using.
func (ctx Context) FS() fs.FS {
	return ctx.fs
}

// Open opens the named file for reading.
func (ctx Context) Open(name string) (fs.File, error) {
	f, err := ctx.fs.Open(name)
	if err == nil {
		return f, nil
	}
	// FIXME(sbinet): ctx.fs.Open may fail to open the named file because
	// of absolute vs relative path issues.
	// e.g.:
	//  - ctx.fs is rooted at /usr/share/texmf-dist
	//  - one requests /usr/share/texmf-dist/foo.txt (which exists)
	// Name is thus "/usr/share/texmf-dist/foo.txt",
	// but from the POV of ctx.fs, only "foo.txt" exists.
	// Giving the absolute path from "/" won't work.
	//
	// In the meantime, resort to just calling to os.Open.
	return os.Open(name)
}

// Find returns the full path to the named file if it could be found within the
// TeXMF distribution system.
// Find returns an error if no file or more than one file were found.
func (ctx Context) Find(name string) (string, error) {
	names, err := ctx.FindAll(name)
	if err != nil {
		return "", err
	}

	switch n := len(names); n {
	case 1:
		return names[0], nil
	case 0:
		return "", fmt.Errorf("kpath: could not find a match for %q", name)
	default:
		return "", fmt.Errorf("kpath: too many hits for file %q (n=%d)", name, n)
	}
}

// FindAll returns the full path to all the files matching name that could be
// found within the TeXMF distribution system.
// Find returns an error if no file was found.
func (ctx Context) FindAll(name string) ([]string, error) {
	// TODO(sbinet): handle (all) standard exts.
	// TODO(sbinet): handle multi-root TEXMFs

	orig := name
	name = filepath.ToSlash(name)
	var (
		subdir = strings.Contains(name, "/")
		ext    = stdpath.Ext(name)
	)
	switch ext {
	case "":
		// try some extensions.
		for _, ext := range ctx.exts.ks {
			names, ok := ctx.lookup(name+ext, subdir)
			if ok {
				return names, nil
			}
		}

		names, ok := ctx.lookup(name, subdir)
		if ok {
			return names, nil
		}

	default:

		if !ctx.exts.has(ext) {
			for _, ext := range ctx.exts.ks {
				names, ok := ctx.lookup(name+ext, subdir)
				if ok {
					return names, nil
				}
			}
		}

		names, ok := ctx.lookup(name, subdir)
		if ok {
			return names, nil
		}
	}

	return nil, fmt.Errorf("kpath: could not find file %q", orig)
}

func (s Context) lookup(name string, subdir bool) ([]string, bool) {
	if !subdir {
		names, ok := s.db[name]
		return names, ok
	}

	var (
		ok    = false
		names = make([]string, 0, 16)
	)
	for _, vs := range s.db {
		for _, v := range vs {
			if !strings.HasSuffix(v, name) {
				continue
			}
			names = append(names, v)
			ok = true
		}
	}

	return names, ok
}
