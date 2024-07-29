package theme

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
)

var fontExts = []string{".otf", ".otc", ".ttf", ".ttc"}

// LoadBuiltin loads fonts from the belowing sources:
// 1. The provided dir.
// 2. embedded font bytes.
// 3. The Gio builtin Go font collection.
func LoadBuiltin(fontDir string, embeddedFonts [][]byte) []font.FontFace {
	var fonts []font.FontFace
	fonts = append(fonts, gofont.Collection()...)

	// load embedded fonts:
	for _, f := range embeddedFonts {
		face, err := loadFont(f)
		if err != nil {
			log.Printf("loading embedded font failed: %v", err)
		}

		fonts = append(fonts, *face)
	}

	fonts = append(fonts, loadFromFs(fontDir)...)

	for _, f := range fonts {
		log.Printf("loaded builtin font face: %s, style: %s, weight: %s", f.Font.Typeface, f.Font.Style, f.Font.Weight)
	}

	return fonts
}

// load fonts from directory
func loadFromFs(fontDir string) []font.FontFace {
	if st, err := os.Stat(fontDir); err != nil || os.IsNotExist(err) || !st.IsDir() {
		return nil
	}

	var fonts []font.FontFace

	entries, err := os.ReadDir(fontDir)
	if err != nil {
		log.Printf("loading fonts from dir failed: %v", err)
		return fonts
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !slices.Contains(fontExts, filepath.Ext(filename)) {
			continue
		}
		ttfData, err := os.ReadFile(filepath.Join(fontDir, filename))
		if err != nil {
			log.Printf("read font %s from dir failed: %v", filename, err)
			continue
		}

		face, err := loadFont(ttfData)
		if err != nil {
			log.Printf("loading font %s failed: %v", filename, err)
			continue
		}
		fonts = append(fonts, *face)
	}

	return fonts
}

func loadFont(ttf []byte) (*font.FontFace, error) {
	faces, err := opentype.ParseCollection(ttf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %v", err)
	}

	return &text.FontFace{
		Font: faces[0].Font,
		Face: faces[0].Face,
	}, nil
}
