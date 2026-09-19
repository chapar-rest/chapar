// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dvi implements encoding and decoding DVI documents.
//
// More informations about the DVI standard can be found here:
//
//   - https://ctan.crest.fr/tex-archive/dviware/driv-standard/level-0/dvistd0.pdf
package dvi // import "star-tex.org/x/tex/dvi"

import (
	"errors"
	"fmt"
	"image/color"
	"io"

	"star-tex.org/x/tex/internal/iobuf"
)

// Renderer defines the protocol to draw a DVI document.
type Renderer interface {
	Init(pre *CmdPre, post *CmdPost)
	BOP(bop *CmdBOP)
	EOP()

	DrawGlyph(x, y int32, font Font, glyph rune, c color.Color)
	DrawRule(x, y, w, h int32, c color.Color)
}

type nopRenderer struct{}

func (nopRenderer) Init(pre *CmdPre, post *CmdPost) {}
func (nopRenderer) BOP(cmd *CmdBOP)                 {}
func (nopRenderer) EOP()                            {}

func (nopRenderer) DrawGlyph(x, y int32, font Font, glyph rune, c color.Color) {}
func (nopRenderer) DrawRule(x, y, w, h int32, c color.Color)                   {}

var (
	_ Renderer = (*nopRenderer)(nil)
)

// Dump reads r until EOF and calls f for each decoded DVI command.
func Dump(r io.Reader, f func(cmd Cmd) error) error {
	buf, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("dvi: could not read DVI program: %w", err)
	}

	rr := iobuf.NewReader(buf)

	for {
		var (
			op  = opCode(rr.PeekU8())
			cmd = op.cmd()
		)
		if cmd == nil {
			return fmt.Errorf("dvi: unknown opcode %v (v=0x%x)", op, op)
		}

		cmd.read(rr)

		err = f(cmd)
		if err != nil {
			return fmt.Errorf("dvi: could not call user provided function: %w", err)
		}

		if cmd.opcode() == opPostPost {
			break
		}
	}

	return nil
}

// Handler handles special DVI XXXn commands.
// Users can customize how a DVI Machine will handle these commands.
//
// Special commands are usually written in DVI files with an opaque payload
// of bytes, starting with a "well known" prefix.
// Ex:
//
//	color push gray 0
//	color pop
//	color push  BurntOrange
//
// A Handler should return ErrSkipHandler if does not know how to handle a given
// special command data.
type Handler interface {
	// Handle handles a special DVI command.
	Handle(p []byte) error
}

// ErrSkipHandler is used as a return value from Handler.Handle to indicate
// that the Handler should be skipped and is not suited to handle that special
// command.
var ErrSkipHandler = errors.New("dvi: skip handler")
