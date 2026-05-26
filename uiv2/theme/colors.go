package theme

import (
	"image/color"

	"github.com/chapar-rest/chapar/internal/domain"
)

var (
	White       = rgb(0xffffff)
	Black       = rgb(0x000000)
	LightGreen  = rgb(0x8bc34a)
	LightRed    = rgb(0xff7373)
	LightYellow = rgb(0xffe073)
	LightBlue   = rgb(0x4589f5)
	LightPurple = rgb(0x9c27b0)
)

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{
		A: 0xff,
		R: uint8(c >> 16),
		G: uint8(c >> 8),
		B: uint8(c),
	}
}

func ContrastText(bg color.NRGBA) color.NRGBA {
	lum := (299*int(bg.R) + 587*int(bg.G) + 114*int(bg.B)) / 1000
	if lum > 128 {
		return Black
	}
	return White
}

func GetRequestPrefixColor(method string) color.NRGBA {
	switch method {
	case "gRPC", domain.RequestMethodGET:
		return LightGreen
	case domain.RequestMethodPOST:
		return LightYellow
	case domain.RequestMethodPUT:
		return LightBlue
	case domain.RequestMethodDELETE:
		return LightRed
	case domain.RequestMethodPATCH:
		return LightPurple
	case domain.RequestMethodOPTIONS:
		return color.NRGBA{R: 0x00, G: 0x80, B: 0x80, A: 0xff}
	case domain.RequestMethodHEAD:
		return color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
	default:
		return color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
	}
}
