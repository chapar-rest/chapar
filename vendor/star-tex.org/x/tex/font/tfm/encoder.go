// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tfm

import (
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"

	"star-tex.org/x/tex/font/fixed"
)

type label struct {
	cc int16
	rr int
}

const (
	ligsize = 5000

	actUnreachable = 0
	actPassthrough = 1
	actAccessible  = 2
)

var (
	// ascii04 = []byte(" !\"#$%&'()*+,-./0123456789:;<=>?")
	ascii10 = []byte("@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_")
	ascii14 = []byte("`abcdefghijklmnopqrstuvwxyz{|}~ ")
)

type textEncoder struct {
	w   io.Writer
	err error
	lvl int

	fntType      int
	labels       []label
	act          []uint8
	boundarychar int16
}

func newTextEncoder(w io.Writer) *textEncoder {
	return &textEncoder{
		w: w,

		fntType:      0,
		labels:       make([]label, 0, 259),
		act:          make([]uint8, ligsize),
		boundarychar: 256,
	}
}

func (te *textEncoder) indent() {
	for i := 0; i < te.lvl; i++ {
		fmt.Fprintf(te.w, "   ")
	}
}

func (te *textEncoder) left() {
	te.indent()
	te.inc()
	fmt.Fprintf(te.w, "(")
}
func (te *textEncoder) right() { fmt.Fprintf(te.w, ")\n"); te.dec() }
func (te *textEncoder) inc()   { te.lvl++ }
func (te *textEncoder) dec()   { te.lvl-- }

func (te *textEncoder) line(vs ...string) {
	te.left()
	fmt.Fprint(te.w, strings.Join(vs, " "))
	te.right()
}

func (te *textEncoder) bcpl(s string) string {
	return s // FIXME(sbinet)
}

func (te *textEncoder) octU32(v uint32) string {
	return fmt.Sprintf("O %o", v)
}

func (te *textEncoder) face(v byte) string {
	if v >= 18 {
		return fmt.Sprintf("O %o", v)
	}
	const (
		mblString = "MBL"
		riString  = "RI "
		rceString = "RCE"
	)

	s := v & 1
	b := v / 2
	return "F " + string(mblString[b%3]) + string(riString[s]) + string(rceString[b/3])
}

func (te *textEncoder) fword(v fixed.Int12_20) string {
	var (
		j     byte
		delta = int32(10)
		raw   [4]byte
		str   string
		dig   = make([]byte, 0, 12)
	)
	binary.BigEndian.PutUint32(raw[:], uint32(v))

	a := int16(raw[0])*16 + int16(raw[1])/16
	f := int32(int(raw[1]&15)*256+int(raw[2]))*256 + int32(raw[3])
	if a > 2047 {
		str = "-"
		a = 4096 - a
		if f > 0 {
			f = 1048576 - f
			a--
		}
	}

	for {
		dig = append(dig, 48+byte(a%10))
		a /= 10
		j++
		if a == 0 {
			break
		}
	}

	dig = dig[:j]
	for i := len(dig)/2 - 1; i >= 0; i-- {
		opp := len(dig) - 1 - i
		dig[i], dig[opp] = dig[opp], dig[i]
	}

	str += string(dig[:j]) + "."
	f = (f * 10) + 5
	for f > delta {
		if delta > 1048576 {
			f += 524288 - (delta / 2)
		}
		str += fmt.Sprintf("%d", f/1048576)
		f = (f & 1048575) * 10
		delta *= 10
	}
	if strings.HasSuffix(str, ".") {
		str += "0"
	}

	return "R " + str
}

func (te *textEncoder) char(v byte) string {
	if te.fntType > 0 {
		return fmt.Sprintf(" O %o", v)
	}
	switch {
	case 48 <= v && v <= 57:
		return fmt.Sprintf(" C %d", v-48)
	case 65 <= v && v <= 90:
		return fmt.Sprintf(" C %c", ascii10[v-64])
	case 97 <= v && v <= 122:
		return fmt.Sprintf(" C %c", ascii14[v-96])
	default:
		return fmt.Sprintf(" O %o", v)
	}
}

func (te *textEncoder) encodeStr(s string) {
	fmt.Fprint(te.w, s)
}

func (te *textEncoder) encode(fnt *Font) error {
	te.encodeHeader(fnt)
	te.encodeChars(fnt)
	return te.err
}

