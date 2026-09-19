package theme

import (
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/render"
)

// rgb is a small alias for opaque colors (alpha 255).
func rgb(r, g, b uint8) render.Color { return render.RGBA8(r, g, b, 255) }

// syntax builds a syntax color map from per-class colors.
func syntax(def, keyword, str, comment, number, typ render.Color) map[highlight.ColorClass]render.Color {
	return map[highlight.ColorClass]render.Color{
		highlight.ClassDefault: def,
		highlight.ClassKeyword: keyword,
		highlight.ClassString:  str,
		highlight.ClassComment: comment,
		highlight.ClassNumber:  number,
		highlight.ClassType:    typ,
	}
}

// baseTheme returns a Theme with shared non-color defaults filled in.
func baseTheme(name string, dark bool) Theme {
	t := Theme{Name: name, Dark: dark}
	t.Spacing = DefaultSpacing()
	t.Radius = DefaultRadius()
	t.Stroke = DefaultStroke()
	t.Typography = DefaultTypography()
	t.Metrics = DefaultComponentMetrics()
	if dark {
		t.Elevation = DefaultElevationDark()
	} else {
		t.Elevation = DefaultElevationLight()
	}
	return t
}

// finishTheme syncs legacy aliases and returns the theme.
func finishTheme(t Theme) Theme {
	syncLegacyFromYoga(&t)
	return t
}

// builtins returns all themes shipped with the framework.
func builtins() []Theme {
	return []Theme{
		yogaDark(),
		yogaLight(),
		yogaSystem(),
		yogaHighContrast(),
		yogaMidnight(),
		githubDark(),
		githubLight(),
		catppuccin(),
		catppuccinLatte(),
		dracula(),
		nord(),
		solarizedDark(),
		solarizedLight(),
		gruvboxDark(),
		gruvboxLight(),
		monokai(),
		everforestDark(),
		everforestLight(),
	}
}

func githubDark() Theme {
	t := baseTheme("github-dark", true)
	t.Surface = rgb(13, 17, 23)
	t.Chrome = rgb(22, 27, 34)
	t.ChromeMuted = rgb(33, 38, 45)
	t.Foreground = rgb(230, 237, 243)
	t.ForegroundMuted = rgb(125, 133, 144)
	t.ForegroundSubtle = rgb(90, 98, 110)
	t.ForegroundDisabled = rgba(125, 133, 144, 0.45)
	t.Accent = rgb(47, 129, 247)
	t.AccentHover = rgb(65, 145, 255)
	t.AccentPressed = rgb(31, 111, 235)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(48, 54, 61)
	t.BorderStrong = rgb(68, 76, 86)
	t.ListHover = rgb(33, 38, 45)
	t.ListActive = rgb(48, 54, 61)
	t.FocusRing = rgb(47, 129, 247)
	t.Selection = rgb(56, 90, 140)
	t.ScrollTrack = rgb(28, 33, 40)
	t.ScrollThumb = rgb(90, 98, 110)
	t.ScrollThumbHover = rgb(47, 129, 247)
	t.Error = rgb(248, 81, 73)
	t.Warning = rgb(210, 153, 34)
	t.Success = rgb(63, 185, 80)
	t.Syntax = syntax(
		rgb(230, 237, 243), rgb(255, 123, 114), rgb(165, 214, 255),
		rgb(139, 148, 158), rgb(121, 192, 255), rgb(255, 166, 87))
	return finishTheme(t)
}

func githubLight() Theme {
	t := baseTheme("github-light", false)
	t.Surface = rgb(255, 255, 255)
	t.Chrome = rgb(246, 248, 250)
	t.ChromeMuted = rgb(234, 238, 242)
	t.Foreground = rgb(31, 35, 40)
	t.ForegroundMuted = rgb(101, 109, 118)
	t.ForegroundSubtle = rgb(140, 148, 158)
	t.ForegroundDisabled = rgba(101, 109, 118, 0.45)
	t.Accent = rgb(9, 105, 218)
	t.AccentHover = rgb(26, 127, 255)
	t.AccentPressed = rgb(7, 88, 185)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(208, 215, 222)
	t.BorderStrong = rgb(175, 184, 195)
	t.ListHover = rgb(234, 238, 242)
	t.ListActive = rgb(215, 222, 228)
	t.FocusRing = rgb(9, 105, 218)
	t.Selection = rgb(184, 215, 255)
	t.ScrollTrack = rgb(225, 228, 233)
	t.ScrollThumb = rgb(155, 162, 172)
	t.ScrollThumbHover = rgb(9, 105, 218)
	t.Error = rgb(207, 34, 46)
	t.Warning = rgb(154, 103, 0)
	t.Success = rgb(26, 127, 55)
	t.Syntax = syntax(
		rgb(31, 35, 40), rgb(207, 34, 46), rgb(10, 48, 105),
		rgb(110, 119, 129), rgb(5, 80, 174), rgb(149, 56, 0))
	return finishTheme(t)
}

