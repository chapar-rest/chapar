// Copyright ©2021 The star-tex Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package afm

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"star-tex.org/x/tex/font/fixed"
)

type parser struct {
	s    *bufio.Scanner
	err  error
	line int
	toks []string
}

func newParser(r io.Reader) *parser {
	return &parser{s: bufio.NewScanner(r)}
}

//TODO(sbinet): "BlendAxisTypes"
//TODO(sbinet): "BlendDesignPositions"
//TODO(sbinet): "BlendDesignMap"
//TODO(sbinet): "WeightVector"

func (p *parser) parse(fnt *Font) error {
loop:
	for p.scan() {
		switch p.toks[0] {
		case "StartFontMetrics":
			// ok.
		case "MetricsSets":
			fnt.metricsSets = p.readInt(1)
		case "Comment":
			// ignore.
		case "FontName":
			fnt.fontName = p.readStr(1)
		case "FullName":
			fnt.fullName = p.readStr(1)
		case "FamilyName":
			fnt.familyName = p.readStr(1)
		case "Weight":
			fnt.weight = p.readStr(1)
		case "FontBBox":
			fnt.bbox.llx = p.readFixed(1)
			fnt.bbox.lly = p.readFixed(2)
			fnt.bbox.urx = p.readFixed(3)
			fnt.bbox.ury = p.readFixed(4)
		case "Version":
			fnt.version = p.readStr(1)
		case "Notice":
			fnt.notice = p.readStr(1)
		case "EncodingScheme":
			fnt.encodingScheme = p.readStr(1)
		case "MappingScheme":
			fnt.mappingScheme = p.readInt(1)
		case "EscChar":
			fnt.escChar = p.readInt(1)
		case "CharacterSet":
			fnt.characterSet = p.readStr(1)
		case "Characters":
			fnt.characters = p.readInt(1)
		case "IsBaseFont":
			fnt.isBaseFont = p.readBool(1)
		case "VVector":
			fnt.vvector[0] = p.readFixed(1)
			fnt.vvector[1] = p.readFixed(2)
		case "IsFixedV":
			fnt.isFixedV = p.readBool(1)
		case "IsCIDFont":
			fnt.isCIDFont = p.readBool(1)
		case "CapHeight":
			fnt.capHeight = p.readFixed(1)
		case "XHeight":
			fnt.xHeight = p.readFixed(1)
		case "Ascender":
			fnt.ascender = p.readFixed(1)
		case "Descender":
			fnt.descender = p.readFixed(1)
		case "StdHW":
			fnt.stdHW = p.readFixed(1)
		case "StdVW":
			fnt.stdVW = p.readFixed(1)

		case "UnderlinePosition":
			fnt.direction[0].underlinePosition = p.readFixed(1)
		case "UnderlineThickness":
			fnt.direction[0].underlineThickness = p.readFixed(1)
		case "ItalicAngle":
			fnt.direction[0].italicAngle = p.readFixed(1)
		case "CharWidth":
			fnt.direction[0].charWidth.x = p.readFixed(1)
			fnt.direction[0].charWidth.y = p.readFixed(2)
		case "IsFixedPitch":
			fnt.direction[0].isFixedPitch = p.readBool(1)

		case "StartDirection":
			i := p.readInt(1)
			err := p.parseDirection(&fnt.direction[i])
			if err != nil {
				return fmt.Errorf("could not scan AFM Direction section: %w", err)
			}

		case "StartAxis":
			err := p.parseAxis(fnt)
			if err != nil {
				return fmt.Errorf("could not scan AFM Axis section: %w", err)
			}

		case "StartCharMetrics":
			err := p.parseCharMetrics(fnt, p.readInt(1))
			if err != nil {
				return fmt.Errorf("could not scan AFM CharMetrics section: %w", err)
			}
		case "StartKernData":
			err := p.parseKernData(fnt)
			if err != nil {
				return fmt.Errorf("could not scan AFM KernData section: %w", err)
			}
		case "StartComposites":
			err := p.parseComposites(fnt, p.readInt(1))
			if err != nil {
				return fmt.Errorf("could not scan AFM Composites section: %w", err)
			}
		case "EndFontMetrics":
			break loop
		default:
			log.Printf("invalid FontMetrics token %q (line=%d)", p.toks[0], p.line)
		}
	}

	if p.err != nil {
		return fmt.Errorf("could not parse AFM file: %w", p.err)
	}

	if err := p.s.Err(); err != nil {
		return fmt.Errorf("could not parse AFM file: %w", p.err)
	}

	return nil
}