func (te *textEncoder) encodeHeader(fnt *Font) {
	if te.err != nil {
		return
	}

	if fnt.hdr.lh < 12 {
		return
	}

	scheme := strings.ToUpper(fnt.body.header.codingScheme)
	fntType := fontFamily(scheme)
	switch fntType {
	case fontTypeMathSym:
		te.fntType = 1
	case fontTypeMathExt:
		te.fntType = 2
	default:
		fntType = fontTypeVanilla
	}

	if fnt.hdr.lh >= 17 {
		te.line("FAMILY", te.bcpl(strings.ToUpper(fnt.Name())))
		if fnt.hdr.lh >= 18 {
			te.line("FACE", te.face(fnt.body.header.face))
			for i, v := range fnt.body.header.extra {
				te.line(fmt.Sprintf("HEADER D %d", i+18), te.octU32(uint32(v)))
			}
			// for i := 18; i < int(fnt.hdr.lh); i++ {
			// 	te.line(fmt.Sprintf("HEADER D %d", i), te.octU32(uint32(i*4)+24))
			// }
		}
		te.line("CODINGSCHEME", te.bcpl(scheme))
	}
	te.line("DESIGNSIZE", te.fword(fnt.DesignSize()))
	te.line("COMMENT", "DESIGNSIZE IS IN POINTS")
	te.line("COMMENT", "OTHER SIZES ARE MULTIPLES OF DESIGNSIZE")

	te.line("CHECKSUM", te.octU32(fnt.body.header.chksum))
	if fnt.body.header.sevenBitSafe {
		te.line("SEVENBITSAFEFLAG", "TRUE")
	}

	te.encodeFontDimen(fnt, fntType)
	te.encodeLigTable(fnt)
}

func (te *textEncoder) encodeFontDimen(fnt *Font, fntType fontFamily) {
	if te.err != nil {
		return
	}

	if fnt.hdr.np == 0 {
		return
	}

	te.left()
	te.encodeStr("FONTDIMEN\n")
	names := []string{
		"SLANT", "SPACE", "STRETCH", "SHRINK",
		"XHEIGHT", "QUAD", "EXTRASPACE",
	}
	switch n := len(fnt.body.param); {
	case n <= 7:
		// ok.
	case n <= 22 && fntType == fontTypeMathSym:
		names = append(names,
			"NUM1", "NUM2", "NUM3",
			"DENOM1", "DENOM2",
			"SUP1", "SUP2", "SUP3",
			"SUB1", "SUB2",
			"SUPDROP", "SUBDROP",
			"DELIM1", "DELIM2",
			"AXISHEIGHT",
		)
	case n <= 13 && fntType == fontTypeMathExt:
		names = append(names,
			"DEFAULTRULETHICKNESS",
			"BIGOPSPACING1",
			"BIGOPSPACING2",
			"BIGOPSPACING3",
			"BIGOPSPACING4",
			"BIGOPSPACING5",
		)
	default:
		panic("invalid np")
	}

	for i, p := range fnt.body.param {
		name := names[i]
		te.line(name, te.fword(p))
	}
	te.indent()
	te.right()

	switch {
	case fntType == fontTypeMathSym && fnt.hdr.np != 22:
		panic("invalid math-sym TFM")
	case fntType == fontTypeMathExt && fnt.hdr.np != 13:
		panic("invalid math-ext TFM")
	}
}

func (te *textEncoder) encodeLigTable(fnt *Font) {
	if te.err != nil {
		return
	}

	if fnt.hdr.nl == 0 {
		return
	}
	if lk := fnt.body.ligKern[0]; lk.raw[0] == 255 {
		te.boundarychar = int16(lk.raw[1])
		te.line("BOUNDARYCHAR" + te.char(byte(te.boundarychar)))
		te.act[0] = actPassthrough
	}

	te.buildLabels(fnt)

	te.left()
	te.encodeStr("LIGTABLE\n")
	il := 0
	for i, lk := range fnt.body.ligKern {
		for i == int(te.labels[il].rr) {
			str := ""
			switch cc := te.labels[il].cc; cc {
			case 256:
				str = " BOUNDARYCHAR"
			default:
				str = te.char(byte(cc))
			}
			te.line("LABEL" + str)
			il++
		}
		switch {
		case lk.raw[0] > 128:
			// FIXME(sbinet): check unconditional stop command.
		case lk.op() == krnCmd:
			line := "KRN" + te.char(byte(lk.nextChar()))
			ii := lk.nextIndex()
			vv := fnt.body.kern[ii]
			line += " " + te.fword(vv)
			te.line(line)
		case lk.op() == ligCmd:
			nxt := lk.nextChar()
			switch {
			case int(fnt.hdr.bc) <= nxt && nxt < int(fnt.hdr.ec)+1:
				//				if int(nxt) != int(te.boundarychar) {
				//					panic(fmt.Errorf("not implemented: BOUNDARYCHAR (%d)", nxt))
				//				}
			case int(lk.raw[3]) < int(fnt.hdr.bc) || int(lk.raw[3]) > int(fnt.hdr.ec):
				// FIXME(sbinet): add recover procedure
				panic("not implemented: non-existent character")
			}
			r := lk.raw[2]
			if r == 4 || (r > 7 && r != 11) {
				// non standard code changed to LIG.
				r = 0
			}
			name := ""
			if r&3 > 1 {
				name += "/"
			}
			name += "LIG"
			if r&1 != 0 {
				name += "/"
			}
			for r > 3 {
				name += ">"
				r -= 4
			}
			te.line(name + te.char(byte(lk.raw[1])) + te.char(byte(lk.raw[3])))
		default:
			panic("impossible")
		}
		if lk.raw[0] > 0 {
			switch {
			case lk.raw[0] >= 128:
				te.line("STOP")
			default:
				// FIXME(sbinet): skip.
				te.line("SKIP", "D", "???")
			}
		}
	}
	te.indent()
	te.right()
}