func catppuccin() Theme {
	t := baseTheme("catppuccin", true)
	t.Surface = rgb(30, 30, 46)    // base
	t.Chrome = rgb(24, 24, 37)     // mantle — darker chrome
	t.ChromeMuted = rgb(49, 50, 68) // surface0
	t.Foreground = rgb(205, 214, 244)
	t.ForegroundMuted = rgb(166, 173, 200)
	t.ForegroundSubtle = rgb(127, 132, 156)
	t.ForegroundDisabled = rgba(166, 173, 200, 0.45)
	t.Accent = rgb(137, 180, 250)
	t.AccentHover = rgb(155, 195, 255)
	t.AccentPressed = rgb(110, 160, 235)
	t.AccentForeground = rgb(30, 30, 46)
	t.Border = rgb(69, 71, 90)
	t.BorderStrong = rgb(88, 91, 112)
	t.ListHover = rgb(49, 50, 68)
	t.ListActive = rgb(69, 71, 90)
	t.FocusRing = rgb(137, 180, 250)
	t.Selection = rgb(88, 91, 112)
	t.ScrollTrack = rgb(36, 36, 54)
	t.ScrollThumb = rgb(120, 125, 148)
	t.ScrollThumbHover = rgb(137, 180, 250)
	t.Error = rgb(243, 139, 168)
	t.Warning = rgb(249, 226, 175)
	t.Success = rgb(166, 227, 161)
	t.Syntax = syntax(
		rgb(205, 214, 244), rgb(203, 166, 247), rgb(166, 227, 161),
		rgb(140, 145, 170), rgb(250, 179, 135), rgb(249, 226, 175))
	return finishTheme(t)
}

func catppuccinLatte() Theme {
	t := baseTheme("catppuccin-latte", false)
	t.Surface = rgb(239, 241, 245)    // base
	t.Chrome = rgb(230, 233, 239)     // mantle
	t.ChromeMuted = rgb(220, 224, 232) // surface0
	t.Foreground = rgb(76, 79, 105)
	t.ForegroundMuted = rgb(108, 111, 133)
	t.ForegroundSubtle = rgb(140, 143, 161)
	t.ForegroundDisabled = rgba(108, 111, 133, 0.45)
	t.Accent = rgb(30, 102, 245)
	t.AccentHover = rgb(45, 120, 255)
	t.AccentPressed = rgb(22, 85, 210)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(204, 208, 218)
	t.BorderStrong = rgb(172, 176, 195)
	t.ListHover = rgb(220, 224, 232)
	t.ListActive = rgb(204, 208, 218)
	t.FocusRing = rgb(30, 102, 245)
	t.Selection = rgb(172, 195, 255)
	t.ScrollTrack = rgb(220, 224, 232)
	t.ScrollThumb = rgb(156, 160, 176)
	t.ScrollThumbHover = rgb(30, 102, 245)
	t.Error = rgb(210, 15, 57)
	t.Warning = rgb(223, 142, 29)
	t.Success = rgb(64, 160, 43)
	t.Syntax = syntax(
		rgb(76, 79, 105), rgb(136, 57, 239), rgb(64, 160, 43),
		rgb(140, 143, 161), rgb(254, 100, 11), rgb(4, 165, 229))
	return finishTheme(t)
}

func dracula() Theme {
	t := baseTheme("dracula", true)
	t.Surface = rgb(40, 42, 54)
	t.Chrome = rgb(33, 34, 44)
	t.ChromeMuted = rgb(68, 71, 90)
	t.Foreground = rgb(248, 248, 242)
	t.ForegroundMuted = rgb(98, 114, 164)
	t.ForegroundSubtle = rgb(130, 145, 185)
	t.ForegroundDisabled = rgba(98, 114, 164, 0.45)
	t.Accent = rgb(189, 147, 249)
	t.AccentHover = rgb(205, 165, 255)
	t.AccentPressed = rgb(165, 125, 230)
	t.AccentForeground = rgb(40, 42, 54)
	t.Border = rgb(68, 71, 90)
	t.BorderStrong = rgb(98, 102, 125)
	t.ListHover = rgb(68, 71, 90)
	t.ListActive = rgb(88, 91, 112)
	t.FocusRing = rgb(189, 147, 249)
	t.Selection = rgb(68, 71, 90)
	t.ScrollTrack = rgb(50, 52, 68)
	t.ScrollThumb = rgb(120, 125, 148)
	t.ScrollThumbHover = rgb(189, 147, 249)
	t.Error = rgb(255, 85, 85)
	t.Warning = rgb(241, 250, 140)
	t.Success = rgb(80, 250, 123)
	t.Syntax = syntax(
		rgb(248, 248, 242), rgb(255, 121, 198), rgb(241, 250, 140),
		rgb(145, 158, 200), rgb(189, 147, 249), rgb(139, 233, 253))
	return finishTheme(t)
}

