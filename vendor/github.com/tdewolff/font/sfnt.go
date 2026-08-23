package font

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
	"sync"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"github.com/tdewolff/parse/v2"
)

// MaxCmapSegments is the maximum number of cmap segments that will be accepted.
const MaxCmapSegments = 20000

// Pather is an interface to append a glyph's path to canvas.Path.
type Pather interface {
	MoveTo(float64, float64)
	LineTo(float64, float64)
	QuadTo(float64, float64, float64, float64)
	CubeTo(float64, float64, float64, float64, float64, float64)
	Close()
}

// Hinting specifies the type of hinting to use (none supported yes).
type Hinting int

// see Hinting
const (
	NoHinting Hinting = iota
	VerticalHinting
)

// SFNT is a parsed OpenType font.
type SFNT struct {
	Length            uint32
	Version           string
	IsCFF, IsTrueType bool // only one can be true
	Tables            map[string][]byte

	// required
	Cmap *cmapTable
	Head *headTable
	Hhea *hheaTable
	Hmtx *hmtxTable
	Maxp *maxpTable
	Name *nameTable
	OS2  *os2Table
	Post *postTable

	// TrueType
	Glyf *glyfTable
	Loca *locaTable

	// CFF
	CFF *cffTable

	// optional
	Kern *kernTable
	Vhea *vheaTable
	Vmtx *vmtxTable

	// TODO: SFNT tables
	//Hdmx *hdmxTable
	Gpos *gposgsubTable
	Gsub *gposgsubTable
	Jsft *jsftTable
	//Gasp *gaspTable
	//Base *baseTable
	//Prep *baseTable
	//Fpgm *baseTable
	//Cvt *baseTable
}

// NumGlyphs returns the number of glyphs the font contains.
func (sfnt *SFNT) NumGlyphs() uint16 {
	if sfnt.Maxp == nil {
		return 0
	}
	return sfnt.Maxp.NumGlyphs
}

// UnitsPerEm returns the number of units per em
func (sfnt *SFNT) UnitsPerEm() uint16 {
	if sfnt.Head == nil {
		return 0
	}
	return sfnt.Head.UnitsPerEm
}

// GlyphIndex returns the glyph ID for the corresponding rune. It looks for each character map subtable in the order in which they appear and returns the first match, or 0 when no match is found.
func (sfnt *SFNT) GlyphIndex(r rune) uint16 {
	if sfnt.Cmap == nil {
		return 0
	}
	return sfnt.Cmap.Get(r)
}

// GlyphToUnicode returns the runes for the corresponding glyph ID. It looks for each character map subtable in the order in which they appear and returns the runes which reference the glyph ID, or nil when no match is found. This is the inverse mapping of the character map table and is cached after the first call. Each glyph may be mapped from multiple runes.
func (sfnt *SFNT) GlyphToUnicode(glyphID uint16) []rune {
	if sfnt.Cmap == nil {
		return nil
	}
	return sfnt.Cmap.ToUnicode(glyphID)
}

// GlyphName returns the name of the glyph. It returns an empty string when no name exists.
func (sfnt *SFNT) GlyphName(glyphID uint16) string {
	if sfnt.IsCFF {
		name, _ := sfnt.CFF.GlyphName(glyphID)
		return name
	}
	return sfnt.Post.Get(glyphID)
}

// FindGlyphName returns the glyphID for a given glyph name. When the name is not defined it returns 0.
func (sfnt *SFNT) FindGlyphName(name string) uint16 {
	if sfnt.IsCFF {
		for glyphID, glyphName := range sfnt.CFF.charset {
			if name == glyphName {
				return uint16(glyphID)
			}
		}
		return 0
	}
	glyphID, _ := sfnt.Post.Find(name)
	return glyphID
}

// VerticalMetrics returns the ascender, descender, and line gap values. It returns the "win" values, or the "typo" values if OS/2.FsSelection.USE_TYPO_METRICS is set. If those are zero or not set, default to the "hhea" values.
func (sfnt *SFNT) VerticalMetrics() (uint16, uint16, uint16) {
	// see https://learn.microsoft.com/en-us/typography/opentype/spec/recom#baseline-to-baseline-distances
	var ascender, descender, lineGap uint16
	if 0 < sfnt.Hhea.Ascender {
		ascender = uint16(sfnt.Hhea.Ascender)
	}
	if sfnt.Hhea.Descender < 0 {
		descender = uint16(-sfnt.Hhea.Descender)
	}
	if 0 < sfnt.Hhea.LineGap {
		lineGap = uint16(sfnt.Hhea.LineGap)
	}

	if (sfnt.OS2.FsSelection & 0x0080) != 0 { // USE_TYPO_METRICS
		if 0 < sfnt.OS2.STypoAscender && sfnt.OS2.STypoDescender < 0 {
			ascender = uint16(sfnt.OS2.STypoAscender)
			descender = uint16(-sfnt.OS2.STypoDescender)
			if 0 < sfnt.OS2.STypoLineGap {
				lineGap = uint16(sfnt.OS2.STypoLineGap)
			} else {
				lineGap = 0
			}
		}
	} else {
		if sfnt.OS2.UsWinAscent != 0 && sfnt.OS2.UsWinDescent != 0 {
			ascender, descender = sfnt.OS2.UsWinAscent, sfnt.OS2.UsWinDescent
			externalLeading := int(sfnt.Hhea.Ascender-sfnt.Hhea.Descender+sfnt.Hhea.LineGap) - int(sfnt.OS2.UsWinAscent+sfnt.OS2.UsWinDescent)
			if 0 < externalLeading {
				lineGap = uint16(externalLeading)
			} else {
				lineGap = 0
			}
		}
	}
	return ascender, descender, lineGap
}

