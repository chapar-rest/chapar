// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dvi

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"star-tex.org/x/tex/font/fixed"
	"star-tex.org/x/tex/font/tfm"
	"star-tex.org/x/tex/kpath"
)

const (
	maxdrift = 2
)

// Machine defines a DVI machine, handling DVI registers and DVI commands.
type Machine struct {
	rdr Renderer
	ktx kpath.Context

	state state

	conv     float32 // converts DVI units to pixels
	trueConv float32 // converts unmagnified DVI units to pixels
	xoff     int32   // width offset
	yoff     int32   // height offset

	w   io.Writer
	buf []byte // 80-col buffer of text

	handlers []Handler // list of registered special handlers for CmdXXXn commands.

	color colorHandler // special handling of colors
}

// NewMachine creates a new DVI machine.
func NewMachine(opts ...Option) Machine {
	cfg := newConfig()
	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			panic(err)
		}
	}

	m := Machine{
		ktx:   cfg.ctx,
		rdr:   cfg.rdr,
		state: newState(),

		xoff: cfg.xoff,
		yoff: cfg.yoff,

		w:   cfg.out,
		buf: make([]byte, 0, 80-len("[]\n")),

		handlers: cfg.handlers,
		color:    nilColorHandler{},
	}

	for _, h := range m.handlers {
		if hh, ok := h.(colorHandler); ok {
			m.color = hh
			break
		}
	}

	return m
}

// Run executes the whole DVI program on this DVI machine.
func (m *Machine) Run(p Program) error {
	m.load(p)
	for i := range p.pages {
		err := m.run(p, i)
		if err != nil {
			return fmt.Errorf("dvi: could not process page %d: %w", i+1, err)
		}
	}

	return nil
}

func (m *Machine) load(p Program) {

	m.printf("numerator/denominator=%d/%d\n", p.pre.Num, p.pre.Den)
	res := float32(300.0)
	conv := float32(p.pre.Num) / 254000.0 * (res / float32(p.pre.Den))
	m.trueConv = conv
	m.conv = conv * float32(p.pre.Mag) / 1000.0
	m.printf("magnification=%d; %16.8f pixels per DVI unit\n", p.pre.Mag, m.conv)
	m.printf("'%s'\n", p.pre.Msg)

	m.printf("Postamble starts at byte %d.\n", p.ppost)
	m.printf("maxv=%d, maxh=%d, maxstackdepth=%d, totalpages=%d\n",
		p.post.Height, p.post.Width, p.post.MaxStack,
		p.post.Pages,
	)

	m.state.fonts = p.fonts
	fonts := make([]int, 0, len(p.fonts))
	for id := range p.fonts {
		fonts = append(fonts, id)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(fonts)))

	for _, id := range fonts {
		fnt := m.state.fonts[id]
		m.printf(
			"Font %d: %s---loaded at size %d DVI units \n",
			id, fnt.Name, fnt.Size,
		)
	}
}