func nord() Theme {
	t := baseTheme("nord", true)
	t.Surface = rgb(46, 52, 64)
	t.Chrome = rgb(59, 66, 82)
	t.ChromeMuted = rgb(67, 76, 94)
	t.Foreground = rgb(216, 222, 233)
	t.ForegroundMuted = rgb(123, 136, 161)
	t.ForegroundSubtle = rgb(94, 108, 138)
	t.ForegroundDisabled = rgba(123, 136, 161, 0.45)
	t.Accent = rgb(136, 192, 208)
	t.AccentHover = rgb(152, 205, 220)
	t.AccentPressed = rgb(110, 170, 190)
	t.AccentForeground = rgb(46, 52, 64)
	t.Border = rgb(76, 86, 106)
	t.BorderStrong = rgb(94, 108, 138)
	t.ListHover = rgb(67, 76, 94)
	t.ListActive = rgb(76, 86, 106)
	t.FocusRing = rgb(136, 192, 208)
	t.Selection = rgb(76, 86, 106)
	t.ScrollTrack = rgb(55, 62, 76)
	t.ScrollThumb = rgb(110, 122, 145)
	t.ScrollThumbHover = rgb(136, 192, 208)
	t.Error = rgb(191, 97, 106)
	t.Warning = rgb(235, 203, 139)
	t.Success = rgb(163, 190, 140)
	t.Syntax = syntax(
		rgb(216, 222, 233), rgb(129, 161, 193), rgb(163, 190, 140),
		rgb(155, 170, 198), rgb(180, 142, 173), rgb(143, 188, 187))
	return finishTheme(t)
}

func solarizedDark() Theme {
	t := baseTheme("solarized-dark", true)
	t.Surface = rgb(0, 43, 54)
	t.Chrome = rgb(7, 54, 66)
	t.ChromeMuted = rgb(14, 63, 76)
	t.Foreground = rgb(131, 148, 150)
	t.ForegroundMuted = rgb(120, 148, 155)
	t.ForegroundSubtle = rgb(100, 130, 140)
	t.ForegroundDisabled = rgba(120, 148, 155, 0.45)
	t.Accent = rgb(38, 139, 210)
	t.AccentHover = rgb(55, 155, 225)
	t.AccentPressed = rgb(25, 115, 185)
	t.AccentForeground = rgb(0, 26, 34)
	t.Border = rgb(72, 98, 108)
	t.BorderStrong = rgb(90, 120, 130)
	t.ListHover = rgb(18, 68, 84)
	t.ListActive = rgb(28, 80, 98)
	t.FocusRing = rgb(38, 139, 210)
	t.Selection = rgb(0, 80, 130)
	t.ScrollTrack = rgb(5, 48, 58)
	t.ScrollThumb = rgb(100, 120, 128)
	t.ScrollThumbHover = rgb(38, 139, 210)
	t.Error = rgb(220, 50, 47)
	t.Warning = rgb(181, 137, 0)
	t.Success = rgb(133, 153, 0)
	t.Syntax = syntax(
		rgb(131, 148, 150), rgb(133, 153, 0), rgb(42, 161, 152),
		rgb(110, 135, 143), rgb(211, 54, 130), rgb(181, 137, 0))
	return finishTheme(t)
}