// GlyphPath draws the glyph's contour as a path to the pather interface. It will use the specified ppem (pixels-per-EM) for hinting purposes. The path is draws to the (x,y) coordinate and scaled using the given scale factor.
func (sfnt *SFNT) GlyphPath(p Pather, glyphID, ppem uint16, x, y, scale float64, hinting Hinting) error {
	if sfnt.IsTrueType {
		return sfnt.Glyf.ToPath(p, glyphID, ppem, x, y, scale, hinting)
	} else if sfnt.IsCFF {
		return sfnt.CFF.ToPath(p, glyphID, ppem, x, y, scale, hinting)
	}
	return fmt.Errorf("only TrueType and CFF are supported")
}

// GlyphAdvance returns the (horizontal) advance width of the glyph.
func (sfnt *SFNT) GlyphAdvance(glyphID uint16) uint16 {
	return sfnt.Hmtx.Advance(glyphID)
}

// GlyphVerticalAdvance returns the vertical advance width of the glyph.
func (sfnt *SFNT) GlyphVerticalAdvance(glyphID uint16) uint16 {
	if sfnt.Vmtx == nil {
		return sfnt.Head.UnitsPerEm
	}
	return sfnt.Vmtx.Advance(glyphID)
}

// GlyphBounds returns the bounding rectangle (xmin,ymin,xmax,ymax) of the glyph.
func (sfnt *SFNT) GlyphBounds(glyphID uint16) (int16, int16, int16, int16) {
	if sfnt.IsTrueType {
		contour, err := sfnt.Glyf.Contour(glyphID)
		if err == nil {
			return contour.XMin, contour.YMin, contour.XMax, contour.YMax
		}
	} else if sfnt.IsCFF {
		p := &bboxPather{}
		if err := sfnt.CFF.ToPath(p, glyphID, 0, 0, 0, 1.0, NoHinting); err == nil {
			return int16(p.XMin), int16(p.YMin), int16(math.Ceil(p.XMax)), int16(math.Ceil(p.YMax))
		}
	}
	return 0, 0, 0, 0
}

// Kerning returns the kerning between two glyphs, i.e. the advance correction for glyph pairs.
func (sfnt *SFNT) Kerning(left, right uint16) int16 {
	if sfnt.Kern == nil {
		return 0
	}
	return sfnt.Kern.Get(left, right)
}

// ParseSFNT parses an OpenType file format (TTF, OTF, TTC). The index is used for font collections to select a single font.
func ParseSFNT(b []byte, index int) (*SFNT, error) {
	sfntBytes, err := ToSFNT(b)
	if err != nil {
		return nil, err
	}
	return parseSFNT(sfntBytes, index, false)
}

// ParseEmbeddedSFNT is like ParseSFNT but for embedded font files in PDFs. It allows font files with fewer required tables.
func ParseEmbeddedSFNT(b []byte, index int) (*SFNT, error) {
	return parseSFNT(b, index, true)
}

// ParseCFF parses a bare CFF font file, such as those embedded in PDFs.
// TODO: work in progress
//func ParseCFF(b []byte) (*SFNT, error) {
//	w, _ := os.Create("out.cff")
//	w.Write(b)
//	w.Close()
//
//	sfnt := &SFNT{}
//	sfnt.Version = "OTTO"
//	sfnt.IsCFF = true
//	sfnt.Tables = map[string][]byte{
//		"CFF ": b,
//	}
//	if err := sfnt.parseCFF(); err != nil {
//		return nil, err
//	} else if 256 <= sfnt.CFF.charStrings.Len() {
//		return nil, fmt.Errorf("unsupported CFF with more than 255 glyphs")
//	}
//	sfnt.Maxp = &maxpTable{
//		NumGlyphs: uint16(sfnt.CFF.charStrings.Len()),
//	}
//
//	encoding := windows1252
//
//	cmapFormat := &cmapFormat12{}
//	cmapFormat.StartCharCode = []uint32{uint32(encoding[0])}
//	cmapFormat.StartGlyphID = []uint32{0}
//	for id, r := range encoding {
//		d := uint32(r) - cmapFormat.StartCharCode[len(cmapFormat.StartCharCode)-1]
//		if uint32(id) != cmapFormat.StartGlyphID[len(cmapFormat.StartGlyphID)-1]+d {
//			cmapFormat.StartCharCode = append(cmapFormat.StartCharCode, uint32(r))
//			cmapFormat.EndCharCode = append(cmapFormat.EndCharCode, uint32(encoding[id-1]))
//			cmapFormat.StartGlyphID = append(cmapFormat.StartGlyphID, uint32(id))
//		}
//	}
//	cmapFormat.EndCharCode = append(cmapFormat.EndCharCode, uint32(encoding[len(encoding)-1]))
//	sfnt.Cmap = &cmapTable{
//		Subtables: []cmapSubtable{cmapFormat},
//	}
//
//	fmt.Println(sfnt.CFF.charStrings.Len())
//	return sfnt, nil
//}

