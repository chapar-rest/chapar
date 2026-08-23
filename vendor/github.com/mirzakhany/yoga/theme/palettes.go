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

// builtins returns all themes shipped with the framework.
func builtins() []Theme {
	return []Theme{
		yogaDark(),
		yogaLight(),
		yogaHighContrast(),
		dark(),
		light(),
		githubDark(),
		githubLight(),
		catppuccin(),
		dracula(),
		nord(),
		solarizedDark(),
		gruvboxDark(),
		gruvboxLight(),
		oneDark(),
		monokai(),
		tokyoNight(),
		rosePine(),
		yogaMidnight(),
	}
}

func dark() Theme {
	return Theme{
		Name: "dark", Dark: true,
		Background: rgb(24, 24, 29),
		Panel:      rgb(33, 33, 40),
		PanelAlt:   rgb(43, 43, 52),
		Text:       rgb(213, 217, 227),
		TextDim:    rgb(126, 131, 145),
		Accent:     rgb(94, 129, 232),
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(54, 56, 68),
		Active:     rgb(70, 74, 92),
		Border:           rgb(54, 56, 68),
		ScrollTrack:      rgb(36, 36, 44),
		ScrollThumb:      rgb(100, 104, 118),
		ScrollThumbHover: rgb(94, 129, 232),
		Selection:        rgb(58, 70, 104),
		Error:      rgb(224, 108, 117),
		Warning:    rgb(229, 192, 123),
		Success:    rgb(152, 195, 121),
		Syntax: syntax(
			rgb(213, 217, 227), rgb(197, 134, 192), rgb(152, 195, 121),
			rgb(130, 140, 150), rgb(209, 154, 102), rgb(86, 182, 194)),
	}
}

func light() Theme {
	return Theme{
		Name: "light", Dark: false,
		Background: rgb(250, 250, 250),
		Panel:      rgb(240, 240, 242),
		PanelAlt:   rgb(230, 230, 233),
		Text:       rgb(30, 32, 38),
		TextDim:    rgb(110, 116, 128),
		Accent:     rgb(40, 110, 230),
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(225, 228, 235),
		Active:     rgb(210, 215, 225),
		Border:           rgb(215, 218, 225),
		ScrollTrack:      rgb(220, 222, 228),
		ScrollThumb:      rgb(160, 165, 175),
		ScrollThumbHover: rgb(40, 110, 230),
		Selection:        rgb(180, 205, 250),
		Error:      rgb(200, 50, 60),
		Warning:    rgb(190, 130, 20),
		Success:    rgb(40, 140, 70),
		Syntax: syntax(
			rgb(30, 32, 38), rgb(168, 40, 150), rgb(30, 130, 60),
			rgb(100, 106, 118), rgb(180, 90, 20), rgb(20, 120, 150)),
	}
}

func githubDark() Theme {
	return Theme{
		Name: "github-dark", Dark: true,
		Background: rgb(13, 17, 23),
		Panel:      rgb(22, 27, 34),
		PanelAlt:   rgb(33, 38, 45),
		Text:       rgb(230, 237, 243),
		TextDim:    rgb(125, 133, 144),
		Accent:     rgb(47, 129, 247),
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(33, 38, 45),
		Active:     rgb(48, 54, 61),
		Border:           rgb(48, 54, 61),
		ScrollTrack:      rgb(28, 33, 40),
		ScrollThumb:      rgb(90, 98, 110),
		ScrollThumbHover: rgb(47, 129, 247),
		Selection:        rgb(56, 90, 140),
		Error:      rgb(248, 81, 73),
		Warning:    rgb(210, 153, 34),
		Success:    rgb(63, 185, 80),
		Syntax: syntax(
			rgb(230, 237, 243), rgb(255, 123, 114), rgb(165, 214, 255),
			rgb(139, 148, 158), rgb(121, 192, 255), rgb(255, 166, 87)),
	}
}

