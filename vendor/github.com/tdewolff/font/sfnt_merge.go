package font

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/tdewolff/parse/v2"
)

type MergeOptions struct {
	RearrangeCmap bool // Rearrange glyph unicode mapping, assigning a sequential codepoint for each glyph in order starting at 33 (exclamation). May be useful when embedding into a PDF with a custom unicode mapping.
}

// Merge merges the glyphs of another font into the current one in-place by merging the glyf, loca, kern tables (or CFF table for CFF fonts), as well as the hmtx, cmap, and post tables. Also updates the maxp, head, and OS/2 tables. The other font remains untouched.
func (sfnt *SFNT) Merge(sfnt2 *SFNT, options MergeOptions) error {
	if sfnt.IsTrueType != sfnt2.IsTrueType || sfnt.IsCFF != sfnt2.IsCFF {
		return fmt.Errorf("can only merge two TrueType or two CFF fonts")
	} else if math.MaxUint16-sfnt.NumGlyphs() < sfnt2.NumGlyphs() {
		return fmt.Errorf("too many glyphs for one font")
	} else if sfnt2.NumGlyphs() < 2 {
		return nil // sfnt2 only contains .notdef
	}

	// update maxp table
	origNumGlyphs := sfnt.NumGlyphs()
	numGlyphs := origNumGlyphs + sfnt2.NumGlyphs() - 1 // -1 to skip .notdef
	if _, ok := sfnt.Tables["maxp"]; ok {
		sfnt.Maxp.NumGlyphs = numGlyphs
		if !sfnt.IsCFF {
			sfnt.Maxp.MaxPoints = max(sfnt.Maxp.MaxPoints, sfnt2.Maxp.MaxPoints)
			sfnt.Maxp.MaxContours = max(sfnt.Maxp.MaxContours, sfnt2.Maxp.MaxContours)
			sfnt.Maxp.MaxCompositePoints = max(sfnt.Maxp.MaxCompositePoints, sfnt2.Maxp.MaxCompositePoints)
			sfnt.Maxp.MaxCompositeContours = max(sfnt.Maxp.MaxCompositeContours, sfnt2.Maxp.MaxCompositeContours)
			sfnt.Maxp.MaxZones = max(sfnt.Maxp.MaxZones, sfnt2.Maxp.MaxZones)
			sfnt.Maxp.MaxTwilightPoints = max(sfnt.Maxp.MaxTwilightPoints, sfnt2.Maxp.MaxTwilightPoints)
			sfnt.Maxp.MaxStorage = max(sfnt.Maxp.MaxStorage, sfnt2.Maxp.MaxStorage)
			sfnt.Maxp.MaxFunctionDefs = max(sfnt.Maxp.MaxFunctionDefs, sfnt2.Maxp.MaxFunctionDefs)
			sfnt.Maxp.MaxInstructionDefs = max(sfnt.Maxp.MaxInstructionDefs, sfnt2.Maxp.MaxInstructionDefs)
			sfnt.Maxp.MaxStackElements = max(sfnt.Maxp.MaxStackElements, sfnt2.Maxp.MaxStackElements)
			sfnt.Maxp.MaxSizeOfInstructions = max(sfnt.Maxp.MaxSizeOfInstructions, sfnt2.Maxp.MaxSizeOfInstructions)
			sfnt.Maxp.MaxComponentElements = max(sfnt.Maxp.MaxComponentElements, sfnt2.Maxp.MaxComponentElements)
			sfnt.Maxp.MaxComponentDepth = max(sfnt.Maxp.MaxComponentDepth, sfnt2.Maxp.MaxComponentDepth)
		}
		sfnt.Tables["maxp"] = sfnt.Maxp.Write()
	}

	// update glyf/loca or CFF tables
	if sfnt.IsTrueType {
		glyf, ok1 := sfnt.Tables["glyf"]
		glyf2, ok2 := sfnt2.Tables["glyf"]
		loca, ok3 := sfnt.Tables["loca"]
		_, ok4 := sfnt2.Tables["loca"]
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return fmt.Errorf("glyf of loca tables missing")
		}

		// skip .notdef
		firstGlyf, _ := sfnt2.Loca.Get(1)
		glyf2 = glyf2[firstGlyf:]

		// update glyphIDs of composite glyphs and make sure not to write to glyf2
		startGlyf := uint32(len(glyf))
		glyf = append(glyf, glyf2...)
		for glyphID := 1; glyphID < int(sfnt2.NumGlyphs()); glyphID++ {
			// +1 to skip .notdef
			if sfnt2.Glyf.IsComposite(uint16(glyphID)) {
				start, _ := sfnt2.Loca.Get(uint16(glyphID))
				start += startGlyf - firstGlyf

				offset := uint32(10)
				for {
					flags := binary.BigEndian.Uint16(glyf[start+offset:])
					subGlyphID := binary.BigEndian.Uint16(glyf[start+offset+2:])
					binary.BigEndian.PutUint16(glyf[start+offset+2:], origNumGlyphs+subGlyphID-1)

					length, more := glyfCompositeLength(flags)
					if !more {
						break
					}
					offset += length
				}
			}
		}
		sfnt.Tables["glyf"] = glyf
		sfnt.Glyf.data = glyf

		if _, ok := sfnt.Tables["loca"]; ok {
			indexToLocFormat := int16(1) // long format
			if len(glyf) <= math.MaxUint16 {
				indexToLocFormat = 0 // short format
			}
			n := 2 * (numGlyphs + 1)
			if indexToLocFormat == 1 {
				n *= 2
			}

			// write current offsets
			glyphID := 0
			w := parse.NewBinaryWriter(make([]byte, 0, n))
			if indexToLocFormat == sfnt.Loca.Format {
				w.WriteBytes(loca)
				glyphID = int(origNumGlyphs)
			} else if table, ok := sfnt.Tables["head"]; ok {
				binary.BigEndian.PutUint16(table[50:], uint16(indexToLocFormat))
				sfnt.Head.IndexToLocFormat = indexToLocFormat
			}

			// write offsets (+1 since we do iterate over .notdef, but skip it below)
			for ; glyphID < int(numGlyphs); glyphID++ {
				var offset uint32
				if glyphID < int(origNumGlyphs) {
					offset, _ = sfnt.Loca.Get(uint16(glyphID))
				} else {
					// +1 to skip .notdef, +1 to get the end of the glyph
					offset, _ = sfnt2.Loca.Get(uint16(1 + glyphID - int(origNumGlyphs) + 1))
					offset += startGlyf - firstGlyf
				}

				if indexToLocFormat == 0 {
					// short format
					w.WriteUint16(uint16(offset / 2))
				} else {
					// long format
					w.WriteUint32(offset)
				}
			}

			sfnt.Tables["loca"] = w.Bytes()
			sfnt.Loca.Format = indexToLocFormat
			sfnt.Loca.data = w.Bytes()
		}

		if _, ok := sfnt.Tables["kern"]; ok {
			// update glyph IDs in kern table
			for _, subtable := range sfnt2.Kern.Subtables {
				offset := uint32(origNumGlyphs-1)<<16 + uint32(origNumGlyphs-1) // -1 to skip .notdef
				pairs := make([]kernPair, len(subtable.Pairs))
				for i, pair := range subtable.Pairs {
					if pair.Key&0x0F == 0 || pair.Key&0xF0 == 0 {
						// skip .notdef
						continue
					}
					pairs[i] = kernPair{
						Key:   offset + pair.Key,
						Value: pair.Value,
					}
				}
				sfnt.Kern.Subtables = append(sfnt.Kern.Subtables, kernFormat0{
					Coverage: subtable.Coverage,
					Pairs:    pairs,
				})
			}
			if 0 < len(sfnt.Kern.Subtables) {
				sfnt.Tables["kern"] = sfnt.Kern.Write()
			}
		}
	} else if _, ok := sfnt.Tables["CFF "]; ok && sfnt.IsCFF {
		// find new number of subroutines, which may change the index encoding / bias
		localSubrsLen, globalSubrsLen := 0, 0 // new length
		localSubrs, localSubrs2 := &cffINDEX{}, &cffINDEX{}
		globalSubrs, globalSubrs2 := &cffINDEX{}, &cffINDEX{}
		if 0 < len(sfnt.CFF.fonts.localSubrs) && 0 < sfnt.CFF.fonts.localSubrs[0].Len() {
			*localSubrs = *sfnt.CFF.fonts.localSubrs[0]
			localSubrsLen += sfnt.CFF.fonts.localSubrs[0].Len()
		}
		if 0 < len(sfnt2.CFF.fonts.localSubrs) && 0 < sfnt2.CFF.fonts.localSubrs[0].Len() {
			*localSubrs2 = *sfnt2.CFF.fonts.localSubrs[0]
			localSubrsLen += sfnt2.CFF.fonts.localSubrs[0].Len()
		}
		if 0 < sfnt.CFF.globalSubrs.Len() {
			*globalSubrs = *sfnt.CFF.globalSubrs
			globalSubrsLen += sfnt.CFF.globalSubrs.Len()
		}
		if 0 < sfnt2.CFF.globalSubrs.Len() {
			*globalSubrs2 = *sfnt2.CFF.globalSubrs
			globalSubrsLen += sfnt2.CFF.globalSubrs.Len()
		}

		// update index encoding (identity mapping) for the original glyphs
		var localSubrsMap map[int32]int32  // old to new index
		var globalSubrsMap map[int32]int32 // old to new index
		if cffNumberSize(localSubrsLen) != cffNumberSize(localSubrs.Len()) {
			localSubrsMap = make(map[int32]int32, localSubrs.Len())
			for i := 0; i < localSubrs.Len(); i++ {
				localSubrsMap[int32(i)] = int32(i)
			}
		}
		if cffNumberSize(globalSubrsLen) != cffNumberSize(globalSubrs.Len()) {
			globalSubrsMap = make(map[int32]int32, globalSubrs.Len())
			for i := 0; i < globalSubrs.Len(); i++ {
				globalSubrsMap[int32(i)] = int32(i)
			}
		}

		if err := sfnt.CFF.updateSubrs(localSubrsMap, globalSubrsMap, localSubrs, globalSubrs, localSubrsLen, globalSubrsLen); err != nil {
			return err
		}

		// update indices and index encoding for the merging glyphs
		var localSubrsMap2 map[int32]int32  // old to new index
		var globalSubrsMap2 map[int32]int32 // old to new index
		if 0 < localSubrs.Len() || cffNumberSize(localSubrsLen) != cffNumberSize(localSubrs2.Len()) {
			offset := localSubrs.Len()
			localSubrsMap2 = make(map[int32]int32, localSubrs2.Len())
			for i := 0; i < localSubrs2.Len(); i++ {
				localSubrsMap2[int32(i)] = int32(offset + i)
			}
		}
		if 0 < globalSubrs.Len() || cffNumberSize(globalSubrsLen) != cffNumberSize(globalSubrs2.Len()) {
			offset := globalSubrs.Len()
			globalSubrsMap2 = make(map[int32]int32, globalSubrs2.Len())
			for i := 0; i < globalSubrs2.Len(); i++ {
				globalSubrsMap2[int32(i)] = int32(offset + i)
			}
		}

		// copy to keep original untouched
		cff2 := &cffTable{
			version:     sfnt2.CFF.version,
			charStrings: sfnt2.CFF.charStrings.Copy(),
			globalSubrs: sfnt2.CFF.globalSubrs,
			fonts:       sfnt2.CFF.fonts,
		}

		// remove .notdef
		offset := cff2.charStrings.offset[1]
		cff2.charStrings.offset = cff2.charStrings.offset[1:]
		cff2.charStrings.data = cff2.charStrings.data[offset:]
		for i := range cff2.charStrings.offset {
			cff2.charStrings.offset[i] -= offset
		}

		// update subroutine indices for merging font
		if err := cff2.updateSubrs(localSubrsMap2, globalSubrsMap2, localSubrs, globalSubrs, localSubrsLen, globalSubrsLen); err != nil {
			return err
		}

		// merge charStrings
		sfnt.CFF.charStrings.Extend(cff2.charStrings)
		sfnt.CFF.globalSubrs = globalSubrs
		sfnt.CFF.fonts.localSubrs = []*cffINDEX{localSubrs}

		// update charset
		if sfnt2.CFF.charset != nil {
			if sfnt.CFF.charset == nil {
				numGlyphs := sfnt.CFF.charStrings.Len()
				for i := 0; i < numGlyphs; i++ {
					sfnt.CFF.charset = append(sfnt.CFF.charset, "")
				}
			}
			sfnt.CFF.charset = append(sfnt.CFF.charset, sfnt2.CFF.charset[1:]...)
		} else if sfnt.CFF.charset != nil {
			numGlyphs := sfnt2.CFF.charStrings.Len() - 1
			for i := 0; i < numGlyphs; i++ {
				sfnt.CFF.charset = append(sfnt.CFF.charset, "")
			}
		}

		b, err := sfnt.CFF.Write()
		if err != nil {
			return err
		}
		sfnt.Tables["CFF "] = b
	}

	if _, ok := sfnt.Tables["hmtx"]; ok {
		lsbs := make([]int16, numGlyphs)
		advances := make([]uint16, numGlyphs)
		for glyphID := 0; glyphID < int(numGlyphs); glyphID++ {
			if glyphID < int(origNumGlyphs) {
				lsbs[glyphID] = sfnt.Hmtx.LeftSideBearing(uint16(glyphID))
				advances[glyphID] = sfnt.Hmtx.Advance(uint16(glyphID))
			} else {
				// +1 to skip .notdef
				lsbs[glyphID] = sfnt2.Hmtx.LeftSideBearing(uint16(1+glyphID) - origNumGlyphs)
				advances[glyphID] = sfnt2.Hmtx.Advance(uint16(1+glyphID) - origNumGlyphs)
			}
		}
		numberOfHMetrics := numGlyphs
		for 1 < numberOfHMetrics {
			if advances[numberOfHMetrics-1] != advances[numberOfHMetrics-2] {
				break
			}
			numberOfHMetrics--
		}

		sfnt.Hmtx = &hmtxTable{}
		sfnt.Hmtx.HMetrics = make([]hmtxLongHorMetric, numberOfHMetrics)
		sfnt.Hmtx.LeftSideBearings = lsbs[numberOfHMetrics:]

		n := 4*int(numberOfHMetrics) + 2*(int(numGlyphs)-int(numberOfHMetrics))
		w := parse.NewBinaryWriter(make([]byte, 0, n))
		for glyphID := 0; glyphID < int(numGlyphs); glyphID++ {
			if glyphID < int(numberOfHMetrics) {
				sfnt.Hmtx.HMetrics[glyphID].AdvanceWidth = advances[glyphID]
				sfnt.Hmtx.HMetrics[glyphID].LeftSideBearing = lsbs[glyphID]
				w.WriteUint16(advances[glyphID])
			}
			w.WriteInt16(lsbs[glyphID])
		}
		sfnt.Tables["hmtx"] = w.Bytes()

		if table, ok := sfnt.Tables["hhea"]; ok {
			binary.BigEndian.PutUint16(table[34:], numberOfHMetrics) // numberOfHMetrics
			sfnt.Hhea.NumberOfHMetrics = numberOfHMetrics
		}
	}

	if _, ok := sfnt.Tables["cmap"]; ok {
		rs := make([]rune, 0, numGlyphs)
		runeMap := make(map[rune]uint16, numGlyphs) // for OS/2
		if options.RearrangeCmap {
			rs = append(rs, 65535) // .notdef
			runeMap[65535] = 0
			for glyphID := 1; glyphID < int(numGlyphs); glyphID++ {
				r := rune(glyphID + 32) // unicode codepoints below 33 don't display in firefox
				rs = append(rs, r)
				runeMap[r] = uint16(glyphID)
			}
		} else {
			for glyphID := 0; glyphID < int(numGlyphs); glyphID++ {
				var rsGlyph []rune
				if glyphID < int(origNumGlyphs) {
					rsGlyph = sfnt.Cmap.ToUnicode(uint16(glyphID))
				} else {
					rsGlyph = sfnt2.Cmap.ToUnicode(uint16(1+glyphID) - origNumGlyphs) // +1 to skip .notdef
				}
				rs = append(rs, rsGlyph...)
				for _, r := range rsGlyph {
					if otherGlyphID, ok := runeMap[r]; ok {
						glyphName, otherGlyphName := "", ""
						if glyphID < int(origNumGlyphs) {
							glyphName = sfnt.GlyphName(uint16(glyphID))
						} else {
							glyphName = sfnt2.GlyphName(uint16(1+glyphID) - origNumGlyphs)
						}
						if otherGlyphID < origNumGlyphs {
							otherGlyphName = sfnt.GlyphName(otherGlyphID)
						} else {
							otherGlyphName = sfnt2.GlyphName(1 + otherGlyphID - origNumGlyphs)
						}
						return fmt.Errorf("cmap: two or more glyphs have the same unicode mapping: %s(%d) and %s(%d)", glyphName, glyphID, otherGlyphName, otherGlyphID)
					}
					runeMap[r] = uint16(glyphID)
				}
			}
		}

		sfnt.Tables["cmap"] = cmapWrite(rs, runeMap)
		if err := sfnt.parseCmap(); err != nil {
			return err
		}
	}

	if table, ok := sfnt.Tables["OS/2"]; ok {
		sfnt.OS2.UlUnicodeRange1 |= sfnt2.OS2.UlUnicodeRange1
		sfnt.OS2.UlUnicodeRange2 |= sfnt2.OS2.UlUnicodeRange2
		sfnt.OS2.UlUnicodeRange3 |= sfnt2.OS2.UlUnicodeRange3
		sfnt.OS2.UlUnicodeRange4 |= sfnt2.OS2.UlUnicodeRange4
		binary.BigEndian.PutUint32(table[42:], sfnt.OS2.UlUnicodeRange1)
		binary.BigEndian.PutUint32(table[46:], sfnt.OS2.UlUnicodeRange2)
		binary.BigEndian.PutUint32(table[50:], sfnt.OS2.UlUnicodeRange3)
		binary.BigEndian.PutUint32(table[54:], sfnt.OS2.UlUnicodeRange4)
	}

	if _, ok := sfnt.Tables["post"]; ok && sfnt.Post.NumGlyphs != 0 && sfnt2.Post.NumGlyphs != 0 {
		if sfnt.Post.NumGlyphs == 0 {
			// initialise the first font to zero
			sfnt.Post.glyphNameIndex = make([]uint16, origNumGlyphs, numGlyphs)
		}
		for glyphID2 := 1; glyphID2 < int(sfnt2.NumGlyphs()); glyphID2++ {
			var index uint16
			name := sfnt2.Post.Get(uint16(glyphID2))
			if glyphID, ok := sfnt.Post.Find(name); ok {
				// reuse existing names
				index = sfnt.Post.glyphNameIndex[glyphID]
			} else if math.MaxUint16 < len(sfnt.Post.stringData)+258 {
				return fmt.Errorf("invalid post table: stringData has too many entries")
			} else {
				index = uint16(len(sfnt.Post.stringData) + 258)
				sfnt.Post.stringData = append(sfnt.Post.stringData, []byte(name))
				sfnt.Post.nameMap[name] = uint16(glyphID2)
			}
			sfnt.Post.glyphNameIndex = append(sfnt.Post.glyphNameIndex, index)
		}

		if sfnt2.Post.IsFixedPitch == 0 {
			sfnt.Post.IsFixedPitch = 0
		}
		sfnt.Post.MinMemType42 += sfnt2.Post.MinMemType42
		sfnt.Post.MaxMemType42 += sfnt2.Post.MaxMemType42
		sfnt.Post.MinMemType1 += sfnt2.Post.MinMemType1
		sfnt.Post.MaxMemType1 += sfnt2.Post.MaxMemType1
		sfnt.Post.NumGlyphs = numGlyphs

		b, err := sfnt.Post.Write()
		if err != nil {
			return fmt.Errorf("invalid post table: %v", err)
		}
		sfnt.Tables["post"] = b
	}
	return nil
}
