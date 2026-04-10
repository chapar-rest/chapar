package chapartheme

import (
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
)

var (
	White       = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	Black       = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
	LightGreen  = color.NRGBA{R: 0x8b, G: 0xc3, B: 0x4a, A: 0xff}
	LightRed    = color.NRGBA{R: 0xff, G: 0x73, B: 0x73, A: 0xff}
	LightYellow = color.NRGBA{R: 0xff, G: 0xe0, B: 0x73, A: 0xff}
	LightBlue   = color.NRGBA{R: 0x45, G: 0x89, B: 0xf5, A: 0xff}
	LightPurple = color.NRGBA{R: 0x9c, G: 0x27, B: 0xb0, A: 0xff}
)

type Theme struct {
	*material.Theme
	isDark bool

	LoaderColor           color.NRGBA
	BorderColor           color.NRGBA
	BorderColorFocused    color.NRGBA
	TextColor             color.NRGBA
	ButtonTextColor       color.NRGBA
	ActionButtonBgColor   color.NRGBA
	DeleteButtonBgColor   color.NRGBA
	SwitchBgColor         color.NRGBA
	TabInactiveColor      color.NRGBA
	SeparatorColor        color.NRGBA
	SideBarBgColor        color.NRGBA
	SideBarTextColor      color.NRGBA
	TableBorderColor      color.NRGBA
	CheckBoxColor         color.NRGBA
	RequestMethodColor    color.NRGBA
	DropDownMenuBgColor   color.NRGBA
	MenuBgColor           color.NRGBA
	TextSelectionColor    color.NRGBA
	NotificationBgColor   color.NRGBA
	NotificationTextColor color.NRGBA
	ResponseStatusColor   color.NRGBA
	ErrorColor            color.NRGBA
	WarningColor          color.NRGBA
	BadgeBgColor          color.NRGBA
}

func New(material *material.Theme, isDark bool) *Theme {
	t := &Theme{
		Theme: material,
		//SideBarBgColor:   rgb(0x202224),
		//SideBarTextColor: rgb(0xffffff),
	}

	t.Theme.TextSize = unit.Sp(14)
	// default theme is dark
	t.Switch("dark")
	return t
}

func (t *Theme) Material() *material.Theme {
	return t.Theme
}

