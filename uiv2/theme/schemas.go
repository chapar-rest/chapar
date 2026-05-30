package theme

import (
	"image/color"
)

type schema struct {
	primary color.RGBA
	isDark  bool

	// Explicit palette overrides. When surface is non-zero, these replace
	// the MD3-generated surface/text/outline/accent values after setup.
	surface                 color.RGBA
	surfaceContainerLowest  color.RGBA
	surfaceContainerLow     color.RGBA
	surfaceContainer        color.RGBA
	surfaceContainerHigh    color.RGBA
	surfaceContainerHighest color.RGBA
	surfaceVariant          color.RGBA
	onSurface               color.RGBA
	onSurfaceVariant        color.RGBA
	outline                 color.RGBA
	outlineVariant          color.RGBA
	onPrimary               color.RGBA
	secondary               color.RGBA
	onSecondary             color.RGBA
	secondaryContainer      color.RGBA
	onSecondaryContainer    color.RGBA
}

func (s schema) hasExplicitPalette() bool {
	return s.surface.A != 0
}

func schemeRGB(c uint32) color.RGBA {
	n := rgb(c)
	return color.RGBA{R: n.R, G: n.G, B: n.B, A: n.A}
}

var schemas = map[Name]schema{
	Dark:  {primary: schemeRGB(0x4589f5), isDark: true},
	Light: {primary: schemeRGB(0x4589f5), isDark: false},

	GithubDark: {
		primary:                 schemeRGB(0x1f6feb),
		isDark:                  true,
		surface:                 schemeRGB(0x0d1117),
		surfaceContainerLowest:  schemeRGB(0x010409),
		surfaceContainerLow:     schemeRGB(0x161b22),
		surfaceContainer:        schemeRGB(0x0d1117),
		surfaceContainerHigh:    schemeRGB(0x21262d),
		surfaceContainerHighest: schemeRGB(0x30363d),
		surfaceVariant:          schemeRGB(0x21262d),
		onSurface:               schemeRGB(0xe6edf3),
		onSurfaceVariant:        schemeRGB(0x7d8590),
		outline:                 schemeRGB(0x30363d),
		outlineVariant:          schemeRGB(0x21262d),
		onPrimary:               schemeRGB(0xffffff),
		secondary:               schemeRGB(0x8957e5),
		onSecondary:             schemeRGB(0xffffff),
		secondaryContainer:      schemeRGB(0x21262d),
		onSecondaryContainer:    schemeRGB(0xe6edf3),
	},
	GithubLight: {
		primary:                 schemeRGB(0x0969da),
		isDark:                  false,
		surface:                 schemeRGB(0xffffff),
		surfaceContainerLowest:  schemeRGB(0xeaeef2),
		surfaceContainerLow:     schemeRGB(0xf6f8fa),
		surfaceContainer:        schemeRGB(0xffffff),
		surfaceContainerHigh:    schemeRGB(0xeaeef2),
		surfaceContainerHighest: schemeRGB(0xd0d7de),
		surfaceVariant:          schemeRGB(0xd8dee4),
		onSurface:               schemeRGB(0x1f2328),
		onSurfaceVariant:        schemeRGB(0x656d76),
		outline:                 schemeRGB(0xd0d7de),
		outlineVariant:          schemeRGB(0xd8dee4),
		onPrimary:               schemeRGB(0xffffff),
		secondary:               schemeRGB(0x8250df),
		onSecondary:             schemeRGB(0xffffff),
		secondaryContainer:      schemeRGB(0xeaeef2),
		onSecondaryContainer:    schemeRGB(0x1f2328),
	},

	CatppuccinLatte: {
		primary:                 schemeRGB(0x1e66f5),
		isDark:                  false,
		surface:                 schemeRGB(0xeff1f5),
		surfaceContainerLowest:  schemeRGB(0xdce0e8),
		surfaceContainerLow:     schemeRGB(0xe6e9ef),
		surfaceContainer:        schemeRGB(0xeff1f5),
		surfaceContainerHigh:    schemeRGB(0xccd0da),
		surfaceContainerHighest: schemeRGB(0xbcc0cc),
		surfaceVariant:          schemeRGB(0xacb0be),
		onSurface:               schemeRGB(0x4c4f69),
		onSurfaceVariant:        schemeRGB(0x6c6f85),
		outline:                 schemeRGB(0x8c8fa1),
		outlineVariant:          schemeRGB(0x9ca0b0),
		onPrimary:               schemeRGB(0xffffff),
		secondary:               schemeRGB(0x8839ef),
		onSecondary:             schemeRGB(0xffffff),
		secondaryContainer:      schemeRGB(0xccd0da),
		onSecondaryContainer:    schemeRGB(0x4c4f69),
	},
	CatppuccinFrappe: {
		primary:                 schemeRGB(0x8caaee),
		isDark:                  true,
		surface:                 schemeRGB(0x303446),
		surfaceContainerLowest:  schemeRGB(0x232634),
		surfaceContainerLow:     schemeRGB(0x292c3c),
		surfaceContainer:        schemeRGB(0x303446),
		surfaceContainerHigh:    schemeRGB(0x414559),
		surfaceContainerHighest: schemeRGB(0x51576d),
		surfaceVariant:          schemeRGB(0x626880),
		onSurface:               schemeRGB(0xc6d0f5),
		onSurfaceVariant:        schemeRGB(0xa5adce),
		outline:                 schemeRGB(0x838ba7),
		outlineVariant:          schemeRGB(0x737994),
		onPrimary:               schemeRGB(0x232634),
		secondary:               schemeRGB(0xca9ee6),
		onSecondary:             schemeRGB(0x232634),
		secondaryContainer:      schemeRGB(0x414559),
		onSecondaryContainer:    schemeRGB(0xc6d0f5),
	},
	CatppuccinMacchiato: {
		primary:                 schemeRGB(0x8aadf4),
		isDark:                  true,
		surface:                 schemeRGB(0x24273a),
		surfaceContainerLowest:  schemeRGB(0x181926),
		surfaceContainerLow:     schemeRGB(0x1e2030),
		surfaceContainer:        schemeRGB(0x24273a),
		surfaceContainerHigh:    schemeRGB(0x363a4f),
		surfaceContainerHighest: schemeRGB(0x494d64),
		surfaceVariant:          schemeRGB(0x5b6078),
		onSurface:               schemeRGB(0xcad3f5),
		onSurfaceVariant:        schemeRGB(0xa5adcb),
		outline:                 schemeRGB(0x8087a2),
		outlineVariant:          schemeRGB(0x6e738d),
		onPrimary:               schemeRGB(0x181926),
		secondary:               schemeRGB(0xc6a0f6),
		onSecondary:             schemeRGB(0x181926),
		secondaryContainer:      schemeRGB(0x363a4f),
		onSecondaryContainer:    schemeRGB(0xcad3f5),
	},
	CatppuccinMocha: {
		primary:                 schemeRGB(0x89b4fa),
		isDark:                  true,
		surface:                 schemeRGB(0x1e1e2e),
		surfaceContainerLowest:  schemeRGB(0x11111b),
		surfaceContainerLow:     schemeRGB(0x181825),
		surfaceContainer:        schemeRGB(0x1e1e2e),
		surfaceContainerHigh:    schemeRGB(0x313244),
		surfaceContainerHighest: schemeRGB(0x45475a),
		surfaceVariant:          schemeRGB(0x585b70),
		onSurface:               schemeRGB(0xcdd6f4),
		onSurfaceVariant:        schemeRGB(0xa6adc8),
		outline:                 schemeRGB(0x7f849c),
		outlineVariant:          schemeRGB(0x6c7086),
		onPrimary:               schemeRGB(0x11111b),
		secondary:               schemeRGB(0xcba6f7),
		onSecondary:             schemeRGB(0x11111b),
		secondaryContainer:      schemeRGB(0x313244),
		onSecondaryContainer:    schemeRGB(0xcdd6f4),
	},

	RosePine: {
		primary:                 schemeRGB(0xc4a7e7),
		isDark:                  true,
		surface:                 schemeRGB(0x191724),
		surfaceContainerLowest:  schemeRGB(0x191724),
		surfaceContainerLow:     schemeRGB(0x1f1d2e),
		surfaceContainer:        schemeRGB(0x191724),
		surfaceContainerHigh:    schemeRGB(0x26233a),
		surfaceContainerHighest: schemeRGB(0x403d52),
		surfaceVariant:          schemeRGB(0x524f67),
		onSurface:               schemeRGB(0xe0def4),
		onSurfaceVariant:        schemeRGB(0x908caa),
		outline:                 schemeRGB(0x524f67),
		outlineVariant:          schemeRGB(0x26233a),
		onPrimary:               schemeRGB(0x191724),
		secondary:               schemeRGB(0xebbcba),
		onSecondary:             schemeRGB(0x191724),
		secondaryContainer:      schemeRGB(0x26233a),
		onSecondaryContainer:    schemeRGB(0xe0def4),
	},
	RosePineMoon: {
		primary:                 schemeRGB(0xc4a7e7),
		isDark:                  true,
		surface:                 schemeRGB(0x232136),
		surfaceContainerLowest:  schemeRGB(0x232136),
		surfaceContainerLow:     schemeRGB(0x2a273f),
		surfaceContainer:        schemeRGB(0x232136),
		surfaceContainerHigh:    schemeRGB(0x393552),
		surfaceContainerHighest: schemeRGB(0x44415a),
		surfaceVariant:          schemeRGB(0x56526e),
		onSurface:               schemeRGB(0xe0def4),
		onSurfaceVariant:        schemeRGB(0x908caa),
		outline:                 schemeRGB(0x56526e),
		outlineVariant:          schemeRGB(0x393552),
		onPrimary:               schemeRGB(0x232136),
		secondary:               schemeRGB(0xebbcba),
		onSecondary:             schemeRGB(0x232136),
		secondaryContainer:      schemeRGB(0x393552),
		onSecondaryContainer:    schemeRGB(0xe0def4),
	},
	RosePineDawn: {
		primary:                 schemeRGB(0x907aa9),
		isDark:                  false,
		surface:                 schemeRGB(0xfaf4ed),
		surfaceContainerLowest:  schemeRGB(0xf9f5d7),
		surfaceContainerLow:     schemeRGB(0xfffaf3),
		surfaceContainer:        schemeRGB(0xfaf4ed),
		surfaceContainerHigh:    schemeRGB(0xf4ede8),
		surfaceContainerHighest: schemeRGB(0xf2d9c3),
		surfaceVariant:          schemeRGB(0xcecacd),
		onSurface:               schemeRGB(0x575279),
		onSurfaceVariant:        schemeRGB(0x797593),
		outline:                 schemeRGB(0xcecacd),
		outlineVariant:          schemeRGB(0xf4ede8),
		onPrimary:               schemeRGB(0xfaf4ed),
		secondary:               schemeRGB(0xd7827e),
		onSecondary:             schemeRGB(0xfaf4ed),
		secondaryContainer:      schemeRGB(0xf4ede8),
		onSecondaryContainer:    schemeRGB(0x575279),
	},
}

func schemaFor(name Name) (schema, bool) {
	def, ok := schemas[name]
	return def, ok
}