func parseSFNT(b []byte, index int, embedded bool) (*SFNT, error) {
	if len(b) < 12 || uint(math.MaxUint32) < uint(len(b)) {
		return nil, ErrInvalidFontData
	}

	r := parse.NewBinaryReaderBytes(b)
	sfntVersion := r.ReadString(4)
	isCollection := sfntVersion == "ttcf"
	if isCollection {
		majorVersion := r.ReadUint16()
		minorVersion := r.ReadUint16()
		if majorVersion != 1 && majorVersion != 2 || minorVersion != 0 {
			return nil, fmt.Errorf("bad TTC version")
		}

		numFonts := r.ReadUint32()
		if index < 0 || numFonts <= uint32(index) {
			return nil, fmt.Errorf("bad font index %d", index)
		}
		if r.Len() < 4*int64(numFonts) {
			return nil, ErrInvalidFontData
		}

		_ = r.ReadBytes(int64(4 * index))
		offset := r.ReadUint32()
		var length uint32
		if uint32(index)+1 == numFonts {
			length = uint32(len(b)) - offset
		} else {
			length = r.ReadUint32() - offset
		}
		if uint32(len(b))-8 < offset || uint32(len(b))-8-offset < length {
			return nil, ErrInvalidFontData
		}

		r.Seek(int64(offset), 0)
		sfntVersion = r.ReadString(4)
	} else if index != 0 {
		return nil, fmt.Errorf("bad font index %d", index)
	}
	if sfntVersion != "OTTO" && sfntVersion != "true" && binary.BigEndian.Uint32([]byte(sfntVersion)) != 0x00010000 {
		return nil, fmt.Errorf("bad SFNT version")
	}
	numTables := r.ReadUint16()
	_ = r.ReadUint16()                 // searchRange
	_ = r.ReadUint16()                 // entrySelector
	_ = r.ReadUint16()                 // rangeShift
	if r.Len() < 16*int64(numTables) { // can never exceed uint32 as numTables is uint16
		return nil, ErrInvalidFontData
	}

	tables := make(map[string][]byte, numTables)
	for i := 0; i < int(numTables); i++ {
		tag := r.ReadString(4)
		_ = r.ReadUint32() // checksum
		offset := r.ReadUint32()
		length := r.ReadUint32()

		padding := (4 - length&3) & 3
		if uint32(len(b)) <= offset || uint32(len(b))-offset < length || uint32(len(b))-offset-length < padding {
			return nil, ErrInvalidFontData
		}

		if tag == "head" {
			if length < 12 {
				return nil, ErrInvalidFontData
			}

			// NOTE: checksum validation is disabled so as to parse broken fonts as best as possible
			// to check checksum for head table, replace the overal checksum with zero and reset it at the end
			//checksumAdjustment := binary.BigEndian.Uint32(b[offset+8:])
			//binary.BigEndian.PutUint32(b[offset+8:], 0x00000000)
			//if calcChecksum(b[offset:offset+length+padding]) != checksum {
			//	return nil, fmt.Errorf("%s: bad checksum", tag)
			//} else if 0xB1B0AFBA-calcChecksum(b) != checksumAdjustment {
			//	return nil, fmt.Errorf("bad checksum")
			//}
			//binary.BigEndian.PutUint32(b[offset+8:], checksumAdjustment)
			//} else if calcChecksum(b[offset:offset+length+padding]) != checksum {
			//	return nil, fmt.Errorf("%s: bad checksum", tag)
		}
		tables[tag] = b[offset : offset+length : offset+length]
	}
	if err := r.Err(); err == io.EOF {
		return nil, ErrInvalidFontData
	} else if err != nil {
		return nil, err
	}

	sfnt := &SFNT{}
	sfnt.Length = uint32(len(b))
	sfnt.Version = sfntVersion
	sfnt.IsCFF = sfntVersion == "OTTO"
	sfnt.IsTrueType = sfntVersion == "true" || binary.BigEndian.Uint32([]byte(sfntVersion)) == 0x00010000
	sfnt.Tables = tables
	//if isCollection {
	//	sfnt.Data = sfnt.Write() // TODO: what is this?
	//}

	var requiredTables []string
	if embedded {
		// see Table 126 of the PDF32000 specification
		if sfnt.IsTrueType {
			requiredTables = []string{"glyf", "head", "hhea", "hmtx", "loca", "maxp"}
		} else if sfnt.IsCFF {
			requiredTables = []string{"cmap", "CFF "}
		}
	} else {
		requiredTables = []string{"cmap", "head", "hhea", "hmtx", "maxp", "name", "post"} // OS/2 not required by TrueType
		if sfnt.IsTrueType {
			requiredTables = append(requiredTables, "glyf", "loca")
		} else if sfnt.IsCFF {
			_, hasCFF := tables["CFF "]
			_, hasCFF2 := tables["CFF2"]
			if !hasCFF && !hasCFF2 {
				return nil, fmt.Errorf("CFF: missing table")
			} else if hasCFF && hasCFF2 {
				return nil, fmt.Errorf("CFF2: CFF table already exists")
			}
		}
	}
	for _, requiredTable := range requiredTables {
		if _, ok := tables[requiredTable]; !ok {
			return nil, fmt.Errorf("%s: missing table", requiredTable)
		}
	}

	if embedded && sfnt.IsCFF {
		if err := sfnt.parseCFF(); err != nil {
			return nil, err
		} else if err := sfnt.parseCmap(); err != nil {
			return nil, err
		}
		return sfnt, nil
	}

	// required tables before parsing other tables
	if err := sfnt.parseHead(); err != nil {
		return nil, err
	} else if err := sfnt.parseMaxp(); err != nil {
		return nil, err
	}
	if sfnt.IsTrueType {
		if err := sfnt.parseLoca(); err != nil {
			return nil, err
		}
	}

	tags := make([]string, 0, len(tables))
	for tag := range tables {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	for _, tag := range tags {
		var err error
		switch tag {
		case "CFF ":
			err = sfnt.parseCFF()
		case "CFF2":
			err = sfnt.parseCFF2()
		case "cmap":
			err = sfnt.parseCmap()
		case "glyf":
			err = sfnt.parseGlyf()
		//case "GPOS":
		//	err = sfnt.parseGPOS()
		//case "GSUB":
		//	err = sfnt.parseGSUB()
		case "hhea":
			err = sfnt.parseHhea()
		case "hmtx":
			err = sfnt.parseHmtx()
		case "kern":
			err = sfnt.parseKern()
		case "name":
			err = sfnt.parseName()
		case "OS/2":
			err = sfnt.parseOS2()
		case "post":
			err = sfnt.parsePost()
		case "vhea":
			err = sfnt.parseVhea()
		case "vmtx":
			err = sfnt.parseVmtx()
		}
		if err != nil {
			return nil, err
		}
	}
	if sfnt.OS2 != nil && sfnt.OS2.Version <= 1 {
		sfnt.estimateOS2()
	}
	return sfnt, nil
}

// Write writes out the SFNT file.
func (sfnt *SFNT) Write() []byte {
	tags := make([]string, 0, len(sfnt.Tables))
	for tag := range sfnt.Tables {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	// write header
	w := parse.NewBinaryWriter([]byte{})
	if sfnt.IsTrueType {
		w.WriteUint32(0x00010000) // sfntVersion
	} else if sfnt.IsCFF {
		w.WriteString("OTTO") // sfntVersion
	}
	numTables := uint16(len(tags))
	entrySelector := uint16(math.Log2(float64(numTables)))
	searchRange := uint16(1 << (entrySelector + 4))
	w.WriteUint16(numTables)                  // numTables
	w.WriteUint16(searchRange)                // searchRange
	w.WriteUint16(entrySelector)              // entrySelector
	w.WriteUint16(numTables<<4 - searchRange) // rangeShift

	// we'll write the table records at the end
	w.WriteBytes(make([]byte, numTables<<4))

	// write tables
	var checksumAdjustmentPos uint32
	offsets, lengths := make([]uint32, numTables), make([]uint32, numTables)
	for i, tag := range tags {
		offsets[i] = uint32(w.Len())
		table := sfnt.Tables[tag]
		if tag == "head" {
			checksumAdjustmentPos = uint32(w.Len()) + 8
			w.WriteBytes(table[:8])
			w.WriteUint32(0)
			w.WriteBytes(table[12:28])
			w.WriteInt64(int64(time.Now().UTC().Sub(time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)) / 1e9)) // modified
			w.WriteBytes(table[36:])
		} else {
			w.WriteBytes(table)
		}
		lengths[i] = uint32(w.Len()) - offsets[i]

		padding := (4 - lengths[i]&3) & 3
		for i := 0; i < int(padding); i++ {
			w.WriteByte(0)
		}
	}

	// add table record entries
	buf := w.Bytes()
	for i, tag := range tags {
		pos := 12 + i<<4
		copy(buf[pos:], []byte(tag))
		padding := (4 - lengths[i]&3) & 3
		checksum := calcChecksum(buf[offsets[i] : offsets[i]+lengths[i]+padding])
		binary.BigEndian.PutUint32(buf[pos+4:], checksum)
		binary.BigEndian.PutUint32(buf[pos+8:], offsets[i])
		binary.BigEndian.PutUint32(buf[pos+12:], lengths[i])
	}
	binary.BigEndian.PutUint32(buf[checksumAdjustmentPos:], 0xB1B0AFBA-calcChecksum(buf))
	return buf
}

////////////////////////////////////////////////////////////////

type headTable struct {
	FontRevision           uint32
	Flags                  [16]bool
	UnitsPerEm             uint16
	Created, Modified      time.Time
	XMin, YMin, XMax, YMax int16
	MacStyle               [16]bool
	LowestRecPPEM          uint16
	FontDirectionHint      int16
	IndexToLocFormat       int16
	GlyphDataFormat        int16
}

func (sfnt *SFNT) parseHead() error {
	b, ok := sfnt.Tables["head"]
	if !ok {
		return fmt.Errorf("head: missing table")
	} else if len(b) != 54 {
		return fmt.Errorf("head: bad table")
	}

	sfnt.Head = &headTable{}
	r := parse.NewBinaryReaderBytes(b)
	majorVersion := r.ReadUint16()
	minorVersion := r.ReadUint16()
	if majorVersion != 1 && minorVersion != 0 {
		return fmt.Errorf("head: bad version")
	}
	sfnt.Head.FontRevision = r.ReadUint32()
	_ = r.ReadUint32()                // checksumAdjustment
	if r.ReadUint32() != 0x5F0F3CF5 { // magicNumber
		return fmt.Errorf("head: bad magic version")
	}
	sfnt.Head.Flags = Uint16ToFlags(r.ReadUint16())
	sfnt.Head.UnitsPerEm = r.ReadUint16()
	created := r.ReadUint64()
	modified := r.ReadUint64()
	if math.MaxInt64 < created || math.MaxInt64 < modified {
		return fmt.Errorf("head: created and/or modified dates too large")
	}
	sfnt.Head.Created = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Second * time.Duration(created))
	sfnt.Head.Modified = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Second * time.Duration(modified))
	sfnt.Head.XMin = r.ReadInt16()
	sfnt.Head.YMin = r.ReadInt16()
	sfnt.Head.XMax = r.ReadInt16()
	sfnt.Head.YMax = r.ReadInt16()
	sfnt.Head.MacStyle = Uint16ToFlags(r.ReadUint16())
	sfnt.Head.LowestRecPPEM = r.ReadUint16()
	sfnt.Head.FontDirectionHint = r.ReadInt16()
	sfnt.Head.IndexToLocFormat = r.ReadInt16()
	if sfnt.Head.IndexToLocFormat != 0 && sfnt.Head.IndexToLocFormat != 1 {
		return fmt.Errorf("head: bad indexToLocFormat")
	}
	sfnt.Head.GlyphDataFormat = r.ReadInt16()
	return nil
}