func (t *Theme) Switch(themeName string) *material.Theme {
	switch themeName {
	case "dark":
		t.isDark = true
		t.Theme.Palette.Fg = rgb(0xd7dade)
		t.LoaderColor = rgb(0xd7dade)
		t.Theme.Palette.Bg = rgb(0x202224)
		t.Theme.Palette.ContrastBg = rgb(0x202224)
		t.Theme.Palette.ContrastFg = rgb(0xffffff)
		t.BorderColorFocused = rgb(0xffffff)
		t.TextColor = rgb(0x8b8e95)
		t.BorderColor = rgb(0x6c6f76)
		t.TabInactiveColor = rgb(0x4589f5)
		t.ActionButtonBgColor = rgb(0x4589f5)
		t.SwitchBgColor = rgb(0x4589f5)
		t.TextColor = rgb(0xffffff)
		t.ButtonTextColor = rgb(0xffffff)
		t.SeparatorColor = rgb(0x2b2d31)
		t.TableBorderColor = rgb(0x2b2d31)
		t.CheckBoxColor = rgb(0xb0b3b8)
		t.RequestMethodColor = rgb(0x8bc34a)
		t.DropDownMenuBgColor = rgb(0x2b2d31)
		t.MenuBgColor = rgb(0x2b2d31)
		t.TextSelectionColor = rgb(0x6380ad)
		t.NotificationBgColor = rgb(0x4589f5)
		t.NotificationTextColor = rgb(0xffffff)
		t.ResponseStatusColor = rgb(0x8bc34a)
		t.ErrorColor = rgb(0xff7373)
		t.WarningColor = rgb(0xffe073)
		t.BadgeBgColor = rgb(0x2b2d31)
		t.DeleteButtonBgColor = rgb(0xff7373)
		t.SideBarTextColor = rgb(0xffffff)
	case "github-dark":
		t.isDark = true
		t.Theme.Palette.Fg = rgb(0xc9d1d9)
		t.LoaderColor = rgb(0xc9d1d9)
		t.Theme.Palette.Bg = rgb(0x0d1117)
		t.Theme.Palette.ContrastBg = rgb(0x161b22)
		t.Theme.Palette.ContrastFg = rgb(0xc9d1d9)
		t.BorderColorFocused = rgb(0x58a6ff)
		t.BorderColor = rgb(0x30363d)
		t.TextColor = rgb(0xc9d1d9)
		t.ButtonTextColor = rgb(0xffffff)
		t.TabInactiveColor = rgb(0x8b949e)
		t.ActionButtonBgColor = rgb(0x238636)
		t.SwitchBgColor = rgb(0x238636)
		t.SeparatorColor = rgb(0x21262d)
		t.TableBorderColor = rgb(0x30363d)
		t.CheckBoxColor = rgb(0x58a6ff)
		t.RequestMethodColor = rgb(0x3fb950)
		t.DropDownMenuBgColor = rgb(0x161b22)
		t.MenuBgColor = rgb(0x161b22)
		t.TextSelectionColor = rgb(0x264f78)
		t.NotificationBgColor = rgb(0x1f6feb)
		t.NotificationTextColor = rgb(0xffffff)
		t.ResponseStatusColor = rgb(0x3fb950)
		t.ErrorColor = rgb(0xf85149)
		t.WarningColor = rgb(0xd29922)
		t.BadgeBgColor = rgb(0x21262d)
		t.DeleteButtonBgColor = rgb(0xda3633)
		t.SideBarTextColor = rgb(0xc9d1d9)
	case "light":
		t.isDark = false
		t.LoaderColor = rgb(0x000000)
		t.Theme.Palette.Fg = rgb(0x000000)
		t.Theme.Palette.Bg = rgb(0xffffff)
		t.Theme.Palette.ContrastBg = rgb(0xd4d7d9)
		t.Theme.Palette.ContrastFg = rgb(0x000000)
		t.TextColor = rgb(0x8b8e95)
		t.BorderColorFocused = rgb(0x4589f5)
		t.BorderColor = rgb(0x6c6f76)
		t.TabInactiveColor = rgb(0x4589f5)
		t.ActionButtonBgColor = rgb(0xd4d7d9)
		t.SwitchBgColor = rgb(0x4589f5)
		t.ButtonTextColor = rgb(0x000000)
		t.SeparatorColor = rgb(0xbcbebf)
		t.TableBorderColor = rgb(0xb0b3b8)
		t.CheckBoxColor = rgb(0x80878c)
		t.RequestMethodColor = rgb(0x007518)
		t.DropDownMenuBgColor = rgb(0x696969)
		t.MenuBgColor = rgb(0x2b2d31)
		t.TextSelectionColor = rgb(0xccd3de)
		t.NotificationBgColor = rgb(0x4589f5)
		t.NotificationTextColor = rgb(0xffffff)
		t.ResponseStatusColor = rgb(0x007518)
		t.ErrorColor = rgb(0xff7373)
		t.WarningColor = rgb(0xffe073)
		t.BadgeBgColor = rgb(0x2b2d31)
		t.DeleteButtonBgColor = rgb(0xff7373)
		t.SideBarTextColor = rgb(0x000000)
	case "github-light":
		t.isDark = false
		t.LoaderColor = rgb(0x24292f)
		t.Theme.Palette.Fg = rgb(0x24292f)
		t.Theme.Palette.Bg = rgb(0xffffff)
		t.Theme.Palette.ContrastBg = rgb(0xf6f8fa)
		t.Theme.Palette.ContrastFg = rgb(0x24292f)
		t.TextColor = rgb(0x57606a)
		t.BorderColorFocused = rgb(0x0969da)
		t.BorderColor = rgb(0xd0d7de)
		t.TabInactiveColor = rgb(0x57606a)
		t.ActionButtonBgColor = rgb(0x2da44e)
		t.SwitchBgColor = rgb(0x2da44e)
		t.ButtonTextColor = rgb(0xffffff)
		t.SeparatorColor = rgb(0xd8dee4)
		t.TableBorderColor = rgb(0xd0d7de)
		t.CheckBoxColor = rgb(0x0969da)
		t.RequestMethodColor = rgb(0x1a7f37)
		t.DropDownMenuBgColor = rgb(0x24292f)
		t.MenuBgColor = rgb(0xf6f8fa)
		t.TextSelectionColor = rgb(0xddf4ff)
		t.NotificationBgColor = rgb(0x0969da)
		t.NotificationTextColor = rgb(0xffffff)
		t.ResponseStatusColor = rgb(0x1a7f37)
		t.ErrorColor = rgb(0xcf222e)
		t.WarningColor = rgb(0x9a6700)
		t.BadgeBgColor = rgb(0xeaeef2)
		t.DeleteButtonBgColor = rgb(0xcf222e)
		t.SideBarTextColor = rgb(0x24292f)
	}

	return t.Theme
}

func (t *Theme) IsDark() bool {
	return t.isDark
}

func rgb(c uint32) color.NRGBA {
	return argb(0xff000000 | c)
}

func argb(c uint32) color.NRGBA {
	return color.NRGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}

func GetRequestPrefixColor(method string) color.NRGBA {
	switch method {
	case "gRPC":
		return LightGreen
	case domain.RequestMethodGET:
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