func githubLight() Theme {
	return Theme{
		Name: "github-light", Dark: false,
		Background: rgb(255, 255, 255),
		Panel:      rgb(246, 248, 250),
		PanelAlt:   rgb(234, 238, 242),
		Text:       rgb(31, 35, 40),
		TextDim:    rgb(101, 109, 118),
		Accent:     rgb(9, 105, 218),
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(234, 238, 242),
		Active:     rgb(215, 222, 228),
		Border:           rgb(208, 215, 222),
		ScrollTrack:      rgb(225, 228, 233),
		ScrollThumb:      rgb(155, 162, 172),
		ScrollThumbHover: rgb(9, 105, 218),
		Selection:        rgb(184, 215, 255),
		Error:      rgb(207, 34, 46),
		Warning:    rgb(154, 103, 0),
		Success:    rgb(26, 127, 55),
		Syntax: syntax(
			rgb(31, 35, 40), rgb(207, 34, 46), rgb(10, 48, 105),
			rgb(110, 119, 129), rgb(5, 80, 174), rgb(149, 56, 0)),
	}
}

func catppuccin() Theme {
	return Theme{
		Name: "catppuccin", Dark: true,
		Background: rgb(30, 30, 46),
		Panel:      rgb(24, 24, 37),
		PanelAlt:   rgb(49, 50, 68),
		Text:       rgb(205, 214, 244),
		TextDim:    rgb(166, 173, 200),
		Accent:     rgb(137, 180, 250),
		AccentText: rgb(30, 30, 46),
		Hover:      rgb(49, 50, 68),
		Active:     rgb(69, 71, 90),
		Border:           rgb(69, 71, 90),
		ScrollTrack:      rgb(40, 41, 56),
		ScrollThumb:      rgb(120, 125, 148),
		ScrollThumbHover: rgb(137, 180, 250),
		Selection:        rgb(88, 91, 112),
		Error:      rgb(243, 139, 168),
		Warning:    rgb(249, 226, 175),
		Success:    rgb(166, 227, 161),
		Syntax: syntax(
			rgb(205, 214, 244), rgb(203, 166, 247), rgb(166, 227, 161),
			rgb(140, 145, 170), rgb(250, 179, 135), rgb(249, 226, 175)),
	}
}

func dracula() Theme {
	return Theme{
		Name: "dracula", Dark: true,
		Background: rgb(40, 42, 54),
		Panel:      rgb(33, 34, 44),
		PanelAlt:   rgb(68, 71, 90),
		Text:       rgb(248, 248, 242),
		TextDim:    rgb(98, 114, 164),
		Accent:     rgb(189, 147, 249),
		AccentText: rgb(40, 42, 54),
		Hover:      rgb(68, 71, 90),
		Active:     rgb(88, 91, 112),
		Border:           rgb(68, 71, 90),
		ScrollTrack:      rgb(50, 52, 68),
		ScrollThumb:      rgb(120, 125, 148),
		ScrollThumbHover: rgb(189, 147, 249),
		Selection:        rgb(68, 71, 90),
		Error:            rgb(255, 85, 85),
		Warning:          rgb(241, 250, 140),
		Success:          rgb(80, 250, 123),
		Syntax: syntax(
			rgb(248, 248, 242), rgb(255, 121, 198), rgb(241, 250, 140),
			rgb(145, 158, 200), rgb(189, 147, 249), rgb(139, 233, 253)),
	}
}

func nord() Theme {
	return Theme{
		Name: "nord", Dark: true,
		Background: rgb(46, 52, 64),
		Panel:      rgb(59, 66, 82),
		PanelAlt:   rgb(67, 76, 94),
		Text:       rgb(216, 222, 233),
		TextDim:    rgb(123, 136, 161),
		Accent:     rgb(136, 192, 208),
		AccentText: rgb(46, 52, 64),
		Hover:      rgb(67, 76, 94),
		Active:     rgb(76, 86, 106),
		Border:           rgb(76, 86, 106),
		ScrollTrack:      rgb(55, 62, 76),
		ScrollThumb:      rgb(110, 122, 145),
		ScrollThumbHover: rgb(136, 192, 208),
		Selection:        rgb(76, 86, 106),
		Error:            rgb(191, 97, 106),
		Warning:    rgb(235, 203, 139),
		Success:    rgb(163, 190, 140),
		Syntax: syntax(
			rgb(216, 222, 233), rgb(129, 161, 193), rgb(163, 190, 140),
			rgb(155, 170, 198), rgb(180, 142, 173), rgb(143, 188, 187)),
	}
}

