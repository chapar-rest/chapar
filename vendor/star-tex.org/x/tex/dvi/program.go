// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dvi

import (
	"errors"
	"fmt"
	"io"

	"star-tex.org/x/tex/internal/iobuf"
)

var (
	errNoPre          = errors.New("dvi: missing preamble")
	errInvalidVersion = errors.New("dvi: invalid DVI version")
	errInvalidDVI     = errors.New("dvi: invalid DVI file")

	errNoBOP = errors.New("dvi: missing BOP")
	errNoEOP = errors.New("dvi: missing EOP")
)

type span struct {
	beg uint32
	end uint32
}

type Program struct {
	r *iobuf.Reader

	pre   CmdPre
	post  CmdPost
	ppost int

	// npages int

	pages []span
	fonts map[int]fntdef

	// max struct {
	// 	width  int
	// 	height int
	// 	stack  int
	// }
}

func Compile(instr []byte) (Program, error) {
	prog := Program{
		r:     iobuf.NewReader(instr),
		fonts: make(map[int]fntdef),
	}

	if opCode(prog.r.PeekU8()) != opPre {
		return prog, errNoPre
	}

	prog.pre.read(prog.r)
	if prog.pre.Version != dviVersion {
		return prog, errInvalidVersion
	}

	var (
		start = prog.r.Pos()
		vers  uint8
	)

	// locate post- and post-postamble
	_, err := prog.r.Seek(5, io.SeekEnd)
	if err != nil {
		return prog, fmt.Errorf("dvi: could not find post-postamble: %w", err)
	}

	for {
		v := prog.r.ReadU8()
		if v != dviEOF {
			vers = v
			break
		}
		_, err = prog.r.Seek(-2, io.SeekCurrent)
		if err != nil {
			return prog, fmt.Errorf("dvi: could not rewind: %w", err)
		}
	}

	if vers != prog.pre.Version {
		return prog, fmt.Errorf(
			"dvi: version skew (pre=%d, post=%d)",
			prog.pre.Version, vers,
		)
	}

	_, err = prog.r.Seek(-5, io.SeekCurrent)
	if err != nil {
		return prog, fmt.Errorf("dvi: could not rewind to post-pointer: %w", err)
	}
	var (
		bop = prog.r.ReadU32()
		eop = bop
	)
	prog.ppost = int(bop)

	_, err = prog.r.Seek(int64(bop), io.SeekStart)
	if err != nil {
		return prog, fmt.Errorf("dvi: could not seek to postamble: %w", err)
	}

	if opCode(prog.r.PeekU8()) != opPost {
		return prog, fmt.Errorf("dvi: could not locate postamble: %w", errInvalidDVI)
	}
	prog.post.read(prog.r)

fonts:
	for {
		switch op := opCode(prog.r.PeekU8()); op {
		case opFntDef1:
			cmd := op.cmd().(*CmdFntDef1)
			cmd.read(prog.r)
			prog.defineFont(int(cmd.ID), fntdef{
				ID:       int(cmd.ID),
				Checksum: cmd.Checksum,
				Size:     cmd.Size,
				Design:   cmd.Design,
				Area:     cmd.Area,
				Name:     cmd.Font,
			})
		case opFntDef2:
			cmd := op.cmd().(*CmdFntDef2)
			cmd.read(prog.r)
			prog.defineFont(int(cmd.ID), fntdef{
				ID:       int(cmd.ID),
				Checksum: cmd.Checksum,
				Size:     cmd.Size,
				Design:   cmd.Design,
				Area:     cmd.Area,
				Name:     cmd.Font,
			})
		case opFntDef3:
			cmd := op.cmd().(*CmdFntDef3)
			cmd.read(prog.r)
			prog.defineFont(int(cmd.ID), fntdef{
				ID:       int(cmd.ID),
				Checksum: cmd.Checksum,
				Size:     cmd.Size,
				Design:   cmd.Design,
				Area:     cmd.Area,
				Name:     cmd.Font,
			})
		case opFntDef4:
			cmd := op.cmd().(*CmdFntDef4)
			cmd.read(prog.r)
			prog.defineFont(int(cmd.ID), fntdef{
				ID:       int(cmd.ID),
				Checksum: cmd.Checksum,
				Size:     cmd.Size,
				Design:   cmd.Design,
				Area:     cmd.Area,
				Name:     cmd.Font,
			})
		case opNOP:
			cmd := op.cmd()
			cmd.read(prog.r)
		default:
			break fonts
		}
	}

	bop = prog.post.BOP
	prog.pages = make([]span, int(prog.post.Pages))
	page := len(prog.pages)
	// build pages look-up table.
	for bop != ^uint32(0) {
		page--
		prog.pages[page].end = eop
		_, err = prog.r.Seek(int64(bop), io.SeekStart)
		if err != nil {
			return prog, fmt.Errorf("dvi: could not seek to page %d: %w", page, err)
		}

		var (
			pos = uint32(prog.r.Pos())
			op  = opCode(prog.r.PeekU8())
		)
		eop = pos
		if op != opBOP {
			return prog, fmt.Errorf("could not seek to BOP page=%d: op=%v", page, op.cmd().Name())
		}

		cmd := op.cmd().(*CmdBOP)
		cmd.read(prog.r)
		prog.pages[int(cmd.C0)-1].beg = pos
		bop = uint32(cmd.Prev)
	}

	prog.r.SetPos(start)
	return prog, nil
}

func (prog *Program) defineFont(id int, def fntdef) {
	prog.fonts[id] = def
}
