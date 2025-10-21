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

// ColorPalette defines a semantic color system for consistent theming
type ColorPalette struct {
	// Primary colors
	Primary        color.NRGBA
	PrimaryVariant color.NRGBA
	Secondary      color.NRGBA

	// Surface colors
	Surface        color.NRGBA
	SurfaceVariant color.NRGBA
	Background     color.NRGBA

	// Text colors
	OnPrimary    color.NRGBA
	OnSecondary  color.NRGBA
	OnSurface    color.NRGBA
	OnBackground color.NRGBA

	// State colors
	Error   color.NRGBA
	Warning color.NRGBA
	Success color.NRGBA
	Info    color.NRGBA

	// Interactive colors
	Hover    color.NRGBA
	Focus    color.NRGBA
	Selected color.NRGBA
	Disabled color.NRGBA

	// Border and separator colors
	Border        color.NRGBA
	BorderFocused color.NRGBA
	Separator     color.NRGBA

	// Component-specific colors
	Sidebar      color.NRGBA
	SidebarText  color.NRGBA
	TreeView     color.NRGBA
	Dropdown     color.NRGBA
	Menu         color.NRGBA
	Notification color.NRGBA
	Badge        color.NRGBA
}

// Predefined color palettes
var (
	// LightTheme provides a clean, high-contrast color scheme for light mode
	LightTheme = ColorPalette{
		Primary:        rgb(0x1976d2), // Material Blue 700
		PrimaryVariant: rgb(0x1565c0), // Material Blue 800
		Secondary:      rgb(0x424242), // Material Grey 800
		Surface:        rgb(0xffffff), // Pure white
		SurfaceVariant: rgb(0xf5f5f5), // Light grey
		Background:     rgb(0xfafafa), // Very light grey
		OnPrimary:      rgb(0xffffff), // White text
		OnSecondary:    rgb(0xffffff), // White text
		OnSurface:      rgb(0x212121), // Dark grey text
		OnBackground:   rgb(0x212121), // Dark grey text
		Error:          rgb(0xd32f2f), // Material Red 700
		Warning:        rgb(0xf57c00), // Material Orange 700
		Success:        rgb(0x388e3c), // Material Green 700
		Info:           rgb(0x1976d2), // Material Blue 700
		Hover:          rgb(0x000000), // Black with alpha
		Focus:          rgb(0x1976d2), // Primary blue
		Selected:       rgb(0x1976d2), // Primary blue
		Disabled:       rgb(0x9e9e9e), // Material Grey 500
		Border:         rgb(0x9e9e9e), // Darker grey border for better visibility
		BorderFocused:  rgb(0x1976d2), // Primary blue
		Separator:      rgb(0x9e9e9e), // Darker grey separator
		Sidebar:        rgb(0xf5f5f5), // Light grey sidebar
		SidebarText:    rgb(0x424242), // Dark grey text
		TreeView:       rgb(0xf5f5f5), // Light grey tree view
		Dropdown:       rgb(0xffffff), // White dropdown
		Menu:           rgb(0xffffff), // White menu
		Notification:   rgb(0x1976d2), // Primary blue
		Badge:          rgb(0xe3f2fd), // Light blue badge
	}

	// DarkTheme provides an eye-friendly color scheme for dark mode
	DarkTheme = ColorPalette{
		Primary:        rgb(0x64b5f6), // Material Blue 300 (darker blue for active buttons)
		PrimaryVariant: rgb(0x42a5f5), // Material Blue 400 (even darker for variants)
		Secondary:      rgb(0x9e9e9e), // Material Grey 500
		Surface:        rgb(0x1e1e1e), // Dark surface
		SurfaceVariant: rgb(0x2d2d2d), // Slightly lighter dark
		Background:     rgb(0x202120), // Custom dark background as requested
		OnPrimary:      rgb(0x000000), // Black text
		OnSecondary:    rgb(0x000000), // Black text
		OnSurface:      rgb(0xffffff), // White text
		OnBackground:   rgb(0xffffff), // White text
		Error:          rgb(0xef5350), // Material Red 400
		Warning:        rgb(0xffb74d), // Material Orange 300
		Success:        rgb(0x81c784), // Material Green 300
		Info:           rgb(0x90caf9), // Material Blue 200
		Hover:          rgb(0x2d2d2d), // Lighter than background for hover
		Focus:          rgb(0x64b5f6), // Primary blue
		Selected:       rgb(0x64b5f6), // Primary blue
		Disabled:       rgb(0x616161), // Material Grey 600
		Border:         rgb(0x424242), // Dark grey border
		BorderFocused:  rgb(0x64b5f6), // Primary blue
		Separator:      rgb(0x424242), // Dark grey separator
		Sidebar:        rgb(0x404140), // Dark sidebar
		SidebarText:    rgb(0xffffff), // White text
		TreeView:       rgb(0x383939), // Dark tree view
		Dropdown:       rgb(0x2d2d2d), // Dark dropdown
		Menu:           rgb(0x2d2d2d), // Dark menu
		Notification:   rgb(0x64b5f6), // Primary blue
		Badge:          rgb(0x424242), // Dark badge
	}
)