func (m *Machine) run(p Program, ip int) error {
	var (
		beg = int(p.pages[ip].beg)
		end = int(p.pages[ip].end)
		eop = false
	)

	m.rdr.Init(&p.pre, &p.post)

	p.r.SetPos(beg)
	op := opCode(p.r.PeekU8())
	if op != opBOP {
		return errNoBOP
	}

	bop := op.cmd().(*CmdBOP)
	bop.read(p.r)
	m.state.reset(m.xoff, m.yoff, m.pixels)

	m.printf(" \n%d: beginning of page %d \n", beg, bop.C0)

	m.rdr.BOP(bop)

	for p.r.Pos() < end {
		pos := p.r.Pos()
		switch op := opCode(p.r.PeekU8()); op {
		case opBOP, opPre, opPost, opPostPost:
			return fmt.Errorf("dvi: invalid opcode=%s inside a page", op.cmd().Name())

		case opEOP:
			eop = true
			m.flushText()
			m.printf("%d: eop", pos)

			op.cmd().read(p.r)
			m.rdr.EOP()

			lvl := len(m.state.stack) - 1
			if lvl != 0 {
				m.printf("stack not empty at end of page (level %d)!\n", lvl)
			}

		case opSetChar000, opSetChar001, opSetChar002, opSetChar003, opSetChar004,
			opSetChar005, opSetChar006, opSetChar007, opSetChar008, opSetChar009,
			opSetChar010, opSetChar011, opSetChar012, opSetChar013, opSetChar014,
			opSetChar015, opSetChar016, opSetChar017, opSetChar018, opSetChar019,
			opSetChar020, opSetChar021, opSetChar022, opSetChar023, opSetChar024,
			opSetChar025, opSetChar026, opSetChar027, opSetChar028, opSetChar029,
			opSetChar030, opSetChar031, opSetChar032, opSetChar033, opSetChar034,
			opSetChar035, opSetChar036, opSetChar037, opSetChar038, opSetChar039,
			opSetChar040, opSetChar041, opSetChar042, opSetChar043, opSetChar044,
			opSetChar045, opSetChar046, opSetChar047, opSetChar048, opSetChar049,
			opSetChar050, opSetChar051, opSetChar052, opSetChar053, opSetChar054,
			opSetChar055, opSetChar056, opSetChar057, opSetChar058, opSetChar059,
			opSetChar060, opSetChar061, opSetChar062, opSetChar063, opSetChar064,
			opSetChar065, opSetChar066, opSetChar067, opSetChar068, opSetChar069,
			opSetChar070, opSetChar071, opSetChar072, opSetChar073, opSetChar074,
			opSetChar075, opSetChar076, opSetChar077, opSetChar078, opSetChar079,
			opSetChar080, opSetChar081, opSetChar082, opSetChar083, opSetChar084,
			opSetChar085, opSetChar086, opSetChar087, opSetChar088, opSetChar089,
			opSetChar090, opSetChar091, opSetChar092, opSetChar093, opSetChar094,
			opSetChar095, opSetChar096, opSetChar097, opSetChar098, opSetChar099,
			opSetChar100, opSetChar101, opSetChar102, opSetChar103, opSetChar104,
			opSetChar105, opSetChar106, opSetChar107, opSetChar108, opSetChar109,
			opSetChar110, opSetChar111, opSetChar112, opSetChar113, opSetChar114,
			opSetChar115, opSetChar116, opSetChar117, opSetChar118, opSetChar119,
			opSetChar120, opSetChar121, opSetChar122, opSetChar123, opSetChar124,
			opSetChar125, opSetChar126, opSetChar127:
			cmd := op.cmd().(*CmdSetChar)
			cmd.read(p.r)

			switch {
			case cmd.Value > ' ' && cmd.Value <= '~':
				m.outText(cmd.Value)
			default:
				m.flushText()
			}
			m.printf("%d: %s", pos, strings.Replace(cmd.Name(), "_", "", -1))
			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not set char %q: %w", op, err)
			}

		case opSet1:
			cmd := op.cmd().(*CmdSet1)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: set1 %d", pos, cmd.Value)

			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not set1: %w", err)
			}

		case opSet2:
			cmd := op.cmd().(*CmdSet2)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: set2 %d", pos, cmd.Value)

			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not set2: %w", err)
			}

		case opSet3:
			cmd := op.cmd().(*CmdSet3)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: set3 %d", pos, cmd.Value)

			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not set3: %w", err)
			}

		case opSet4:
			cmd := op.cmd().(*CmdSet4)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: set4 %d", pos, cmd.Value)

			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not set4: %w", err)
			}

		case opSetRule:
			cmd := op.cmd().(*CmdSetRule)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: setrule", pos)

			err := m.drawRule(cmd.opcode(), cmd.Height, cmd.Width)
			if err != nil {
				return fmt.Errorf("could not setrule(%d, %d): %w", cmd.Height, cmd.Width, err)
			}

		case opPut1:
			cmd := op.cmd().(*CmdPut1)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: put1 %d", pos, cmd.Value)

			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not put1(%d): %w", cmd.Value, err)
			}

		case opPut2:
			cmd := op.cmd().(*CmdPut2)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: put2 %d", pos, cmd.Value)

			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not put2(%d): %w", cmd.Value, err)
			}

		case opPut3:
			cmd := op.cmd().(*CmdPut3)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: put3 %d", pos, cmd.Value)

			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not put3(%d): %w", cmd.Value, err)
			}

		case opPut4:
			cmd := op.cmd().(*CmdPut4)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: put4 %d", pos, cmd.Value)

			err := m.drawGlyph(cmd.opcode(), int32(cmd.Value))
			if err != nil {
				return fmt.Errorf("could not put4(%d): %w", cmd.Value, err)
			}

		case opPutRule:
			cmd := op.cmd().(*CmdPutRule)
			cmd.read(p.r)

			m.flushText()
			m.printf("%d: putrule", pos)

			err := m.drawRule(cmd.opcode(), cmd.Height, cmd.Width)
			if err != nil {
				return fmt.Errorf("could not putrule(%d, %d): %w", cmd.Height, cmd.Width, err)
			}

		case opPush:
			m.flushText()
			m.printf("%d: push \n", pos)
			lvl := len(m.state.stack) - 1
			op.cmd().(*CmdPush).read(p.r)
			m.state.push()
			cur := m.state.cur()
			m.printf("level %d:(h=%d,v=%d,w=%d,x=%d,y=%d,z=%d,hh=%d,vv=%d)",
				lvl,
				cur.h, cur.v, cur.w, cur.x, cur.y, cur.z,
				cur.hh, cur.vv,
			)

		case opPop:
			m.flushText()
			m.printf("%d: pop \n", pos)
			op.cmd().(*CmdPop).read(p.r)
			m.state.pop()

			lvl := len(m.state.stack) - 1
			cur := m.state.cur()
			m.printf("level %d:(h=%d,v=%d,w=%d,x=%d,y=%d,z=%d,hh=%d,vv=%d)",
				lvl,
				cur.h, cur.v, cur.w, cur.x, cur.y, cur.z,
				cur.hh, cur.vv,
			)

		case opRight1:
			cmd := op.cmd().(*CmdRight1)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: right1 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not right1(%d): %w", cmd.Value, err)
			}

		case opRight2:
			cmd := op.cmd().(*CmdRight2)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: right2 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not right2(%d): %w", cmd.Value, err)
			}

		case opRight3:
			cmd := op.cmd().(*CmdRight3)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: right3 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not right3(%d): %w", cmd.Value, err)
			}

		case opRight4:
			cmd := op.cmd().(*CmdRight4)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: right4 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not right4(%d): %w", cmd.Value, err)
			}

		case opW0:
			cmd := op.cmd().(*CmdW0)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.hh = m.outSpace(cur.w)
			m.printf("%d: w0 %d", pos, cur.w)

			err := m.moveright(cur.w)
			if err != nil {
				return fmt.Errorf("could not w0(%d): %w", cur.w, err)
			}

		case opW1:
			cmd := op.cmd().(*CmdW1)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.w = cmd.Value
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: w1 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not w1(%d): %w", cmd.Value, err)
			}

		case opW2:
			cmd := op.cmd().(*CmdW2)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.w = cmd.Value
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: w2 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not w2(%d): %w", cmd.Value, err)
			}

		case opW3:
			cmd := op.cmd().(*CmdW3)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.w = cmd.Value
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: w3 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not w3(%d): %w", cmd.Value, err)
			}

		case opW4:
			cmd := op.cmd().(*CmdW4)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.w = cmd.Value
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: w4 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not w4(%d): %w", cmd.Value, err)
			}

		case opX0:
			cmd := op.cmd().(*CmdX0)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.hh = m.outSpace(cur.x)
			m.printf("%d: x0 %d", pos, cur.x)

			err := m.moveright(cur.x)
			if err != nil {
				return fmt.Errorf("could not x0(%d): %w", cur.x, err)
			}

		case opX1:
			cmd := op.cmd().(*CmdX1)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.x = cmd.Value
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: x1 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not x1(%d): %w", cmd.Value, err)
			}

		case opX2:
			cmd := op.cmd().(*CmdX2)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.x = cmd.Value
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: x2 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not x2(%d): %w", cmd.Value, err)
			}

		case opX3:
			cmd := op.cmd().(*CmdX3)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.x = cmd.Value
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: x3 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not x3(%d): %w", cmd.Value, err)
			}

		case opX4:
			cmd := op.cmd().(*CmdX4)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.x = cmd.Value
			cur.hh = m.outSpace(cmd.Value)
			m.printf("%d: x4 %d", pos, cmd.Value)

			err := m.moveright(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not x4(%d): %w", cmd.Value, err)
			}

		case opDown1:
			cmd := op.cmd().(*CmdDown1)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: down1 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not down1(%d): %w", cmd.Value, err)
			}

		case opDown2:
			cmd := op.cmd().(*CmdDown2)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: down2 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not down2(%d): %w", cmd.Value, err)
			}

		case opDown3:
			cmd := op.cmd().(*CmdDown3)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: down3 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not down3(%d): %w", cmd.Value, err)
			}

		case opDown4:
			cmd := op.cmd().(*CmdDown4)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: down4 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not down4(%d): %w", cmd.Value, err)
			}

		case opY0:
			cmd := op.cmd().(*CmdY0)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.vv = m.outVMove(cur.y)
			m.flushText()
			m.printf("%d: y0 %d", pos, cur.y)

			err := m.movedown(cur.y)
			if err != nil {
				return fmt.Errorf("could not y0(%d): %w", cur.y, err)
			}

		case opY1:
			cmd := op.cmd().(*CmdY1)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.y = cmd.Value
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: y1 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not y1(%d): %w", cmd.Value, err)
			}

		case opY2:
			cmd := op.cmd().(*CmdY2)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.y = cmd.Value
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: y2 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not y2(%d): %w", cmd.Value, err)
			}

		case opY3:
			cmd := op.cmd().(*CmdY3)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.y = cmd.Value
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: y3 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not y3(%d): %w", cmd.Value, err)
			}

		case opY4:
			cmd := op.cmd().(*CmdY4)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.y = cmd.Value
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: y4 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not y4(%d): %w", cmd.Value, err)
			}

		case opZ0:
			cmd := op.cmd().(*CmdZ0)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.vv = m.outVMove(cur.z)
			m.flushText()
			m.printf("%d: z0 %d", pos, cur.z)

			err := m.movedown(cur.z)
			if err != nil {
				return fmt.Errorf("could not z0(%d): %w", cur.z, err)
			}

		case opZ1:
			cmd := op.cmd().(*CmdZ1)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.z = cmd.Value
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: z1 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not z1(%d): %w", cmd.Value, err)
			}

		case opZ2:
			cmd := op.cmd().(*CmdZ2)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.z = cmd.Value
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: z2 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not z2(%d): %w", cmd.Value, err)
			}

		case opZ3:
			cmd := op.cmd().(*CmdZ3)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.z = cmd.Value
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: z3 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not z3(%d): %w", cmd.Value, err)
			}

		case opZ4:
			cmd := op.cmd().(*CmdZ4)
			cmd.read(p.r)

			cur := m.state.cur()
			cur.z = cmd.Value
			cur.vv = m.outVMove(cmd.Value)
			m.flushText()
			m.printf("%d: z4 %d", pos, cmd.Value)

			err := m.movedown(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not z4(%d): %w", cmd.Value, err)
			}

		case opFntNum00, opFntNum01, opFntNum02, opFntNum03, opFntNum04,
			opFntNum05, opFntNum06, opFntNum07, opFntNum08, opFntNum09,
			opFntNum10, opFntNum11, opFntNum12, opFntNum13, opFntNum14,
			opFntNum15, opFntNum16, opFntNum17, opFntNum18, opFntNum19,
			opFntNum20, opFntNum21, opFntNum22, opFntNum23, opFntNum24,
			opFntNum25, opFntNum26, opFntNum27, opFntNum28, opFntNum29,
			opFntNum30, opFntNum31, opFntNum32, opFntNum33, opFntNum34,
			opFntNum35, opFntNum36, opFntNum37, opFntNum38, opFntNum39,
			opFntNum40, opFntNum41, opFntNum42, opFntNum43, opFntNum44,
			opFntNum45, opFntNum46, opFntNum47, opFntNum48, opFntNum49,
			opFntNum50, opFntNum51, opFntNum52, opFntNum53, opFntNum54,
			opFntNum55, opFntNum56, opFntNum57, opFntNum58, opFntNum59,
			opFntNum60, opFntNum61, opFntNum62, opFntNum63:
			cmd := op.cmd().(*CmdFntNum)
			cmd.read(p.r)

			m.state.f = int(cmd.ID)
			m.flushText()
			m.printf(
				"%d: fntnum%d current font is %s",
				pos, cmd.ID, m.state.fonts[m.state.f].Name,
			)

		case opFnt1:
			cmd := op.cmd().(*CmdFnt1)
			cmd.read(p.r)
			m.state.f = int(cmd.ID)
			m.flushText()
			m.printf(
				"%d: fnt1 %d current font is %s",
				pos, cmd.ID, m.state.fonts[m.state.f].Name,
			)

		case opFnt2:
			cmd := op.cmd().(*CmdFnt2)
			cmd.read(p.r)
			m.state.f = int(cmd.ID)
			m.flushText()
			m.printf(
				"%d: fnt2 %d current font is %s",
				pos, cmd.ID, m.state.fonts[m.state.f].Name,
			)

		case opFnt3:
			cmd := op.cmd().(*CmdFnt3)
			cmd.read(p.r)
			m.state.f = int(cmd.ID)
			m.flushText()
			m.printf(
				"%d: fnt3 %d current font is %s",
				pos, cmd.ID, m.state.fonts[m.state.f].Name,
			)

		case opFnt4:
			cmd := op.cmd().(*CmdFnt4)
			cmd.read(p.r)
			m.state.f = int(cmd.ID)
			m.flushText()
			m.printf(
				"%d: fnt4 %d current font is %s",
				pos, cmd.ID, m.state.fonts[m.state.f].Name,
			)

		case opXXX1:
			cmd := op.cmd().(*CmdXXX1)
			cmd.read(p.r)
			m.flushText()
			m.printf("%d: xxx '%s'", pos, cmd.Value)

			err := m.handleSpecial(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not xxx1 %q: %w", cmd.Value, err)
			}

		case opXXX2:
			cmd := op.cmd().(*CmdXXX2)
			cmd.read(p.r)
			m.flushText()
			m.printf("%d: xxx '%s'", pos, cmd.Value)

			err := m.handleSpecial(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not xxx2 %q: %w", cmd.Value, err)
			}

		case opXXX3:
			cmd := op.cmd().(*CmdXXX3)
			cmd.read(p.r)
			m.flushText()
			m.printf("%d: xxx '%s'", pos, cmd.Value)

			err := m.handleSpecial(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not xxx3 %q: %w", cmd.Value, err)
			}

		case opXXX4:
			cmd := op.cmd().(*CmdXXX4)
			cmd.read(p.r)
			m.flushText()
			m.printf("%d: xxx '%s'", pos, cmd.Value)

			err := m.handleSpecial(cmd.Value)
			if err != nil {
				return fmt.Errorf("could not xxx4 %q: %w", cmd.Value, err)
			}

		case opFntDef1:
			cmd := op.cmd().(*CmdFntDef1)
			cmd.read(p.r)
			m.flushText()
			m.printf("%d: fntdef1 %d: %s", pos, cmd.ID, cmd.Font)

			err := m.defineFont(int(cmd.ID), fntdef{
				ID:       int(cmd.ID),
				Checksum: cmd.Checksum,
				Size:     cmd.Size,
				Design:   cmd.Design,
				Area:     cmd.Area,
				Name:     cmd.Font,
			})
			if err != nil {
				return fmt.Errorf("could not fntdef1 %d: %w", cmd.ID, err)
			}

		case opFntDef2:
			cmd := op.cmd().(*CmdFntDef2)
			cmd.read(p.r)
			m.flushText()
			m.printf("%d: fntdef2 %d: %s", pos, cmd.ID, cmd.Font)

			err := m.defineFont(int(cmd.ID), fntdef{
				ID:       int(cmd.ID),
				Checksum: cmd.Checksum,
				Size:     cmd.Size,
				Design:   cmd.Design,
				Area:     cmd.Area,
				Name:     cmd.Font,
			})
			if err != nil {
				return fmt.Errorf("could not fntdef2 %d: %w", cmd.ID, err)
			}

		case opFntDef3:
			cmd := op.cmd().(*CmdFntDef3)
			cmd.read(p.r)
			m.flushText()
			m.printf("%d: fntdef3 %d: %s", pos, cmd.ID, cmd.Font)

			err := m.defineFont(int(cmd.ID), fntdef{
				ID:       int(cmd.ID),
				Checksum: cmd.Checksum,
				Size:     cmd.Size,
				Design:   cmd.Design,
				Area:     cmd.Area,
				Name:     cmd.Font,
			})
			if err != nil {
				return fmt.Errorf("could not fntdef3 %d: %w", cmd.ID, err)
			}

		case opFntDef4:
			cmd := op.cmd().(*CmdFntDef4)
			cmd.read(p.r)
			m.flushText()
			m.printf("%d: fntdef4 %d: %s", pos, cmd.ID, cmd.Font)

			err := m.defineFont(int(cmd.ID), fntdef{
				ID:       int(cmd.ID),
				Checksum: cmd.Checksum,
				Size:     cmd.Size,
				Design:   cmd.Design,
				Area:     cmd.Area,
				Name:     cmd.Font,
			})
			if err != nil {
				return fmt.Errorf("could not fntdef4 %d: %w", cmd.ID, err)
			}

		default:
			cmd := op.cmd()
			cmd.read(p.r)
			panic(fmt.Errorf("invalid dvi command %q (op=%d)", op.cmd().Name(), op))
		}
		m.printf(" \n")
	}

	if !eop {
		return errNoEOP
	}

	return nil
}

