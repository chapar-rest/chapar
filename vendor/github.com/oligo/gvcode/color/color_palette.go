package color

import (
	"errors"
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"gioui.org/op"
	"gioui.org/op/paint"
)

// Color encodes a color.NRGBA color which is widely used by Gio.
// It provides method to convert the non-alpha-premultiplied color
// to a color OP used by Gio ops.
type Color struct {
	val uint32
	op  op.CallOp
}

func MakeColor(val color.NRGBA) Color {
	v := uint32(uint32(val.R)<<24 | uint32(val.G)<<16 | uint32(val.B)<<8 | uint32(val.A))
	return Color{val: v}
}

func (c Color) NRGBA() color.NRGBA {
	return color.NRGBA{R: uint8(c.val >> 24), G: uint8(c.val >> 16), B: uint8(c.val >> 8), A: uint8(c.val)}
}

func (c *Color) Op(ops *op.Ops) op.CallOp {
	if c.val == 0 {
		return op.CallOp{}
	}
	if c.op != (op.CallOp{}) {
		return c.op
	}

	if ops == nil {
		ops = new(op.Ops)
	}
	m := op.Record(ops)
	paint.ColorOp{Color: c.NRGBA()}.Add(ops)
	c.op = m.Stop()
	return c.op
}

// MulAlpha applies the alpha to the color and return the modified one.
func (c Color) MulAlpha(alpha uint8) Color {
	// update the alpha value
	a := uint8(uint32(uint8(c.val)) * uint32(alpha) / 0xFF)
	// clear the lower 8 bits and set the one value.
	c.val = (c.val &^ 0xFF) | uint32(a)
	return c
}

func (c Color) IsSet() bool {
	return c.val != 0
}

func (c Color) String() string {
	rgba := c.NRGBA()
	return fmt.Sprintf("Color[R: %d, G: %d, B: %d, A: %d]", rgba.R, rgba.G, rgba.B, rgba.A)
}

// Hex2Color converts a non-alpha-premultiplied hexadecimal color string to Color.
// The hexadecimal color can be in RGB or RGBA format. A "#" prefix is also allowed.
//
// Example formats: "#RRGGBB", "RRGGBBAA", "RRGGBB".
func Hex2Color(hexStr string) (Color, error) {
	hexStr = strings.TrimPrefix(hexStr, "#")

	var channels []uint8

	if len(hexStr) != 6 && len(hexStr) != 8 {
		return Color{}, errors.New("invalid hex color string length (must be 6 or 8 characters)")
	}

	off := 0
	for off < len(hexStr) {
		c, err := parseHexByte(hexStr[off : off+2])
		if err != nil {
			return Color{}, fmt.Errorf("invalid component: %w", err)
		}
		channels = append(channels, c)
		off += 2
	}

	if len(channels) == 3 {
		// Set alpha to full opaque.
		channels = append(channels, 0xFF)
	}

	return MakeColor(color.NRGBA{R: channels[0], G: channels[1], B: channels[2], A: channels[3]}), nil
}

// parseHexByte converts a 2-character hex string (e.g., "FF") to a uint8.
func parseHexByte(hexStr string) (uint8, error) {
	// Base 16, 8-bit unsigned integer
	val, err := strconv.ParseUint(hexStr, 16, 8)
	if err != nil {
		return 0, err
	}
	return uint8(val), nil
}

// ColorPalette manages used color of TextPainter. Color is added and referenced by its
// ID(index) in the palette.
type ColorPalette struct {
	// Foreground provides a default text color for the editor.
	Foreground Color
	// Background provides a default text color for the editor.
	Background Color
	// Color used to highlight the selections.
	SelectColor Color
	// Color used to highlight the current paragraph.
	LineColor Color
	// Color used to paint the line number
	LineNumberColor Color
	// Other colors.
	colors []Color
}

// GetColor retrieves a Color by its ID. ID can be acquired when adding the color to
// the palette.
func (p *ColorPalette) GetColor(id int) Color {
	if id < 0 || id >= len(p.colors) {
		return Color{}
	}

	return p.colors[id]
}

// AddColor adds a color to the palette and return its id(index).
func (p *ColorPalette) AddColor(cl Color) int {
	if idx := slices.IndexFunc(p.colors, func(c Color) bool { return c.val == cl.val }); idx >= 0 {
		return idx
	}

	p.colors = append(p.colors, cl)
	return len(p.colors) - 1
}

// Clear clear all added colors.
func (p *ColorPalette) Clear() {
	p.colors = p.colors[:0]
}