type Theme struct {
	*material.Theme
	isDark  bool
	palette ColorPalette

	// Legacy color fields for backward compatibility
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
	TreeViewBgColor       color.NRGBA
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
	}

	t.Theme.TextSize = unit.Sp(14)
	// Initialize with the appropriate theme
	t.Switch(isDark)
	return t
}

func (t *Theme) Material() *material.Theme {
	return t.Theme
}

func (t *Theme) Switch(isDark bool) *material.Theme {
	t.isDark = isDark

	// Set the appropriate color palette
	if isDark {
		t.SetPalette(DarkTheme)
	} else {
		t.SetPalette(LightTheme)
	}

	// Update material theme palette for compatibility
	t.Theme.Palette.Fg = t.palette.OnSurface
	t.Theme.Palette.Bg = t.palette.Background
	t.Theme.Palette.ContrastBg = t.palette.Primary
	t.Theme.Palette.ContrastFg = t.palette.OnPrimary

	return t.Theme
}

func (t *Theme) IsDark() bool {
	return t.isDark
}

// Palette returns the current color palette
func (t *Theme) Palette() ColorPalette {
	return t.palette
}

// MaterialPalette returns the material theme palette for backward compatibility
func (t *Theme) MaterialPalette() *material.Palette {
	return &t.Theme.Palette
}

// SetPalette sets a new color palette and updates all legacy colors
func (t *Theme) SetPalette(palette ColorPalette) {
	t.palette = palette
	t.updateLegacyColors()
}

// updateLegacyColors updates all legacy color fields from the current palette
func (t *Theme) updateLegacyColors() {
	t.LoaderColor = t.palette.OnSurface
	t.BorderColor = t.palette.Border
	t.BorderColorFocused = t.palette.BorderFocused
	t.TextColor = t.palette.OnSurface
	t.ButtonTextColor = t.palette.OnPrimary
	t.ActionButtonBgColor = t.palette.Primary
	t.DeleteButtonBgColor = t.palette.Error
	t.SwitchBgColor = t.palette.Primary
	t.TabInactiveColor = t.palette.Primary
	t.SeparatorColor = t.palette.Separator
	t.SideBarBgColor = t.palette.Sidebar
	t.SideBarTextColor = t.palette.SidebarText
	t.TreeViewBgColor = t.palette.TreeView
	t.TableBorderColor = t.palette.Border
	t.CheckBoxColor = t.palette.Primary
	t.RequestMethodColor = t.palette.Success
	t.DropDownMenuBgColor = t.palette.Dropdown
	t.MenuBgColor = t.palette.Menu
	t.TextSelectionColor = t.palette.Selected
	t.NotificationBgColor = t.palette.Notification
	t.NotificationTextColor = t.palette.OnPrimary
	t.ResponseStatusColor = t.palette.Success
	t.ErrorColor = t.palette.Error
	t.WarningColor = t.palette.Warning
	t.BadgeBgColor = t.palette.Badge
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
