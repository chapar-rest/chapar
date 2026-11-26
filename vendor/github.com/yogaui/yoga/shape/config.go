package shape

import (
	"bytes"
	"fmt"
	"os"

	"github.com/go-text/typesetting/font"
)

// FaceConfig configures a single text face (UI or editor mono).
//
// Zero values mean "keep the default": an empty Family/File uses the embedded
// font, a zero Size uses the 14px default, zero LetterSpacing adds no tracking,
// and a zero LineHeight keeps the natural line metrics.
type FaceConfig struct {
	Family        string  // system font family; "" = embedded default
	File          string  // path to a .ttf/.otf/.ttc; wins over Family when set
	Size          float32 // logical px; 0 = default (14)
	LetterSpacing float32 // extra logical px added to each glyph advance
	LineHeight    float32 // multiplier of (ascent+descent); 0 = natural line height
}

// FontConfig configures both application faces and the editor tab width.
type FontConfig struct {
	UI       FaceConfig // sans-serif UI chrome face
	Mono     FaceConfig // editor monospace face
	TabWidth int        // tab stop width in columns; 0 = default (4)
}

// resolvedSize returns the logical pixel size, defaulting to logicalFontPx.
func (c FaceConfig) resolvedSize() float32 {
	if c.Size > 0 {
		return c.Size
	}
	return logicalFontPx
}

// loadFace resolves cfg to a font face. An explicit File that fails to load is
// an error; a Family that cannot be found falls back to the embedded face.
func (fs *FontSystem) loadFace(cfg FaceConfig, embedded []byte) (*font.Face, error) {
	if cfg.File != "" {
		faces, err := loadFaceFile(cfg.File, 0)
		if err != nil {
			return nil, fmt.Errorf("load font %q: %w", cfg.File, err)
		}
		return faces, nil
	}
	if cfg.Family != "" {
		if loc, ok := fs.fontMap.FindSystemFont(cfg.Family); ok {
			if face, err := loadFaceFile(loc.File, loc.Index); err == nil {
				return face, nil
			}
		}
		// Family not found or unreadable: fall back to the embedded face.
	}
	return font.ParseTTF(bytes.NewReader(embedded))
}

// loadFaceFile parses a font file (single or collection) and returns the face
// at index.
func loadFaceFile(path string, index uint16) (*font.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	faces, err := font.ParseTTC(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if int(index) >= len(faces) {
		index = 0
	}
	return faces[index], nil
}