////////////////////////////////////////////////////////////////

type hheaTable struct {
	Ascender            int16
	Descender           int16
	LineGap             int16
	AdvanceWidthMax     uint16
	MinLeftSideBearing  int16
	MinRightSideBearing int16
	XMaxExtent          int16
	CaretSlopeRise      int16
	CaretSlopeRun       int16
	CaretOffset         int16
	MetricDataFormat    int16
	NumberOfHMetrics    uint16
}

func (sfnt *SFNT) parseHhea() error {
	if sfnt.Maxp == nil {
		return fmt.Errorf("hhea: missing maxp table")
	}

	b, ok := sfnt.Tables["hhea"]
	if !ok {
		return fmt.Errorf("hhea: missing table")
	} else if len(b) != 36 {
		return fmt.Errorf("hhea: bad table")
	}

	sfnt.Hhea = &hheaTable{}
	r := parse.NewBinaryReaderBytes(b)
	majorVersion := r.ReadUint16()
	minorVersion := r.ReadUint16()
	if majorVersion != 1 && minorVersion != 0 {
		return fmt.Errorf("hhea: bad version")
	}
	sfnt.Hhea.Ascender = r.ReadInt16()
	sfnt.Hhea.Descender = r.ReadInt16()
	sfnt.Hhea.LineGap = r.ReadInt16()
	sfnt.Hhea.AdvanceWidthMax = r.ReadUint16()
	sfnt.Hhea.MinLeftSideBearing = r.ReadInt16()
	sfnt.Hhea.MinRightSideBearing = r.ReadInt16()
	sfnt.Hhea.XMaxExtent = r.ReadInt16()
	sfnt.Hhea.CaretSlopeRise = r.ReadInt16()
	sfnt.Hhea.CaretSlopeRun = r.ReadInt16()
	sfnt.Hhea.CaretOffset = r.ReadInt16()
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	sfnt.Hhea.MetricDataFormat = r.ReadInt16()
	sfnt.Hhea.NumberOfHMetrics = r.ReadUint16()
	if sfnt.Maxp.NumGlyphs < sfnt.Hhea.NumberOfHMetrics || sfnt.Hhea.NumberOfHMetrics == 0 {
		return fmt.Errorf("hhea: bad numberOfHMetrics")
	}
	return nil
}

