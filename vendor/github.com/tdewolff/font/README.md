# Font [![API reference](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/tdewolff/font?tab=doc)

Parsers for SFNT-based fonts (TTF, OTF, WOFF, WOFF2, EOT) that can extract glyph paths, advancement and kerning data, Unicode mapping and glyph lookups. It also supports **merging two fonts** as well as **subsetting fonts** using a list of glyphs. It can write out TTF/OTF as well as WOFF2 fonts. The WOFF and WOFF2 converters have been testing using the validation tests from the W3C (https://github.com/w3c/woff/tree/master/woff1/tests and https://github.com/w3c/woff2-tests).

**[fontcmd](https://github.com/tdewolff/font/tree/master/cmd/fontcmd)**: font toolkit that can select a subset of glyphs from a font, merge fonts, or show font information and display glyphs in the command line or as a raster image.

## Usage
### Parse font

``` go
import "github.com/tdewolff/font"

// Can be any of TTF, OTF, TTC, WOFF, WOFF2, or EOT
fontBytes, err := os.ReadFile("DejaVuSerif.otf")
if err != nil {
    panic(err)
}

sfnt, err := font.ParseSFNT(fontBytes, 0) // 0 is the first index for TTC fonts
if err != nil {
    panic(err)
}

// font information
sfnt.NumGlyphs() uint16
sfnt.UnitsPerEm() uint16
sfnt.VerticalMetrics() (uint16, uint16, uint16)

// glyph mappings
sfnt.GlyphIndex(r rune) uint16
sfnt.GlyphToUnicode(glyphID uint16) []rune
sfnt.GlyphName(glyphID uint16) string
sfnt.FindGlyphName(name string) uint16

// glyph shapes
sfnt.GlyphPath(p Pather, glyphID, ppem uint16, x, y, scale float64, hinting Hinting) error
sfnt.GlyphBounds(glyphID uint16) (int16, int16, int16, int16)
sfnt.GlyphAdvance(glyphID uint16) uint16
sfnt.GlyphVerticalAdvance(glyphID uint16) uint16
sfnt.Kerning(left, right uint16) int16

// editting
sfnt.SetGlyphNames(names []string) error
sfnt.Merge(sfnt *SFNT, options MergeOptions) error
sfnt.Subset(glyphIDs []uint16, options SubsetOptions) (*SFNT, error)
sfnt.Write() []byte
sfnt.WriteWOFF2() ([]byte, error)
```

### [Extract glyph shape](https://github.com/tdewolff/font/tree/master/examples/glyphs)

``` go
import (
    "github.com/tdewolff/canvas"
    "github.com/tdewolff/font"
)

const mmPerPt = 25.4 / 72.0

func main () {
    // Can be any of TTF, OTF, TTC, WOFF, WOFF2, or EOT
    fontBytes, err := os.ReadFile("DejaVuSerif.otf")
    if err != nil {
        panic(err)
    }

    sfnt, err := font.ParseSFNT(fontBytes, 0) // 0 is the first index for TTC fonts
    if err != nil {
        panic(err)
    }

    dpmm := 2.0 // pixels per mm, rasterisation resolution
    fontSize := 12.0 // font size in points

    scale := fontSize * mmPerPt / float64(sfnt.UnitsPerEm()) // mm per units-per-em
    ppem := uint16(dpmm * fontSize * mmPerPt + 0.5) // size of em in pixels
    ascender, descender, _ := sfnt.VerticalMetrics()

    var x, y int16
    p := &canvas.Path{}

    indexA := sfnt.GlyphIndex('A') // can be 0 if not present
    indexV := sfnt.GlyphIndex('V') // can be 0 if not present
    if err := sfnt.GlyphPath(p, indexA, ppem, scale*float64(x), scale*float64(y), scale, font.NoHinting); err != nil {
        panic(err)
    }
    x += int16(sfnt.GlyphAdvance(indexA))
    x += sfnt.Kerning(indexA, indexV)

    if err := sfnt.GlyphPath(p, indexV, ppem, scale*float64(x), scale*float64(y), scale, font.NoHinting); err != nil {
        panic(err)
    }
    x += int16(sfnt.GlyphAdvance(indexV))

    width := scale*float64(scale)
    height := scale*float64(ascender+descender)

    c := canvas.New(width, height)
    ctx := canvas.NewContext(c)
    ctx.DrawPath(0.0, 0.0, p)
    if err := renderers.Write("AV.png", c, canvas.Resolution(dpmm)); err != nil {
        panic(err)
    }
}
```

![Output of AV text](https://github.com/tdewolff/font/raw/refs/heads/master/examples/glyphs/out.png)

### [Merge fonts](https://github.com/tdewolff/font/tree/master/examples/merge)

``` go
import "github.com/tdewolff/font"

// Load two fonts
fontBytes, err := os.ReadFile("DejaVuSerif.otf")
if err != nil {
    panic(err)
}
font2Bytes, err := os.ReadFile("DejaVuSerif-Extended.otf")
if err != nil {
    panic(err)
}

sfnt, err := font.ParseSFNT(fontBytes, 0)
if err != nil {
    panic(err)
}
sfnt2, err := font.ParseSFNT(font2Bytes, 0)
if err != nil {
    panic(err)
}

// Add all glyphs of the second font to the first
options := font.MergeOptions{
    RearrangeCmap: false, // don't rewrite unicode mapping to sequential order
}
if err := sfnt.Merge(sfnt2, options); err != nil {
    panic(err)
}

if err := os.WriteFile("DejaVuSerif-Merged.otf", sfnt.Write(), 0644); err != nil{
    panic(err)
}
```

### [Subset font](https://github.com/tdewolff/font/tree/master/examples/subset)

``` go
import "github.com/tdewolff/font"

fontBytes, err := os.ReadFile("DejaVuSerif.otf")
if err != nil {
    panic(err)
}

sfnt, err := font.ParseSFNT(fontBytes, 0)
if err != nil {
    panic(err)
}

var glyphIDs []uint16
for c := 'A'; c <= 'Z'; c++ {
    if glyphID := sfnt.GlyphIndex(c); glyphID != 0 {
        glyphIDs = append(glyphIDs, glyphID)
    }
}

// Create subset
options := font.SubsetOptions{
    Tables: font.KeepMinTables, // keep only required tables
}
sfntSubset, err := sfnt.Subset(glyphIDs, options)
if err != nil {
    panic(err)
}

if err := os.WriteFile("DejaVuSerif-Subset.otf", sfntSubset.Write(), 0644); err != nil{
    panic(err)
}
```

380132 bytes  =>  6700 bytes (1.7%)

### Convert to TTF/OTF

``` go
import "github.com/tdewolff/font"

// Can be any of TTF, OTF, TTC, WOFF, WOFF2, or EOT
fontBytes, err := os.ReadFile("DejaVuSerif.woff")
if err != nil {
    panic(err)
}

// []byte to []byte
sfnt, err := font.ToSFNT(fontBytes)
if err != nil {
    panic(err)
}

ext := font.Extension(sfnt) // .ttf or .otf
if err := os.WriteFile("DejaVuSerif"+ext, sfnt, 0644); err != nil{
    panic(err)
}
```

## License
Released under the [MIT license](LICENSE.md).
