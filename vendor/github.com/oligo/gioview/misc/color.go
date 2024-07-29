package misc

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

type Color struct {
	color.NRGBA
}

func argb(c uint32) color.NRGBA {
	return color.NRGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}

// construct a Color from hex string, int, or wrapping a NRGBA.
func NewColor(value interface{}) Color {
	switch value := value.(type) {
	case color.NRGBA:
		return Color{
			NRGBA: value,
		}
	case string:
		if !strings.HasPrefix(value, "#") || len(value) < 7 {
			return Color{}
		}

		value = strings.TrimPrefix(value, "#")
		bitSize := 24
		if len(value) == 6 {
			bitSize = 24
		} else if len(value) == 8 {
			bitSize = 32
		}

		val, err := strconv.ParseUint(value, 16, bitSize)
		if err != nil {
			return Color{}
		}

		return Color{
			NRGBA: argb(0xff000000 | uint32(val)),
		}

	case uint32:
		return Color{
			NRGBA: argb(0xff000000 | value),
		}
	case int:
		return Color{
			NRGBA: argb(0xff000000 | uint32(value)),
		}
	default:
		panic("wrong color value type: " + fmt.Sprintf("%T", value))
	}
}

// Calculate relative luminance of a RGBA color.
func (c Color) Luminance() float32 {
	return float32(c.R)*0.2126 + float32(c.G)*0.7152 + float32(c.B)*0.0722
}

func HexColor(v interface{}) color.NRGBA {
	return NewColor(v).NRGBA
}