////////////////////////////////////////////////////////////////

type vheaTable struct {
	Ascender             int16
	Descender            int16
	LineGap              int16
	AdvanceHeightMax     int16
	MinTopSideBearing    int16
	MinBottomSideBearing int16
	YMaxExtent           int16
	CaretSlopeRise       int16
	CaretSlopeRun        int16
	CaretOffset          int16
	MetricDataFormat     int16
	NumberOfVMetrics     uint16
}

func (sfnt *SFNT) parseVhea() error {
	if sfnt.Maxp == nil {
		return fmt.Errorf("vhea: missing maxp table")
	}

	b, ok := sfnt.Tables["vhea"]
	if !ok {
		return fmt.Errorf("vhea: missing table")
	} else if len(b) != 36 {
		return fmt.Errorf("vhea: bad table")
	}

	sfnt.Vhea = &vheaTable{}
	r := parse.NewBinaryReaderBytes(b)
	majorVersion := r.ReadUint16()
	minorVersion := r.ReadUint16()
	if majorVersion != 1 && minorVersion != 0 && minorVersion != 1 {
		return fmt.Errorf("vhea: bad version")
	}
	sfnt.Vhea.Ascender = r.ReadInt16()
	sfnt.Vhea.Descender = r.ReadInt16()
	sfnt.Vhea.LineGap = r.ReadInt16()
	sfnt.Vhea.AdvanceHeightMax = r.ReadInt16()
	sfnt.Vhea.MinTopSideBearing = r.ReadInt16()
	sfnt.Vhea.MinBottomSideBearing = r.ReadInt16()
	sfnt.Vhea.YMaxExtent = r.ReadInt16()
	sfnt.Vhea.CaretSlopeRise = r.ReadInt16()
	sfnt.Vhea.CaretSlopeRun = r.ReadInt16()
	sfnt.Vhea.CaretOffset = r.ReadInt16()
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	sfnt.Vhea.MetricDataFormat = r.ReadInt16()
	sfnt.Vhea.NumberOfVMetrics = r.ReadUint16()
	if sfnt.Maxp.NumGlyphs < sfnt.Vhea.NumberOfVMetrics || sfnt.Vhea.NumberOfVMetrics == 0 {
		return fmt.Errorf("vhea: bad numberOfVMetrics")
	}
	return nil
}

////////////////////////////////////////////////////////////////

type hmtxLongHorMetric struct {
	AdvanceWidth    uint16
	LeftSideBearing int16
}

type hmtxTable struct {
	HMetrics         []hmtxLongHorMetric
	LeftSideBearings []int16
}

func (hmtx *hmtxTable) LeftSideBearing(glyphID uint16) int16 {
	if uint16(len(hmtx.HMetrics)) <= glyphID {
		return hmtx.LeftSideBearings[glyphID-uint16(len(hmtx.HMetrics))]
	}
	return hmtx.HMetrics[glyphID].LeftSideBearing
}

func (hmtx *hmtxTable) Advance(glyphID uint16) uint16 {
	if uint16(len(hmtx.HMetrics)) <= glyphID {
		glyphID = uint16(len(hmtx.HMetrics)) - 1
	}
	return hmtx.HMetrics[glyphID].AdvanceWidth
}

