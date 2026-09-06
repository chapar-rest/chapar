package theme

// yogaDark is the default Yoga theme — neutral IDE chrome with a soft blue accent.
func yogaDark() Theme {
	t := Theme{
		Name: "yoga-dark", Dark: true,

		Surface:            rgb(24, 24, 29),
		Chrome:             rgb(33, 33, 40),
		ChromeMuted:        rgb(43, 43, 52),
		Foreground:         rgb(213, 217, 227),
		ForegroundMuted:    rgb(126, 131, 145),
		ForegroundSubtle:   rgb(98, 104, 118),
		ForegroundDisabled: rgba(126, 131, 145, 0.45),
		Accent:             rgb(94, 129, 232),
		AccentHover:        rgb(110, 145, 245),
		AccentPressed:      rgb(70, 100, 200),
		AccentForeground:   rgb(255, 255, 255),
		Border:             rgb(54, 56, 68),
		BorderStrong:       rgb(70, 74, 88),
		ListHover:          rgb(54, 56, 68),
		ListActive:         rgb(70, 74, 92),
		FocusRing:          rgb(94, 129, 232),

		Selection:        rgb(58, 70, 104),
		ScrollTrack:      rgb(36, 36, 44),
		ScrollThumb:      rgb(100, 104, 118),
		ScrollThumbHover: rgb(94, 129, 232),

		Error:   rgb(224, 108, 117),
		Warning: rgb(229, 192, 123),
		Success: rgb(152, 195, 121),

		Spacing:    DefaultSpacing(),
		Radius:     DefaultRadius(),
		Stroke:     DefaultStroke(),
		Typography: DefaultTypography(),
		Metrics:    DefaultComponentMetrics(),
		Elevation:  DefaultElevationDark(),

		Syntax: syntax(
			rgb(213, 217, 227), rgb(197, 134, 192), rgb(152, 195, 121),
			rgb(130, 140, 150), rgb(209, 154, 102), rgb(86, 182, 194)), // comment: was 3.67:1, now ~4.8:1
	}
	syncLegacyFromYoga(&t)
	return t
}

// yogaLight is a clean light Yoga theme for daytime work.
func yogaLight() Theme {
	t := Theme{
		Name: "yoga-light", Dark: false,

		Surface:            rgb(250, 250, 252),
		Chrome:             rgb(240, 240, 244),
		ChromeMuted:        rgb(230, 230, 236),
		Foreground:         rgb(30, 32, 38),
		ForegroundMuted:    rgb(110, 116, 128),
		ForegroundSubtle:   rgb(140, 146, 158),
		ForegroundDisabled: rgba(110, 116, 128, 0.45),
		Accent:             rgb(40, 110, 230),
		AccentHover:        rgb(55, 125, 245),
		AccentPressed:      rgb(30, 90, 200),
		AccentForeground:   rgb(255, 255, 255),
		Border:             rgb(215, 218, 225),
		BorderStrong:       rgb(190, 195, 205),
		ListHover:          rgb(225, 228, 235),
		ListActive:         rgb(210, 215, 225),
		FocusRing:          rgb(40, 110, 230),

		Selection:        rgb(180, 205, 250),
		ScrollTrack:      rgb(220, 222, 228),
		ScrollThumb:      rgb(160, 165, 175),
		ScrollThumbHover: rgb(40, 110, 230),

		Error:   rgb(200, 50, 60),
		Warning: rgb(190, 130, 20),
		Success: rgb(40, 140, 70),

		Spacing:    DefaultSpacing(),
		Radius:     DefaultRadius(),
		Stroke:     DefaultStroke(),
		Typography: DefaultTypography(),
		Metrics:    DefaultComponentMetrics(),
		Elevation:  DefaultElevationLight(),

		Syntax: syntax(
			rgb(30, 32, 38), rgb(168, 40, 150), rgb(30, 130, 60),
			rgb(100, 106, 118), rgb(180, 90, 20), rgb(20, 120, 150)), // comment: was 3.03:1, now ~5.1:1
	}
	syncLegacyFromYoga(&t)
	return t
}

