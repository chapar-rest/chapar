package theme

// Widely used editor palettes, adapted to the Yoga token set. Hues follow each
// upstream palette; values that could not meet the contrast targets at their
// published lightness were moved along their own hue until they did.

// tokyoNight is the storm-blue dark palette, the most widely adopted editor
// theme of recent years.
func tokyoNight() Theme {
	t := baseTheme("tokyo-night", true)
	t.Surface = rgb(26, 27, 38)     // #1a1b26
	t.Chrome = rgb(22, 22, 30)      // #16161e
	t.ChromeMuted = rgb(36, 40, 59) // #24283b
	t.Foreground = rgb(192, 202, 245)
	t.ForegroundMuted = rgb(150, 160, 200)
	t.ForegroundSubtle = rgb(120, 130, 170)
	t.ForegroundDisabled = rgba(150, 160, 200, 0.50)
	t.Accent = rgb(122, 162, 247) // #7aa2f7
	t.AccentHover = rgb(138, 173, 248)
	t.AccentPressed = rgb(154, 184, 249)
	t.AccentForeground = rgb(22, 22, 30)
	t.Border = rgb(50, 54, 76)
	t.BorderStrong = rgb(72, 78, 105)
	t.ListHover = rgb(36, 40, 59)
	t.ListActive = rgb(48, 54, 78)
	t.Selection = rgb(40, 52, 87) // #283457
	t.ScrollTrack = rgb(22, 22, 30)
	t.ScrollThumb = rgb(89, 98, 139) // #565f89
	t.ScrollThumbHover = rgb(122, 162, 247)
	t.Error = rgb(247, 118, 142)   // #f7768e
	t.Warning = rgb(224, 175, 104) // #e0af68
	t.Success = rgb(158, 206, 106) // #9ece6a
	t.Syntax = syntax(
		rgb(192, 202, 245), rgb(187, 154, 247), rgb(158, 206, 106),
		rgb(130, 140, 180), rgb(255, 158, 100), rgb(125, 207, 255))
	return finishTheme(t)
}

// tokyoNightDay is the light half of the Tokyo Night family.
func tokyoNightDay() Theme {
	t := baseTheme("tokyo-night-day", false)
	t.Surface = rgb(225, 226, 231) // #e1e2e7
	t.Chrome = rgb(214, 216, 223)  // slightly deeper chrome
	t.ChromeMuted = rgb(202, 205, 214)
	t.Foreground = rgb(43, 61, 110)
	t.ForegroundMuted = rgb(75, 85, 126)
	t.ForegroundSubtle = rgb(110, 120, 160)
	t.ForegroundDisabled = rgba(84, 94, 140, 0.55)
	t.Accent = rgb(43, 116, 217) // #2e7de9
	t.AccentHover = rgb(38, 102, 191)
	t.AccentPressed = rgb(33, 88, 165)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(188, 192, 203)
	t.BorderStrong = rgb(150, 156, 172)
	t.ListHover = rgb(198, 201, 210)
	t.ListActive = rgb(184, 189, 201)
	t.Selection = rgb(178, 198, 238)
	t.ScrollTrack = rgb(210, 213, 221)
	t.ScrollThumb = rgb(112, 119, 141)
	t.ScrollThumbHover = rgb(46, 125, 233)
	t.Error = rgb(196, 20, 74)
	t.Warning = rgb(150, 76, 0)
	t.Success = rgb(78, 106, 48)
	t.Syntax = syntax(
		rgb(43, 61, 110), rgb(132, 62, 200), rgb(78, 106, 48),
		rgb(92, 99, 132), rgb(152, 78, 0), rgb(24, 106, 157))
	return finishTheme(t)
}

// oneDark is the Atom-derived palette the default yoga-dark syntax map was
// already borrowing from; this is the full theme.
func oneDark() Theme {
	t := baseTheme("one-dark", true)
	t.Surface = rgb(40, 44, 52) // #282c34
	t.Chrome = rgb(33, 37, 43)
	t.ChromeMuted = rgb(50, 56, 66)
	t.Foreground = rgb(171, 178, 191) // #abb2bf
	t.ForegroundMuted = rgb(155, 161, 174)
	t.ForegroundSubtle = rgb(120, 128, 142)
	t.ForegroundDisabled = rgba(145, 152, 166, 0.50)
	t.Accent = rgb(97, 175, 239) // #61afef
	t.AccentHover = rgb(116, 185, 241)
	t.AccentPressed = rgb(135, 194, 243)
	t.AccentForeground = rgb(28, 31, 37)
	t.Border = rgb(62, 68, 81) // #3e4451
	t.BorderStrong = rgb(88, 96, 112)
	t.ListHover = rgb(50, 56, 66)
	t.ListActive = rgb(62, 68, 81)
	t.Selection = rgb(56, 65, 82)
	t.ScrollTrack = rgb(33, 37, 43)
	t.ScrollThumb = rgb(105, 113, 128)
	t.ScrollThumbHover = rgb(97, 175, 239)
	t.Error = rgb(224, 108, 117)   // #e06c75
	t.Warning = rgb(229, 192, 123) // #e5c07b
	t.Success = rgb(152, 195, 121) // #98c379
	t.Syntax = syntax(
		rgb(171, 178, 191), rgb(198, 120, 221), rgb(152, 195, 121),
		rgb(138, 147, 161), rgb(209, 154, 102), rgb(86, 182, 194))
	return finishTheme(t)
}

// oneLight is the light half of the One family.
func oneLight() Theme {
	t := baseTheme("one-light", false)
	t.Surface = rgb(250, 250, 250) // #fafafa
	t.Chrome = rgb(240, 240, 241)
	t.ChromeMuted = rgb(228, 228, 230)
	t.Foreground = rgb(56, 58, 66) // #383a42
	t.ForegroundMuted = rgb(99, 102, 111)
	t.ForegroundSubtle = rgb(130, 133, 142)
	t.ForegroundDisabled = rgba(100, 103, 112, 0.55)
	t.Accent = rgb(60, 113, 227) // #4078f2
	t.AccentHover = rgb(53, 99, 200)
	t.AccentPressed = rgb(46, 86, 173)
	t.AccentForeground = rgb(255, 255, 255)
	t.Border = rgb(214, 214, 217)
	t.BorderStrong = rgb(160, 161, 167)
	t.ListHover = rgb(221, 221, 224)
	t.ListActive = rgb(208, 208, 212)
	t.Selection = rgb(190, 210, 250)
	t.ScrollTrack = rgb(228, 228, 230)
	t.ScrollThumb = rgb(127, 129, 136)
	t.ScrollThumbHover = rgb(64, 120, 242)
	t.Error = rgb(202, 50, 38)   // #e45649, darkened for text
	t.Warning = rgb(150, 100, 0) // #c18401, darkened for text
	t.Success = rgb(56, 118, 55) // #50a14f, darkened for text
	t.Syntax = syntax(
		rgb(56, 58, 66), rgb(166, 38, 164), rgb(56, 118, 55),
		rgb(113, 115, 123), rgb(150, 100, 0), rgb(1, 116, 160))
	return finishTheme(t)
}
