// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dvi

import (
	"io"

	"star-tex.org/x/tex/font/fixed"
	"star-tex.org/x/tex/kpath"
)

type Option func(cfg *config) error

type config struct {
	ctx kpath.Context
	rdr Renderer
	out io.Writer

	xoff int32 // width offset
	yoff int32 // height offset

	handlers []Handler
}

func newConfig() *config {
	return &config{
		ctx: kpath.New(),
		rdr: nopRenderer{},
		out: io.Discard,
	}
}

func WithContext(ctx kpath.Context) Option {
	return func(cfg *config) error {
		cfg.ctx = ctx
		return nil
	}
}

func WithRenderer(rdr Renderer) Option {
	return func(cfg *config) error {
		cfg.rdr = rdr
		return nil
	}
}

func WithLogOutput(w io.Writer) Option {
	return func(cfg *config) error {
		cfg.out = w
		return nil
	}
}

func WithOffsetX(v fixed.Int12_20) Option {
	return func(cfg *config) error {
		cfg.xoff = int32(v)
		return nil
	}
}

func WithOffsetY(v fixed.Int12_20) Option {
	return func(cfg *config) error {
		cfg.yoff = int32(v)
		return nil
	}
}

// WithHandlers specifies a list of Handlers a DVI Machine will use to
// handle the XXXn special commands.
func WithHandlers(hs ...Handler) Option {
	return func(cfg *config) error {
		cfg.handlers = hs
		return nil
	}
}
