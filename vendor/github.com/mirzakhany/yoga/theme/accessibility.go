package theme

// Themes in this file exist for readers the ordinary palettes do not serve:
// people who need maximum contrast, and people who cannot rely on a red/green
// distinction. Every other shipped theme encodes Error as red and Success as
// green, which is the one pairing roughly 8% of men cannot separate.

// yogaHighContrastLight is the light counterpart to yoga-high-contrast, for
// readers who need maximum contrast but find a black workspace uncomfortable.
// Shipping only a dark high-contrast theme leaves light-sensitivity and
// glare-sensitivity users with nothing; both desktop platforms ship both.
func yogaHighContrastLight() Theme {
	t := baseTheme("yoga-high-contrast-light", false)
	t.Surface = rgb(255, 255, 255)
	t.Chrome = rgb(245, 245, 245)
	t.ChromeMuted = rgb(232, 232, 232)
	t.Foreground = rgb(0, 0, 0)
	t.ForegroundMuted = rgb(60, 60, 60)
	t.ForegroundSubtle = rgb(90, 90, 90)
	t.ForegroundDisabled = rgba(90, 90, 90, 0.60)
	t.Accent = rgb(0, 51, 153)
	t.AccentHover = rgb(0, 45, 135)
	t.AccentPressed = rgb(0, 39, 116)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(60, 60, 60)
	t.BorderStrong = rgb(0, 0, 0)
	t.BorderControl = rgb(0, 0, 0)
	t.ListHover = rgb(214, 214, 214)
	t.ListActive = rgb(184, 184, 184)
	t.FocusRing = rgb(0, 51, 153)
	t.Selection = rgb(176, 202, 255)
	t.ScrollTrack = rgb(232, 232, 232)
	t.ScrollThumb = rgb(90, 90, 90)
	t.ScrollThumbHover = rgb(0, 51, 153)
	t.Error = rgb(168, 0, 0)
	t.Warning = rgb(122, 74, 0)
	t.Success = rgb(0, 92, 40)
	// Sharper chrome, matching the dark high-contrast theme.
	t.Stroke = Stroke{Thin: DefaultStroke().Thick, Thick: DefaultStroke().Thicker, Thicker: 4}
	t.Radius = Radius{None: 0, Small: 0, Medium: 2, Large: 4, XLarge: 6, Circular: 9999}
	t.Syntax = syntax(
		rgb(0, 0, 0), rgb(140, 0, 0), rgb(0, 92, 40),
		rgb(80, 80, 80), rgb(122, 60, 0), rgb(0, 51, 153))
	return finishTheme(t)
}

// Okabe-Ito palette, designed so every pair stays distinguishable under
// protanopia, deuteranopia, and tritanopia. The colorblind-safe themes below
// carry status meaning on these hues instead of on red versus green.
var (
	okabeVermillion  = rgb(213, 94, 0)    // error
	okabeOrange      = rgb(230, 159, 0)   // warning
	okabeBluishGreen = rgb(0, 158, 115)   // success
	okabeBlue        = rgb(0, 114, 178)   // info / accent
	okabeSkyBlue     = rgb(86, 180, 233)  // accent on dark
	okabePurple      = rgb(204, 121, 167) // syntax
	okabeYellow      = rgb(240, 228, 66)  // syntax
)

// yogaColorblindDark is a dark theme whose status colors come from the
// Okabe-Ito palette, so Error, Warning, and Success stay separable without
// color vision. Shape and text still carry the meaning; this only makes sure
// the color is not actively misleading.
func yogaColorblindDark() Theme {
	t := baseTheme("yoga-colorblind-dark", true)
	t.Surface = rgb(24, 26, 31)
	t.Chrome = rgb(33, 36, 43)
	t.ChromeMuted = rgb(45, 49, 58)
	t.Foreground = rgb(226, 230, 238)
	t.ForegroundMuted = rgb(155, 162, 175)
	t.ForegroundSubtle = rgb(120, 128, 142)
	t.ForegroundDisabled = rgba(155, 162, 175, 0.50)
	t.Accent = okabeSkyBlue
	t.AccentHover = rgb(106, 189, 236)
	t.AccentPressed = rgb(127, 198, 238)
	t.AccentForeground = rgb(12, 20, 30)
	t.Border = rgb(56, 61, 72)
	t.BorderStrong = rgb(78, 85, 99)
	t.ListHover = rgb(45, 49, 58)
	t.ListActive = rgb(60, 66, 78)
	t.Selection = rgb(24, 68, 102)
	t.ScrollTrack = rgb(30, 33, 40)
	t.ScrollThumb = rgb(110, 118, 132)
	t.ScrollThumbHover = okabeSkyBlue
	t.Error = okabeVermillion
	t.Warning = okabeOrange
	t.Success = okabeBluishGreen
	t.Info = okabeSkyBlue
	t.Syntax = syntax(
		rgb(226, 230, 238), okabePurple, okabeBluishGreen,
		rgb(140, 148, 162), okabeOrange, okabeSkyBlue)
	return finishTheme(t)
}

// yogaColorblindLight is the light half of the colorblind-safe family.
func yogaColorblindLight() Theme {
	t := baseTheme("yoga-colorblind-light", false)
	t.Surface = rgb(252, 252, 253)
	t.Chrome = rgb(241, 242, 245)
	t.ChromeMuted = rgb(228, 230, 235)
	t.Foreground = rgb(26, 29, 35)
	t.ForegroundMuted = rgb(92, 98, 110)
	t.ForegroundSubtle = rgb(122, 129, 141)
	t.ForegroundDisabled = rgba(92, 98, 110, 0.55)
	t.Accent = okabeBlue
	t.AccentHover = rgb(0, 100, 157)
	t.AccentPressed = rgb(0, 87, 135)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(210, 214, 221)
	t.BorderStrong = rgb(150, 157, 168)
	t.ListHover = rgb(222, 226, 233)
	t.ListActive = rgb(202, 208, 218)
	t.Selection = rgb(178, 210, 240)
	t.ScrollTrack = rgb(228, 230, 235)
	t.ScrollThumb = rgb(125, 132, 143)
	t.ScrollThumbHover = okabeBlue
	t.Error = rgb(168, 74, 0)
	t.Warning = rgb(140, 96, 0)
	t.Success = rgb(0, 110, 80)
	t.Info = okabeBlue
	t.Syntax = syntax(
		rgb(26, 29, 35), rgb(150, 70, 120), rgb(0, 110, 80),
		rgb(105, 112, 124), rgb(150, 92, 0), okabeBlue)
	return finishTheme(t)
}