func solarizedLight() Theme {
	t := baseTheme("solarized-light", false)
	t.Surface = rgb(253, 246, 227)    // base3
	t.Chrome = rgb(238, 232, 213)     // base2
	t.ChromeMuted = rgb(233, 226, 207) // between base2 and base3
	t.Foreground = rgb(101, 123, 131)  // base00
	t.ForegroundMuted = rgb(88, 110, 117)
	t.ForegroundSubtle = rgb(131, 148, 150)
	t.ForegroundDisabled = rgba(88, 110, 117, 0.45)
	t.Accent = rgb(38, 139, 210)
	t.AccentHover = rgb(55, 155, 225)
	t.AccentPressed = rgb(25, 115, 185)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(147, 161, 161)
	t.BorderStrong = rgb(101, 123, 131)
	t.ListHover = rgb(228, 222, 205)
	t.ListActive = rgb(220, 212, 195)
	t.FocusRing = rgb(38, 139, 210)
	t.Selection = rgb(181, 220, 255)
	t.ScrollTrack = rgb(238, 232, 213)
	t.ScrollThumb = rgb(147, 161, 161)
	t.ScrollThumbHover = rgb(38, 139, 210)
	t.Error = rgb(220, 50, 47)
	t.Warning = rgb(181, 137, 0)
	t.Success = rgb(133, 153, 0)
	t.Syntax = syntax(
		rgb(101, 123, 131), rgb(133, 153, 0), rgb(42, 161, 152),
		rgb(131, 148, 150), rgb(211, 54, 130), rgb(38, 139, 210))
	return finishTheme(t)
}

func gruvboxDark() Theme {
	t := baseTheme("gruvbox-dark", true)
	t.Surface = rgb(40, 40, 40)
	t.Chrome = rgb(50, 48, 47)
	t.ChromeMuted = rgb(60, 56, 54)
	t.Foreground = rgb(235, 219, 178)
	t.ForegroundMuted = rgb(168, 153, 132)
	t.ForegroundSubtle = rgb(146, 131, 116)
	t.ForegroundDisabled = rgba(168, 153, 132, 0.45)
	t.Accent = rgb(131, 165, 152)
	t.AccentHover = rgb(142, 192, 178)
	t.AccentPressed = rgb(110, 145, 132)
	t.AccentForeground = rgb(40, 40, 40)
	t.Border = rgb(80, 73, 69)
	t.BorderStrong = rgb(102, 92, 84)
	t.ListHover = rgb(60, 56, 54)
	t.ListActive = rgb(80, 73, 69)
	t.FocusRing = rgb(131, 165, 152)
	t.Selection = rgb(76, 69, 62)
	t.ScrollTrack = rgb(50, 48, 47)
	t.ScrollThumb = rgb(124, 111, 100)
	t.ScrollThumbHover = rgb(131, 165, 152)
	t.Error = rgb(251, 73, 52)
	t.Warning = rgb(250, 189, 47)
	t.Success = rgb(184, 187, 38)
	t.Syntax = syntax(
		rgb(235, 219, 178), rgb(250, 189, 47), rgb(184, 187, 38),
		rgb(146, 131, 116), rgb(211, 134, 155), rgb(131, 165, 152))
	return finishTheme(t)
}

func gruvboxLight() Theme {
	t := baseTheme("gruvbox-light", false)
	t.Surface = rgb(251, 241, 199)
	t.Chrome = rgb(242, 229, 188)
	t.ChromeMuted = rgb(235, 219, 178)
	t.Foreground = rgb(60, 56, 54)
	t.ForegroundMuted = rgb(102, 92, 84)
	t.ForegroundSubtle = rgb(146, 131, 116)
	t.ForegroundDisabled = rgba(102, 92, 84, 0.45)
	t.Accent = rgb(7, 102, 120)
	t.AccentHover = rgb(15, 120, 140)
	t.AccentPressed = rgb(5, 85, 100)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(168, 153, 132)
	t.BorderStrong = rgb(146, 131, 116)
	t.ListHover = rgb(235, 219, 178)
	t.ListActive = rgb(213, 196, 161)
	t.FocusRing = rgb(7, 102, 120)
	t.Selection = rgb(213, 196, 161)
	t.ScrollTrack = rgb(242, 229, 188)
	t.ScrollThumb = rgb(168, 153, 132)
	t.ScrollThumbHover = rgb(7, 102, 120)
	t.Error = rgb(157, 0, 6)
	t.Warning = rgb(181, 118, 20)
	t.Success = rgb(121, 116, 14)
	t.Syntax = syntax(
		rgb(60, 56, 54), rgb(204, 36, 29), rgb(104, 157, 106),
		rgb(102, 92, 84), rgb(177, 98, 134), rgb(7, 102, 120))
	return finishTheme(t)
}