func (te *textEncoder) buildLabels(fnt *Font) {
	te.labels = te.labels[:0]
	for i, ci := range fnt.body.glyphs {
		kind, rem := ci.kind()
		if kind != gkLigKern {
			continue
		}
		if rem < len(fnt.body.ligKern) {
			lk := fnt.body.ligKern[rem]
			if lk.skipByte() {
				rem = int(lk.raw[2])*256 + int(lk.raw[3])
				if rem < len(fnt.body.ligKern) {
					if te.act[int(ci.raw[3])] == actUnreachable {
						te.act[int(ci.raw[3])] = actPassthrough
					}
				}
			}
		}
		if rem >= len(fnt.body.ligKern) {
			panic(fmt.Errorf(
				"ligature/kern starting index for character %q is too large",
				i,
			))
		}

		te.act[rem] = actAccessible
		te.labels = append(te.labels, label{
			cc: int16(i),
			rr: rem,
		})

	}

	sort.SliceStable(te.labels, func(i, j int) bool {
		return te.labels[i].rr < te.labels[j].rr
	})

	te.labels = append(te.labels, label{
		rr: ligsize,
	})
}

func (te *textEncoder) encodeChars(fnt *Font) {
	for i, g := range fnt.body.glyphs {
		x := byte(i + int(fnt.hdr.bc))
		te.left()
		te.encodeStr("CHARACTER" + te.char(x) + "\n")
		te.line("CHARWD", te.fword(fnt.body.width[g.wd()]))

		if i := g.ht(); i > 0 {
			te.line("CHARHT", te.fword(fnt.body.height[i]))
		}

		if i := g.dp(); i > 0 {
			te.line("CHARDP", te.fword(fnt.body.depth[i]))
		}

		if i := g.ic(); i > 0 {
			te.line("CHARIC", te.fword(fnt.body.italic[i]))
		}

		switch g.raw[2] & 3 {
		case 0:
			// blank case
		case 1:
			te.left()
			te.encodeStr("COMMENT\n")
			ii := int(g.raw[3])
			rr := fnt.body.ligKern[ii]
			if rr.raw[0] > 128 {
				ii = int(rr.raw[2])*256 + int(rr.raw[3])
			}
			for {
				lk := fnt.body.ligKern[ii]
				switch {
				case lk.raw[0] > 128:
					// FIXME(sinet): test for unconditional stop cmd address.
				case lk.raw[2] >= 128:
					line := "KRN" + te.char(byte(lk.raw[1]))
					rr := lk.nextIndex()
					if rr >= len(fnt.body.kern) {
						panic("Bad TFM file: Kern index too large.")
					}
					te.line(line, te.fword(fnt.body.kern[rr]))
				default:
					// FIXME(sbinet): test ligature step for non-existent char.
					// FIXME(sbinet): test lig-step producing non-existent char.
					r := int(lk.raw[2])
					if r == 4 || (r > 7 && r != 11) {
						r = 0
					}
					name := ""
					if r&3 > 1 {
						name += "/"
					}
					name += "LIG"
					if r&1 != 0 {
						name += "/"
					}
					for r > 3 {
						name += ">"
						r -= 4
					}
					te.line(name + te.char(byte(lk.raw[1])) + te.char(byte(lk.raw[3])))
				}

				switch {
				case lk.raw[0] >= 128:
					ii = len(fnt.body.ligKern)
				default:
					ii += int(lk.raw[0]) + 1
				}

				if ii >= len(fnt.body.ligKern) {
					break
				}
			}
			te.indent()
			te.right()

		case 2:
			// r := int(g.raw[3])
			// FIXME(sbinet): test character list to nonexistent character.

			// FIXME(sbinet): test cycle in character list.
			//	for r < i && g.raw[2]&3 == 2 {
			//		r = int(fnt.body.charInfos[r].raw[3])
			//	}
			//	if r == i {
			//		panic("Bad TFM file: cycle in a character list")
			//	}

			te.line("NEXTLARGER" + te.char(byte(g.raw[3])))
		case 3:
			if int(g.raw[3]) >= int(fnt.hdr.ne) {
				panic("Extensible index for character " + te.octU32(uint32(x)) + " is too large")
			}
			te.left()
			te.encodeStr("VARCHAR\n")
			for k := 0; k < 4; k++ {
				ik := int(g.raw[3])
				ext := fnt.body.exten[ik].raw[k]
				if !(k == 3 || ext > 0) {
					continue
				}
				name := [...]string{"TOP", "MID", "BOT", "REP"}[k]
				var v byte
				switch {
				case int(ext) < int(fnt.hdr.bc),
					int(ext) > int(fnt.hdr.ec),
					ext == 0:
					v = byte(g.raw[0]) // ? FIXME(sbinet)
				default:
					v = byte(ext)
				}
				te.line(name + te.char(v))
			}
			te.indent()
			te.right()
		}
		te.indent()
		te.right()
	}
}