// drawGlyph finishes a command that either sets or puts a character.
func (m *Machine) drawGlyph(op opCode, cmd int32) error {

	cur := m.state.cur()

	fnt, err := m.loadFont(m.state.f)
	if err != nil {
		return err
	}

	m.rdr.DrawGlyph(cur.h, cur.v, *fnt, rune(cmd), m.color.Color())

	adv, ok := fnt.advance(rune(cmd))
	if !ok {
		return fmt.Errorf("dvi: font %q has no glyph %c", fnt.name, cmd)
	}

	if op >= opPut1 {
		return nil
	}

	cur.hh += m.pixels(int32(adv))

	return m.moveright(int32(adv))
}

// drawRule finishes a command that either sets or puts a rule.
func (m *Machine) drawRule(op opCode, height, width int32) error {
	m.printf(" height %d, width %d", height, width)
	switch {
	case height <= 0 || width <= 0:
		m.printf(" (invisible) ")
	default:
		m.printf(" (%dx%d pixels)", m.rulepixels(height), m.rulepixels(width))
	}

	cur := m.state.cur()
	m.rdr.DrawRule(cur.h, cur.v, width, height, m.color.Color())

	if op == opPutRule {
		return nil
	}
	cur.hh += m.rulepixels(width)

	m.printf(" \n")
	return m.moveright(width)
}