func solarizedDark() Theme {
	return Theme{
		Name: "solarized-dark", Dark: true,
		Background: rgb(0, 43, 54),
		Panel:      rgb(7, 54, 66),
		PanelAlt:   rgb(14, 63, 76),
		Text:       rgb(131, 148, 150),
		TextDim:    rgb(120, 148, 155), // brightened: was 2.79:1, now ~4.9:1
		Accent:     rgb(38, 139, 210),
		AccentText: rgb(0, 26, 34),
		Hover:      rgb(18, 68, 84),   // distinct from Panel (was identical)
		Active:     rgb(28, 80, 98),   // distinct from Hover
		Border:           rgb(72, 98, 108),
		ScrollTrack:      rgb(5, 48, 58),
		ScrollThumb:      rgb(100, 120, 128),
		ScrollThumbHover: rgb(38, 139, 210),
		Selection:        rgb(0, 80, 130), // blue-tinted, distinct from Panel
		Error:      rgb(220, 50, 47),
		Warning:    rgb(181, 137, 0),
		Success:    rgb(133, 153, 0),
		Syntax: syntax(
			rgb(131, 148, 150), rgb(133, 153, 0), rgb(42, 161, 152),
			rgb(110, 135, 143), rgb(211, 54, 130), rgb(181, 137, 0)), // comment: was 2.79:1, now ~3.9:1
	}
}

// gruvboxDark is the warm retro Gruvbox dark palette.
func gruvboxDark() Theme {
	return Theme{
		Name: "gruvbox-dark", Dark: true,
		Background: rgb(40, 40, 40),
		Panel:      rgb(50, 48, 47),
		PanelAlt:   rgb(60, 56, 54),
		Text:       rgb(235, 219, 178),
		TextDim:    rgb(168, 153, 132),
		Accent:     rgb(131, 165, 152), // gruvbox bright blue-aqua
		AccentText: rgb(40, 40, 40),
		Hover:      rgb(60, 56, 54),
		Active:     rgb(80, 73, 69),
		Border:           rgb(80, 73, 69),
		ScrollTrack:      rgb(50, 48, 47),
		ScrollThumb:      rgb(124, 111, 100),
		ScrollThumbHover: rgb(131, 165, 152),
		Selection:        rgb(76, 69, 62),
		Error:   rgb(251, 73, 52),
		Warning: rgb(250, 189, 47),
		Success: rgb(184, 187, 38),
		Syntax: syntax(
			rgb(235, 219, 178), rgb(250, 189, 47), rgb(184, 187, 38),
			rgb(146, 131, 116), rgb(211, 134, 155), rgb(131, 165, 152)),
	}
}

// gruvboxLight is the warm retro Gruvbox light palette.
func gruvboxLight() Theme {
	return Theme{
		Name: "gruvbox-light", Dark: false,
		Background: rgb(251, 241, 199),
		Panel:      rgb(242, 229, 188),
		PanelAlt:   rgb(235, 219, 178),
		Text:       rgb(60, 56, 54),
		TextDim:    rgb(102, 92, 84),
		Accent:     rgb(7, 102, 120), // gruvbox bright blue (dark variant for readability)
		AccentText: rgb(255, 255, 255),
		Hover:      rgb(235, 219, 178),
		Active:     rgb(213, 196, 161),
		Border:           rgb(168, 153, 132),
		ScrollTrack:      rgb(242, 229, 188),
		ScrollThumb:      rgb(168, 153, 132),
		ScrollThumbHover: rgb(7, 102, 120),
		Selection:        rgb(213, 196, 161),
		Error:   rgb(157, 0, 6),
		Warning: rgb(181, 118, 20),
		Success: rgb(121, 116, 14),
		Syntax: syntax(
			rgb(60, 56, 54), rgb(204, 36, 29), rgb(104, 157, 106),
			rgb(102, 92, 84), rgb(177, 98, 134), rgb(7, 102, 120)),
	}
}

// oneDark is the Atom-inspired One Dark palette popular in VS Code.
func oneDark() Theme {
	return Theme{
		Name: "one-dark", Dark: true,
		Background: rgb(40, 44, 52),
		Panel:      rgb(33, 37, 43),
		PanelAlt:   rgb(48, 52, 64),
		Text:       rgb(171, 178, 191),
		TextDim:    rgb(150, 158, 180),
		Accent:     rgb(97, 175, 239),
		AccentText: rgb(40, 44, 52), // dark bg: bright blue needs dark text for contrast
		Hover:      rgb(48, 52, 64),
		Active:     rgb(62, 68, 82),
		Border:           rgb(60, 66, 80),
		ScrollTrack:      rgb(36, 40, 50),
		ScrollThumb:      rgb(88, 95, 112),
		ScrollThumbHover: rgb(97, 175, 239),
		Selection:        rgb(61, 73, 101),
		Error:   rgb(224, 108, 117),
		Warning: rgb(229, 192, 123),
		Success: rgb(152, 195, 121),
		Syntax: syntax(
			rgb(171, 178, 191), rgb(198, 120, 221), rgb(152, 195, 121),
			rgb(148, 158, 178), rgb(209, 154, 102), rgb(97, 175, 239)),
	}
}