func (sfnt *SFNT) parseHmtx() error {
	// TODO: lazy parse
	if sfnt.Hhea == nil {
		return fmt.Errorf("hmtx: missing hhea table")
	} else if sfnt.Maxp == nil {
		return fmt.Errorf("hmtx: missing maxp table")
	}

	b, ok := sfnt.Tables["hmtx"]
	length := 4*uint32(sfnt.Hhea.NumberOfHMetrics) + 2*uint32(sfnt.Maxp.NumGlyphs-sfnt.Hhea.NumberOfHMetrics)
	if !ok {
		return fmt.Errorf("hmtx: missing table")
	} else if uint32(len(b)) != length {
		return fmt.Errorf("hmtx: bad table")
	}

	sfnt.Hmtx = &hmtxTable{}
	// numberOfHMetrics is smaller than numGlyphs
	sfnt.Hmtx.HMetrics = make([]hmtxLongHorMetric, sfnt.Hhea.NumberOfHMetrics)
	sfnt.Hmtx.LeftSideBearings = make([]int16, sfnt.Maxp.NumGlyphs-sfnt.Hhea.NumberOfHMetrics)

	r := parse.NewBinaryReaderBytes(b)
	for i := 0; i < int(sfnt.Hhea.NumberOfHMetrics); i++ {
		sfnt.Hmtx.HMetrics[i].AdvanceWidth = r.ReadUint16()
		sfnt.Hmtx.HMetrics[i].LeftSideBearing = r.ReadInt16()
	}
	for i := 0; i < int(sfnt.Maxp.NumGlyphs-sfnt.Hhea.NumberOfHMetrics); i++ {
		sfnt.Hmtx.LeftSideBearings[i] = r.ReadInt16()
	}
	return nil
}

////////////////////////////////////////////////////////////////

type vmtxLongVerMetric struct {
	AdvanceHeight  uint16
	TopSideBearing int16
}

type vmtxTable struct {
	VMetrics        []vmtxLongVerMetric
	TopSideBearings []int16
}

func (vmtx *vmtxTable) TopSideBearing(glyphID uint16) int16 {
	if uint16(len(vmtx.VMetrics)) <= glyphID {
		return vmtx.TopSideBearings[glyphID-uint16(len(vmtx.VMetrics))]
	}
	return vmtx.VMetrics[glyphID].TopSideBearing
}

func (vmtx *vmtxTable) Advance(glyphID uint16) uint16 {
	if uint16(len(vmtx.VMetrics)) <= glyphID {
		glyphID = uint16(len(vmtx.VMetrics)) - 1
	}
	return vmtx.VMetrics[glyphID].AdvanceHeight
}

func (sfnt *SFNT) parseVmtx() error {
	// TODO: lazy parse
	if sfnt.Vhea == nil {
		return fmt.Errorf("vmtx: missing vhea table")
	} else if sfnt.Maxp == nil {
		return fmt.Errorf("vmtx: missing maxp table")
	}

	b, ok := sfnt.Tables["vmtx"]
	length := 4*uint32(sfnt.Vhea.NumberOfVMetrics) + 2*uint32(sfnt.Maxp.NumGlyphs-sfnt.Vhea.NumberOfVMetrics)
	if !ok {
		return fmt.Errorf("vmtx: missing table")
	} else if uint32(len(b)) != length {
		return fmt.Errorf("vmtx: bad table")
	}

	sfnt.Vmtx = &vmtxTable{}
	// numberOfVMetrics is smaller than numGlyphs
	sfnt.Vmtx.VMetrics = make([]vmtxLongVerMetric, sfnt.Vhea.NumberOfVMetrics)
	sfnt.Vmtx.TopSideBearings = make([]int16, sfnt.Maxp.NumGlyphs-sfnt.Vhea.NumberOfVMetrics)

	r := parse.NewBinaryReaderBytes(b)
	for i := 0; i < int(sfnt.Vhea.NumberOfVMetrics); i++ {
		sfnt.Vmtx.VMetrics[i].AdvanceHeight = r.ReadUint16()
		sfnt.Vmtx.VMetrics[i].TopSideBearing = r.ReadInt16()
	}
	for i := 0; i < int(sfnt.Maxp.NumGlyphs-sfnt.Vhea.NumberOfVMetrics); i++ {
		sfnt.Vmtx.TopSideBearings[i] = r.ReadInt16()
	}
	return nil
}

////////////////////////////////////////////////////////////////

type maxpTable struct {
	Version               uint32
	NumGlyphs             uint16
	MaxPoints             uint16
	MaxContours           uint16
	MaxCompositePoints    uint16
	MaxCompositeContours  uint16
	MaxZones              uint16
	MaxTwilightPoints     uint16
	MaxStorage            uint16
	MaxFunctionDefs       uint16
	MaxInstructionDefs    uint16
	MaxStackElements      uint16
	MaxSizeOfInstructions uint16
	MaxComponentElements  uint16
	MaxComponentDepth     uint16
}

func (sfnt *SFNT) parseMaxp() error {
	b, ok := sfnt.Tables["maxp"]
	if !ok {
		return fmt.Errorf("maxp: missing table")
	}

	sfnt.Maxp = &maxpTable{}
	r := parse.NewBinaryReaderBytes(b)
	sfnt.Maxp.Version = binary.BigEndian.Uint32(r.ReadBytes(4))
	sfnt.Maxp.NumGlyphs = r.ReadUint16()
	if sfnt.Maxp.Version == 0x00005000 && !sfnt.IsTrueType && len(b) == 6 {
		return nil
	} else if sfnt.Maxp.Version == 0x00010000 && !sfnt.IsCFF && len(b) == 32 {
		sfnt.Maxp.MaxPoints = r.ReadUint16()
		sfnt.Maxp.MaxContours = r.ReadUint16()
		sfnt.Maxp.MaxCompositePoints = r.ReadUint16()
		sfnt.Maxp.MaxCompositeContours = r.ReadUint16()
		sfnt.Maxp.MaxZones = r.ReadUint16()
		sfnt.Maxp.MaxTwilightPoints = r.ReadUint16()
		sfnt.Maxp.MaxStorage = r.ReadUint16()
		sfnt.Maxp.MaxFunctionDefs = r.ReadUint16()
		sfnt.Maxp.MaxInstructionDefs = r.ReadUint16()
		sfnt.Maxp.MaxStackElements = r.ReadUint16()
		sfnt.Maxp.MaxSizeOfInstructions = r.ReadUint16()
		sfnt.Maxp.MaxComponentElements = r.ReadUint16()
		sfnt.Maxp.MaxComponentDepth = r.ReadUint16()
		return nil
	}
	return fmt.Errorf("maxp: bad table")
}