// moveright finishes a command that sets h += q.
func (m *Machine) moveright(q int32) error {
	cur := m.state.cur()
	old := cur.h
	hhh := m.pixels(cur.h + q)
	if absI32(hhh-cur.hh) > maxdrift {
		switch {
		case hhh > cur.hh:
			cur.hh = hhh - maxdrift
		default:
			cur.hh = hhh + maxdrift
		}
	}
	cur.h += q
	m.printf(" h:=%d%+d=%d, hh:=%d", old, q, cur.h, cur.hh)
	return nil
}

// movedown finishes a command that sets v += q.
func (m *Machine) movedown(q int32) error {
	cur := m.state.cur()
	old := cur.v
	vvv := m.pixels(cur.v + q)
	if absI32(vvv-cur.vv) > maxdrift {
		switch {
		case vvv > cur.vv:
			cur.vv = vvv - maxdrift
		default:
			cur.vv = vvv + maxdrift
		}
	}
	cur.v += q
	m.printf(" v:=%d%+d=%d, vv:=%d", old, q, cur.v, cur.vv)
	return nil
}

func roundF32(v float32) int32 {
	if v > 0 {
		return int32(v + 0.5)
	}
	return int32(v - 0.5)
}

func absI32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func (m *Machine) pixels(v int32) int32 {
	x := m.conv * float32(v)
	return roundF32(x)
}