// monokai is the classic Monokai palette from Sublime Text.
func monokai() Theme {
	return Theme{
		Name: "monokai", Dark: true,
		Background: rgb(39, 40, 34),
		Panel:      rgb(50, 51, 44),
		PanelAlt:   rgb(62, 63, 53),
		Text:       rgb(248, 248, 242),
		TextDim:    rgb(155, 150, 130),
		Accent:     rgb(102, 217, 239), // monokai cyan
		AccentText: rgb(39, 40, 34),
		Hover:      rgb(62, 63, 53),
		Active:     rgb(80, 80, 68),
		Border:           rgb(75, 75, 65),
		ScrollTrack:      rgb(44, 45, 38),
		ScrollThumb:      rgb(105, 105, 90),
		ScrollThumbHover: rgb(102, 217, 239),
		Selection:        rgb(72, 72, 62),
		Error:   rgb(255, 84, 84),
		Warning: rgb(230, 219, 116),
		Success: rgb(166, 226, 46),
		Syntax: syntax(
			rgb(248, 248, 242), rgb(249, 38, 114), rgb(230, 219, 116),
			rgb(153, 148, 128), rgb(174, 129, 255), rgb(166, 226, 46)),
	}
}

// tokyoNight is the modern cool-blue Tokyo Night palette.
func tokyoNight() Theme {
	return Theme{
		Name: "tokyo-night", Dark: true,
		Background: rgb(26, 27, 38),
		Panel:      rgb(36, 40, 59),
		PanelAlt:   rgb(41, 46, 66),
		Text:       rgb(169, 177, 214),
		TextDim:    rgb(124, 134, 170),
		Accent:     rgb(122, 162, 247),
		AccentText: rgb(26, 27, 38), // dark bg: bright blue needs dark text for contrast
		Hover:      rgb(41, 46, 66),
		Active:     rgb(52, 58, 82),
		Border:           rgb(52, 58, 82),
		ScrollTrack:      rgb(30, 32, 48),
		ScrollThumb:      rgb(82, 90, 126),
		ScrollThumbHover: rgb(122, 162, 247),
		Selection:        rgb(56, 71, 120),
		Error:   rgb(247, 118, 142),
		Warning: rgb(224, 175, 104),
		Success: rgb(158, 206, 106),
		Syntax: syntax(
			rgb(169, 177, 214), rgb(187, 154, 247), rgb(158, 206, 106),
			rgb(130, 140, 175), rgb(255, 158, 100), rgb(122, 162, 247)),
	}
}

// rosePine is the elegant dark Rose Piné (Moon) palette.
func rosePine() Theme {
	return Theme{
		Name: "rose-pine", Dark: true,
		Background: rgb(25, 23, 36),
		Panel:      rgb(31, 29, 46),
		PanelAlt:   rgb(38, 35, 58),
		Text:       rgb(224, 222, 244),
		TextDim:    rgb(144, 140, 170),
		Accent:     rgb(196, 167, 231), // iris
		AccentText: rgb(25, 23, 36),
		Hover:      rgb(38, 35, 58),
		Active:     rgb(52, 49, 76),
		Border:           rgb(52, 49, 76),
		ScrollTrack:      rgb(28, 26, 42),
		ScrollThumb:      rgb(108, 103, 135),
		ScrollThumbHover: rgb(196, 167, 231),
		Selection:        rgb(64, 61, 90),
		Error:   rgb(235, 111, 146),
		Warning: rgb(246, 193, 119),
		Success: rgb(156, 207, 216),
		Syntax: syntax(
			rgb(224, 222, 244), rgb(235, 111, 146), rgb(156, 207, 216),
			rgb(138, 133, 175), rgb(246, 193, 119), rgb(196, 167, 231)),
	}
}