func (p *parser) parseDirection(dir *direction) error {
	for p.scan() {
		switch p.toks[0] {
		case "EndDirection":
			return nil
		case "Comment":
			// ignore.
		case "UnderlinePosition":
			dir.underlinePosition = p.readFixed(1)
		case "UnderlineThickness":
			dir.underlineThickness = p.readFixed(1)
		case "ItalicAngle":
			dir.italicAngle = p.readFixed(1)
		case "CharWidth":
			dir.charWidth.x = p.readFixed(1)
			dir.charWidth.y = p.readFixed(2)
		case "IsFixedPitch":
			dir.isFixedPitch = p.readBool(1)
		default:
			return fmt.Errorf("invalid Direction token %q", p.toks[0])
		}
	}
	return p.err
}

func (p *parser) parseAxis(fnt *Font) error {
	// FIXME(sbinet)
	if true {
		return fmt.Errorf("axis section not supported")
	}

	for p.scan() {
		switch p.toks[0] {
		case "EndAxis":
			return nil
		case "Comment":
			// ignore.
		default:
			return fmt.Errorf("invalid Axis token %q", p.toks[0])
		}
	}
	return p.err
}

func (p *parser) parseCharMetrics(fnt *Font, n int) error {
	fnt.charMetrics = make([]CharMetric, 0, n)
	for p.scan() {
		switch p.toks[0] {
		case "EndCharMetrics":
			return nil
		case "Comment":
			// ignore.
		case "C", "CH",
			"WX", "W0X", "W1X",
			"WY", "W0Y", "W1Y",
			"W", "W0", "W1",
			"VV",
			"N",
			"B",
			"L":
			err := p.parseCharMetric(fnt)
			if err != nil {
				return fmt.Errorf("could not parse CharMetric entry: %w", err)
			}
		default:
			return fmt.Errorf("invalid CharMetrics token %q", p.toks[0])
		}
	}
	return p.err
}

func (p *parser) parseCharMetric(fnt *Font) error {
	ch := CharMetric{c: -1}
	for _, v := range strings.Split(p.s.Text(), ";") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		p.toks = strings.Fields(v)
		switch p.toks[0] {
		case "C":
			ch.c = p.readInt(1)
		case "CH":
			ch.c = p.readHex(1)
		case "WX", "W0X":
			ch.w0.x = p.readFixed(1)
			ch.w0.y = 0
		case "W1X":
			ch.w1.x = p.readFixed(1)
			ch.w1.y = 0
		case "WY", "W0Y":
			ch.w0.x = 0
			ch.w0.y = p.readFixed(1)
		case "W1Y":
			ch.w1.x = 0
			ch.w1.y = p.readFixed(1)
		case "W", "W0":
			ch.w0.x = p.readFixed(1)
			ch.w0.y = p.readFixed(2)
		case "W1":
			ch.w1.x = p.readFixed(1)
			ch.w1.y = p.readFixed(2)
		case "VV":
			ch.vv[0] = p.readFixed(1)
			ch.vv[1] = p.readFixed(2)
		case "N":
			ch.name = p.readStr(1)
		case "B":
			ch.bbox.llx = p.readFixed(1)
			ch.bbox.lly = p.readFixed(2)
			ch.bbox.urx = p.readFixed(3)
			ch.bbox.ury = p.readFixed(4)
		case "L":
			ch.ligs = append(ch.ligs, lig{
				succ: p.readStr(1),
				name: p.readStr(2),
			})
		}
	}
	fnt.charMetrics = append(fnt.charMetrics, ch)
	return p.err
}

func (p *parser) parseKernData(fnt *Font) error {
	for p.scan() {
		switch p.toks[0] {
		case "EndKernData":
			return nil
		case "Comment":
			// ignore.
		case "StartKernPairs", "StartKernPairs0":
			err := p.parseKernPairs(fnt, p.readInt(1))
			if err != nil {
				return fmt.Errorf("could not scan AFM KernPairs section: %w", err)
			}
		case "StartKernPairs1":
			return fmt.Errorf("KernPairs in direction 1 not supported")
		case "StartTrackKern":
			err := p.parseTrackKern(fnt, p.readInt(1))
			if err != nil {
				return fmt.Errorf("could not scan AFM KernPairs section: %w", err)
			}
		default:
			return fmt.Errorf("invalid KernData token %q", p.toks[0])
		}
	}
	return p.err
}