// yogaHighContrast is an accessibility-oriented Yoga theme with strong contrast.
func yogaHighContrast() Theme {
	t := baseTheme("yoga-high-contrast", true)
	t.Surface = rgb(0, 0, 0)
	t.Chrome = rgb(16, 16, 16)
	t.ChromeMuted = rgb(32, 32, 32)
	t.Foreground = rgb(255, 255, 255)
	t.ForegroundMuted = rgb(200, 200, 200)
	t.ForegroundSubtle = rgb(160, 160, 160)
	t.ForegroundDisabled = rgba(160, 160, 160, 0.5)
	t.Accent = rgb(255, 213, 0)
	t.AccentHover = rgb(255, 230, 80)
	t.AccentPressed = rgb(220, 180, 0)
	t.AccentForeground = rgb(0, 0, 0)
	t.Border = rgb(200, 200, 200)
	t.BorderStrong = rgb(255, 255, 255)
	t.ListHover = rgb(48, 48, 48)
	t.ListActive = rgb(64, 64, 64)
	t.FocusRing = rgb(255, 213, 0)
	t.Selection = rgb(80, 80, 0)
	t.ScrollTrack = rgb(24, 24, 24)
	t.ScrollThumb = rgb(200, 200, 200)
	t.ScrollThumbHover = rgb(255, 213, 0)
	t.Error = rgb(255, 100, 100)
	t.Warning = rgb(255, 213, 0)
	t.Success = rgb(100, 255, 100)
	// Sharper chrome for accessibility: thicker strokes, squarer corners.
	t.Stroke = Stroke{Thin: DefaultStroke().Thick, Thick: DefaultStroke().Thicker, Thicker: 4}
	t.Radius = Radius{None: 0, Small: 0, Medium: 2, Large: 4, XLarge: 6, Circular: 9999}
	t.Syntax = syntax(
		rgb(255, 255, 255), rgb(255, 180, 180), rgb(180, 255, 180),
		rgb(180, 180, 180), rgb(255, 220, 140), rgb(140, 220, 255))
	return finishTheme(t)
}

// yogaMidnight is a cool blue-slate dark theme: deep slate surfaces, a vivid
// blue accent, and an emerald success green. Matches the "v1" mockup palette.
//
// Note: the app's root background reads th.Background and editors read
// th.Surface; normalize() keeps Background == Surface, so they render as one
// color. The visual tiering comes from Chrome (toolbar/tab strips) sitting one
// step above Surface, plus the 1px Border around editor panels.
func yogaMidnight() Theme {
	t := baseTheme("yoga-midnight", true)
	t.Surface = rgb(21, 26, 33)    // #151A21 editor / workspace
	t.Chrome = rgb(28, 35, 44)   // #1C232C toolbars, tab bars, controls
	t.ChromeMuted = rgb(14, 18, 24) // darker gutters/tracks — distinct from Chrome
	t.Foreground = rgb(232, 236, 241)
	t.ForegroundMuted = rgb(154, 164, 178)
	t.ForegroundSubtle = rgb(90, 101, 115)
	t.ForegroundDisabled = rgba(154, 164, 178, 0.40)
	t.Accent = rgb(47, 111, 237)
	t.AccentHover = rgb(59, 125, 240)
	t.AccentPressed = rgb(37, 96, 216)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(42, 50, 61)
	t.BorderStrong = rgb(58, 68, 82)
	t.ListHover = rgb(34, 41, 51)
	t.ListActive = rgb(42, 50, 61)
	t.FocusRing = rgb(47, 111, 237)
	t.Selection = rgb(40, 64, 110)
	t.ScrollTrack = rgb(14, 18, 24)
	t.ScrollThumb = rgb(58, 68, 82)
	t.ScrollThumbHover = rgb(47, 111, 237)
	t.Error = rgb(237, 106, 94)
	t.Warning = rgb(224, 165, 59)
	t.Success = rgb(79, 184, 112)
	// Midnight-specific syntax (cobalt keys, slate comments, emerald strings).
	t.Syntax = syntax(
		rgb(220, 224, 232), // default
		rgb(100, 149, 237), // keyword — cornflower blue
		rgb(95, 191, 135),  // string — emerald
		rgb(108, 118, 138), // comment — slate gray
		rgb(240, 180, 100), // number — amber
		rgb(130, 170, 255), // type — cobalt
	)
	return finishTheme(t)
}