func (maxp *maxpTable) Write() []byte {
	w := parse.NewBinaryWriter(make([]byte, 0, 32))
	w.WriteUint32(maxp.Version)
	w.WriteUint16(maxp.NumGlyphs)
	if maxp.Version == 0x00010000 {
		w.WriteUint16(maxp.MaxPoints)
		w.WriteUint16(maxp.MaxContours)
		w.WriteUint16(maxp.MaxCompositePoints)
		w.WriteUint16(maxp.MaxCompositeContours)
		w.WriteUint16(maxp.MaxZones)
		w.WriteUint16(maxp.MaxTwilightPoints)
		w.WriteUint16(maxp.MaxStorage)
		w.WriteUint16(maxp.MaxFunctionDefs)
		w.WriteUint16(maxp.MaxInstructionDefs)
		w.WriteUint16(maxp.MaxStackElements)
		w.WriteUint16(maxp.MaxSizeOfInstructions)
		w.WriteUint16(maxp.MaxComponentElements)
		w.WriteUint16(maxp.MaxComponentDepth)
	}
	return w.Bytes()
}

////////////////////////////////////////////////////////////////

type nameRecord struct {
	Platform PlatformID
	Encoding EncodingID
	Language uint16
	Name     NameID
	Value    []byte
}

func (record nameRecord) String() string {
	var decoder *encoding.Decoder
	if record.Platform == PlatformUnicode || record.Platform == PlatformWindows {
		decoder = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
	} else if record.Platform == PlatformMacintosh && record.Encoding == EncodingMacintoshRoman {
		decoder = charmap.Macintosh.NewDecoder()
	}
	s, _, err := transform.String(decoder, string(record.Value))
	if err == nil {
		return s
	}
	return string(record.Value)
}

type nameLangTagRecord struct {
	Value []byte
}

func (record nameLangTagRecord) String() string {
	decoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
	s, _, err := transform.String(decoder, string(record.Value))
	if err == nil {
		return s
	}
	return string(record.Value)
}

type nameTable struct {
	NameRecord []nameRecord
	LangTag    []nameLangTagRecord
}

func (t *nameTable) Get(name NameID) []nameRecord {
	records := []nameRecord{}
	for _, record := range t.NameRecord {
		if record.Name == name {
			records = append(records, record)
		}
	}
	return records
}

func (sfnt *SFNT) parseName() error {
	// TODO: lazy parse
	b, ok := sfnt.Tables["name"]
	if !ok {
		return fmt.Errorf("name: missing table")
	} else if len(b) < 6 {
		return fmt.Errorf("name: bad table")
	}

	sfnt.Name = &nameTable{}
	r := parse.NewBinaryReaderBytes(b)
	version := r.ReadUint16()
	if version != 0 && version != 1 {
		return fmt.Errorf("name: bad version")
	}
	count := r.ReadUint16()
	storageOffset := r.ReadUint16()
	if uint32(len(b)) < 6+12*uint32(count) || uint16(len(b)) < storageOffset {
		return fmt.Errorf("name: bad table")
	}
	sfnt.Name.NameRecord = make([]nameRecord, count)
	for i := 0; i < int(count); i++ {
		sfnt.Name.NameRecord[i].Platform = PlatformID(r.ReadUint16())
		sfnt.Name.NameRecord[i].Encoding = EncodingID(r.ReadUint16())
		sfnt.Name.NameRecord[i].Language = r.ReadUint16()
		sfnt.Name.NameRecord[i].Name = NameID(r.ReadUint16())

		length := r.ReadUint16()
		offset := r.ReadUint16()
		if uint16(len(b))-storageOffset < offset || uint16(len(b))-storageOffset-offset < length {
			return fmt.Errorf("name: bad table")
		}
		sfnt.Name.NameRecord[i].Value = b[storageOffset+offset : storageOffset+offset+length]
	}
	if version == 1 {
		if uint32(len(b)) < 6+12*uint32(count)+2 {
			return fmt.Errorf("name: bad table")
		}
		langTagCount := r.ReadUint16()
		if uint32(len(b)) < 6+12*uint32(count)+2+4*uint32(langTagCount) {
			return fmt.Errorf("name: bad table")
		}
		sfnt.Name.LangTag = make([]nameLangTagRecord, langTagCount)
		for i := 0; i < int(langTagCount); i++ {
			length := r.ReadUint16()
			offset := r.ReadUint16()
			if uint16(len(b))-storageOffset < offset || uint16(len(b))-storageOffset-offset < length {
				return fmt.Errorf("name: bad table")
			}
			sfnt.Name.LangTag[i].Value = b[storageOffset+offset : storageOffset+offset+length]
		}
	}
	if r.Pos() != int64(storageOffset) {
		return fmt.Errorf("name: bad storageOffset")
	}
	return nil
}

////////////////////////////////////////////////////////////////

type postTable struct {
	ItalicAngle        float64
	UnderlinePosition  int16
	UnderlineThickness int16
	IsFixedPitch       uint32
	MinMemType42       uint32
	MaxMemType42       uint32
	MinMemType1        uint32
	MaxMemType1        uint32

	// version 2
	NumGlyphs      uint16
	glyphNameIndex []uint16
	stringData     [][]byte
	nameMap        map[string]uint16
	once           sync.Once
}