func (p *parser) parseKernPairs(fnt *Font, n int) error {
	fnt.pkerns = make([]kernPair, 0, n)
	for p.scan() {
		switch p.toks[0] {
		case "EndKernPairs":
			return nil
		case "Comment":
			// ignore.
		case "KP":
			fnt.pkerns = append(fnt.pkerns, kernPair{
				n1: p.readStr(1),
				n2: p.readStr(2),
				x:  p.readFixed(3),
				y:  p.readFixed(4),
			})
		case "KPH":
			fnt.pkerns = append(fnt.pkerns, kernPair{
				n1: string(rune(p.readHex(1))),
				n2: string(rune(p.readHex(2))),
				x:  p.readFixed(3),
				y:  p.readFixed(4),
			})
		case "KPX":
			fnt.pkerns = append(fnt.pkerns, kernPair{
				n1: p.readStr(1),
				n2: p.readStr(2),
				x:  p.readFixed(3),
				y:  0,
			})
		case "KPY":
			fnt.pkerns = append(fnt.pkerns, kernPair{
				n1: p.readStr(1),
				n2: p.readStr(2),
				x:  0,
				y:  p.readFixed(3),
			})
		default:
			return fmt.Errorf("invalid KernPairs token %q", p.toks[0])
		}
	}
	return p.err
}

func (p *parser) parseTrackKern(fnt *Font, n int) error {
	fnt.tkerns = make([]trackKern, 0, n)
	for p.scan() {
		switch p.toks[0] {
		case "EndTrackKern":
			return nil
		case "Comment":
			// ignore.
		case "TrackKern":
			fnt.tkerns = append(fnt.tkerns, trackKern{
				degree:    p.readInt(1),
				minPtSize: p.readFixed(2),
				minKern:   p.readFixed(3),
				maxPtSize: p.readFixed(4),
				maxKern:   p.readFixed(5),
			})
		default:
			return fmt.Errorf("invalid TrackKern token %q", p.toks[0])
		}
	}
	return p.err
}

func (p *parser) parseComposites(fnt *Font, n int) error {
	fnt.composites = make([]composite, 0, n)
	for p.scan() {
		switch p.toks[0] {
		case "EndComposites":
			return nil
		case "Comment":
			// ignore.
		case "CC":
			err := p.parseComposite(fnt)
			if err != nil {
				return fmt.Errorf("could not parse Composite: %w", err)
			}
		default:
			return fmt.Errorf("invalid Composite token %q", p.toks[0])
		}
	}
	return p.err
}

func (p *parser) parseComposite(fnt *Font) error {
	var comp composite
	for _, v := range strings.Split(p.s.Text(), ";") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		p.toks = strings.Fields(v)
		switch p.toks[0] {
		case "CC":
			comp.name = p.readStr(1)
			comp.parts = make([]part, 0, p.readInt(2))
		case "PCC":
			comp.parts = append(comp.parts, part{
				name: p.readStr(1),
				x:    p.readFixed(2),
				y:    p.readFixed(3),
			})
		}
	}
	return p.err
}

func (p *parser) scan() bool {
	if p.err != nil {
		return false
	}
	p.line++
	ok := p.s.Scan()
	p.toks = strings.Fields(strings.TrimSpace(p.s.Text()))
	if ok && len(p.toks) == 0 {
		// skip empty lines.
		return p.scan()
	}
	p.err = p.s.Err()
	return ok
}

func (p *parser) readStr(i int) string {
	if len(p.toks) <= i {
		return ""
	}
	return p.toks[i]
}

func (p *parser) readInt(i int) int {
	if len(p.toks) <= i {
		return 0
	}
	return atoi(p.toks[i])
}

func (p *parser) readHex(i int) int {
	if len(p.toks) <= i {
		return 0
	}
	return atoHex(p.toks[i])
}

func (p *parser) readBool(i int) bool {
	if len(p.toks) <= i {
		return false
	}
	return atob(p.toks[i])
}

func (p *parser) readFixed(i int) fixed.Int16_16 {
	if len(p.toks) <= i {
		return 0
	}
	return fixedFrom(p.toks[i])
}

func fixedFrom(v string) fixed.Int16_16 {
	v = strings.Replace(v, ",", ".", 1)
	o, err := fixed.ParseInt16_16(v)
	if err != nil {
		panic(err)
	}
	return o
}

func atoi(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return v
}

func atoHex(s string) int {
	s = strings.Trim(s, "<>")
	hex, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		panic(err)
	}
	return int(hex)
}

func atob(s string) bool {
	switch s {
	case "true":
		return true
	case "false":
		return false
	default:
		panic(fmt.Errorf("invalid boolean value %q", s))
	}
}