func (m *Machine) rulepixels(v int32) int32 {
	x := int32(m.conv * float32(v))
	if float32(x) < m.conv*float32(v) {
		return x + 1
	}
	return x
}

func (m *Machine) defineFont(i int, def fntdef) error {
	var (
		scale  = def.Size
		design = def.Design
		mag    int32
	)

	switch {
	case scale <= 0 || design <= 0:
		mag = 1000
	default:
		mag = roundF32((1000 * m.conv * float32(scale)) / (m.trueConv * float32(design)))
	}

	// design = roundF32((100 * m.conv * float32(scale)) / (m.trueConv * float32(design)))

	def.mag = mag
	def.Size = scale
	m.state.fonts[i] = def
	_, err := m.loadFont(i)
	if err != nil {
		return fmt.Errorf("could not load font %q: %w", def.Name, err)
	}
	return nil
}

func (m *Machine) loadFont(i int) (*Font, error) {
	def := m.state.fonts[i]
	if def.font != nil {
		return def.font, nil
	}

	// FIXME: add support for non-TFM fonts?
	name, err := m.ktx.Find(def.Name + ".tfm")
	if err != nil {
		return nil, fmt.Errorf("could not find TFM font %q: %w", def.Name, err)
	}

	f, err := m.ktx.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open TFM font %q: %w", def.Name, err)
	}
	defer f.Close()

	font, err := tfm.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("could not parse TFM font %q: %w", def.Name, err)
	}

	def.font = &Font{
		name:  def.Name,
		font:  font,
		scale: fixed.Int12_20(def.Size),
	}
	m.state.fonts[i] = def

	return def.font, nil
}