func monokai() Theme {
	t := baseTheme("monokai", true)
	t.Surface = rgb(39, 40, 34)
	t.Chrome = rgb(50, 51, 44)
	t.ChromeMuted = rgb(62, 63, 53)
	t.Foreground = rgb(248, 248, 242)
	t.ForegroundMuted = rgb(155, 150, 130)
	t.ForegroundSubtle = rgb(130, 125, 105)
	t.ForegroundDisabled = rgba(155, 150, 130, 0.45)
	t.Accent = rgb(102, 217, 239)
	t.AccentHover = rgb(120, 230, 250)
	t.AccentPressed = rgb(80, 195, 215)
	t.AccentForeground = rgb(39, 40, 34)
	t.Border = rgb(75, 75, 65)
	t.BorderStrong = rgb(95, 95, 82)
	t.ListHover = rgb(62, 63, 53)
	t.ListActive = rgb(80, 80, 68)
	t.FocusRing = rgb(102, 217, 239)
	t.Selection = rgb(72, 72, 62)
	t.ScrollTrack = rgb(44, 45, 38)
	t.ScrollThumb = rgb(105, 105, 90)
	t.ScrollThumbHover = rgb(102, 217, 239)
	t.Error = rgb(255, 84, 84)
	t.Warning = rgb(230, 219, 116)
	t.Success = rgb(166, 226, 46)
	t.Syntax = syntax(
		rgb(248, 248, 242), rgb(249, 38, 114), rgb(230, 219, 116),
		rgb(153, 148, 128), rgb(174, 129, 255), rgb(166, 226, 46))
	return finishTheme(t)
}

func everforestDark() Theme {
	t := baseTheme("everforest-dark", true)
	t.Surface = rgb(45, 53, 59)     // bg
	t.Chrome = rgb(35, 42, 46)      // bg_dim
	t.ChromeMuted = rgb(52, 63, 68) // bg0
	t.Foreground = rgb(211, 198, 170)
	t.ForegroundMuted = rgb(167, 157, 137)
	t.ForegroundSubtle = rgb(127, 119, 104)
	t.ForegroundDisabled = rgba(167, 157, 137, 0.45)
	t.Accent = rgb(127, 187, 179) // blue/aqua
	t.AccentHover = rgb(142, 200, 192)
	t.AccentPressed = rgb(105, 165, 158)
	t.AccentForeground = rgb(45, 53, 59)
	t.Border = rgb(80, 88, 84)
	t.BorderStrong = rgb(100, 110, 105)
	t.ListHover = rgb(52, 63, 68)
	t.ListActive = rgb(68, 78, 74)
	t.FocusRing = rgb(127, 187, 179)
	t.Selection = rgb(68, 78, 74)
	t.ScrollTrack = rgb(35, 42, 46)
	t.ScrollThumb = rgb(100, 110, 105)
	t.ScrollThumbHover = rgb(127, 187, 179)
	t.Error = rgb(230, 126, 128)
	t.Warning = rgb(219, 188, 127)
	t.Success = rgb(167, 192, 128)
	t.Syntax = syntax(
		rgb(211, 198, 170), rgb(214, 153, 182), rgb(167, 192, 128),
		rgb(127, 119, 104), rgb(230, 152, 117), rgb(127, 187, 179))
	return finishTheme(t)
}

func everforestLight() Theme {
	t := baseTheme("everforest-light", false)
	t.Surface = rgb(255, 251, 239)    // bg
	t.Chrome = rgb(248, 245, 228)     // bg_dim
	t.ChromeMuted = rgb(239, 235, 220) // bg0
	t.Foreground = rgb(92, 106, 114)
	t.ForegroundMuted = rgb(127, 140, 148)
	t.ForegroundSubtle = rgb(160, 172, 178)
	t.ForegroundDisabled = rgba(127, 140, 148, 0.45)
	t.Accent = rgb(60, 140, 130)
	t.AccentHover = rgb(75, 155, 145)
	t.AccentPressed = rgb(45, 120, 110)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(180, 190, 185)
	t.BorderStrong = rgb(140, 155, 148)
	t.ListHover = rgb(239, 235, 220)
	t.ListActive = rgb(220, 215, 200)
	t.FocusRing = rgb(60, 140, 130)
	t.Selection = rgb(200, 225, 210)
	t.ScrollTrack = rgb(248, 245, 228)
	t.ScrollThumb = rgb(160, 172, 178)
	t.ScrollThumbHover = rgb(60, 140, 130)
	t.Error = rgb(200, 80, 80)
	t.Warning = rgb(180, 140, 60)
	t.Success = rgb(100, 150, 80)
	t.Syntax = syntax(
		rgb(92, 106, 114), rgb(180, 100, 140), rgb(100, 150, 80),
		rgb(160, 172, 178), rgb(200, 120, 80), rgb(60, 140, 130))
	return finishTheme(t)
}