func (post *postTable) Get(glyphID uint16) string {
	if len(post.glyphNameIndex) <= int(glyphID) {
		return ""
	}
	index := post.glyphNameIndex[glyphID]
	if index < 258 {
		return macintoshGlyphNames[index]
	} else if len(post.stringData) <= int(index)-258 {
		return ""
	}
	return string(post.stringData[index-258])
}

func (post *postTable) Find(name string) (uint16, bool) {
	post.once.Do(func() {
		post.nameMap = make(map[string]uint16, len(post.glyphNameIndex))
		for glyphID, index := range post.glyphNameIndex {
			if index < 258 {
				post.nameMap[macintoshGlyphNames[index]] = uint16(glyphID)
			} else if int(index)-258 < len(post.stringData) {
				post.nameMap[string(post.stringData[index-258])] = uint16(glyphID)
			}
		}
	})
	glyphID, ok := post.nameMap[name] // returns 0 if not found
	return glyphID, ok
}

func (sfnt *SFNT) parsePost() error {
	if sfnt.Maxp == nil {
		return fmt.Errorf("post: missing maxp table")
	}

	b, ok := sfnt.Tables["post"]
	if !ok {
		return fmt.Errorf("post: missing table")
	} else if len(b) < 32 {
		return fmt.Errorf("post: bad table")
	}

	_, isCFF2 := sfnt.Tables["CFF2"]

	sfnt.Post = &postTable{}
	r := parse.NewBinaryReaderBytes(b)
	version := r.ReadUint32()
	sfnt.Post.ItalicAngle = float64(r.ReadInt32()) / (1 << 16)
	sfnt.Post.UnderlinePosition = r.ReadInt16()
	sfnt.Post.UnderlineThickness = r.ReadInt16()
	sfnt.Post.IsFixedPitch = r.ReadUint32()
	sfnt.Post.MinMemType42 = r.ReadUint32()
	sfnt.Post.MaxMemType42 = r.ReadUint32()
	sfnt.Post.MinMemType1 = r.ReadUint32()
	sfnt.Post.MaxMemType1 = r.ReadUint32()
	if version == 0x00010000 && sfnt.IsTrueType && len(b) == 32 {
		sfnt.Post.glyphNameIndex = make([]uint16, 258)
		for i := 0; i < 258; i++ {
			sfnt.Post.glyphNameIndex[i] = uint16(i)
		}
		return nil
	} else if version == 0x00020000 && (sfnt.IsTrueType || isCFF2) && 34 <= len(b) {
		// can be used for TrueType and CFF2 fonts, we check for this in the CFF table
		sfnt.Post.NumGlyphs = r.ReadUint16()
		if sfnt.Post.NumGlyphs != sfnt.Maxp.NumGlyphs {
			return fmt.Errorf("post: numGlyphs does not match maxp table numGlyphs")
		} else if uint32(len(b)) < 34+2*uint32(sfnt.Post.NumGlyphs) {
			return fmt.Errorf("post: bad table")
		}

		sfnt.Post.glyphNameIndex = make([]uint16, sfnt.Post.NumGlyphs)
		for i := 0; i < int(sfnt.Post.NumGlyphs); i++ {
			sfnt.Post.glyphNameIndex[i] = r.ReadUint16()
		}

		// get string data first
		sfnt.Post.stringData = [][]byte{}
		for 2 <= r.Len() {
			length := r.ReadUint8()
			if r.Len() < int64(length) || 63 < length {
				return fmt.Errorf("post: bad stringData")
			}
			sfnt.Post.stringData = append(sfnt.Post.stringData, r.ReadBytes(int64(length)))
		}
		if 1 < r.Len() {
			return fmt.Errorf("post: bad stringData")
		}
		return nil
	} else if version == 0x00025000 && sfnt.IsTrueType && len(b) == 32 {
		return fmt.Errorf("post: version 2.5 not supported")
	} else if version == 0x00030000 && len(b) == 32 {
		// no PostScript glyph names provided
		return nil
	}
	return fmt.Errorf("post: bad version")
}

func (post *postTable) Write() ([]byte, error) {
	version := 0x00030000
	if 0 < post.NumGlyphs {
		version = 0x00020000
	}

	w := parse.NewBinaryWriter(make([]byte, 0, 32))
	w.WriteUint32(uint32(version))
	w.WriteUint32(uint32(post.ItalicAngle*(1<<16) + 0.5))
	w.WriteInt16(post.UnderlinePosition)
	w.WriteInt16(post.UnderlineThickness)
	w.WriteUint32(post.IsFixedPitch)
	w.WriteUint32(post.MinMemType42)
	w.WriteUint32(post.MaxMemType42)
	w.WriteUint32(post.MinMemType1)
	w.WriteUint32(post.MaxMemType1)
	if version == 0x00030000 {
		return w.Bytes(), nil
	} else if len(post.glyphNameIndex) != int(post.NumGlyphs) {
		return nil, fmt.Errorf("insufficient glyph name indices")
	} else if math.MaxUint16 < len(post.stringData) {
		return nil, fmt.Errorf("stringData has too many entries")
	}

	w.WriteUint16(post.NumGlyphs)
	for _, index := range post.glyphNameIndex {
		if 258 <= index && len(post.stringData) <= int(index-258) {
			return nil, fmt.Errorf("glyphNameIndex out-of-range for stringData")
		}
		w.WriteUint16(index)
	}
	for _, str := range post.stringData {
		if 63 < len(str) {
			return nil, fmt.Errorf("glyph name too long")
		}
		w.WriteUint8(uint8(len(str)))
		w.WriteBytes(str)
	}
	return w.Bytes(), nil
}