// func (m *Machine) font(i int) *Font {
// 	fnt, err := m.loadFont(i)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return fnt
// }

func (m *Machine) outText(c uint8) {
	if len(m.buf) == cap(m.buf) {
		m.printf("[%s]\n", m.buf)
		m.buf = m.buf[:0]
	}
	m.buf = append(m.buf, c)
}

func (m *Machine) flushText() {
	if len(m.buf) == 0 {
		return
	}
	m.printf("[%s]\n", m.buf)
	m.buf = m.buf[:0]

}

func (m *Machine) outSpace(v int32) int32 {
	def := m.state.fonts[m.state.f]
	space := def.Size / 6 // this is a 3-unit "thin space"
	cur := m.state.cur()
	switch {
	case v >= space || v <= -4*space:
		m.outText(' ')
		cur.hh = m.pixels(cur.h + v)
	default:
		cur.hh += m.pixels(v)
	}
	return cur.hh
}

func (m *Machine) outVMove(v int32) int32 {
	def := m.state.fonts[m.state.f]
	space := def.Size / 6 // this is a 3-unit "thin space"
	cur := m.state.cur()
	switch {
	case absI32(v) >= 5*space:
		cur.vv = m.pixels(cur.v + v)
	default:
		cur.vv += m.pixels(v)
	}
	return cur.vv
}

func (m *Machine) handleSpecial(p []byte) error {
	for _, h := range m.handlers {
		err := h.Handle(p)
		if err == nil || !errors.Is(err, ErrSkipHandler) {
			return err
		}
	}
	return nil
}

func (m *Machine) printf(format string, args ...interface{}) {
	fmt.Fprintf(m.w, format, args...)
}
