package syntax

import (
	"fmt"
)

type TextStyle uint8

const (
	Bold TextStyle = 1 << iota
	Italic
	Underline
	Squiggle
	Strikethrough
	Border
)

func (s TextStyle) HasStyle(mask TextStyle) bool {
	return s&mask > 0
}

const (
	textStyleOffset  = 0
	backgroundOffset = 6
	foregroundOffset = 14
	tokenTypeOffset  = 22

	tokenTypeMask  = 0b00011111_11000000_00000000_00000000
	foregroundMask = 0b00000000_00011111_11000000_00000000
	backgroundMask = 0b00000000_00000000_00111111_11000000
	textStyleMask  = 0b00000000_00000000_00000000_00111111
)

// StyleMeta applies a bit packed binary format to encode tokens from
// syntax lexer, to be used as styles for text rendering. This is like
// TokenMetadata in Monaco/vscode.
//
// It uses 4 bytes to hold the metadata, and the layout is as follows:
// Bits:  31   ...   0
// [3][7][8][8][6] = 32
// |  |  |  |  |
// |  |  |  |  └── Text style flags (6bits, bold, italic, underline, Squiggle, strikethrough, border)
// |  |  |  └───── Background color ID (8bits, 0–255)
// |  |  └──────── Foreground color ID (8bits, 0–255)
// |  └─────────── Token type (7bits, 0–127)
// └────────────── reserved bits (3bits)
//
// The color IDs are mapped to indices of color palette.
type StyleMeta uint32

func (t StyleMeta) TokenType() int {
	return int(t & tokenTypeMask >> tokenTypeOffset)
}

func (t StyleMeta) Foreground() int {
	return int(t & foregroundMask >> foregroundOffset)
}

func (t StyleMeta) Background() int {
	return int(t & backgroundMask >> backgroundOffset)
}

func (t StyleMeta) TextStyle() TextStyle {
	return TextStyle(t & textStyleMask >> textStyleOffset)
}

func (t StyleMeta) String() string {
	return fmt.Sprintf("Type=%d FG=%d BG=%d Style=%04b",
		t.TokenType(), t.Foreground(), t.Background(), t.TextStyle())
}

func packTokenStyle(tokenType int, fg, bg int, textStyles TextStyle) StyleMeta {
	s := StyleMeta(0)

	s |= StyleMeta(tokenType << tokenTypeOffset)
	s |= StyleMeta(fg << foregroundOffset)
	s |= StyleMeta(bg << backgroundOffset)
	s |= StyleMeta(textStyles << textStyleOffset)
	return s
}

type TokenStyle struct {
	// offset of the start rune in the document.
	Start int
	// offset of the end rune in the document.
	End   int
	Style StyleMeta
}
