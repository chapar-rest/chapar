package theme

import (
	"image/color"

	"cogentcore.org/core/colors"
)

type schema struct {
	primary color.RGBA
	isDark  bool
}

var schemas = map[Name]schema{
	Dark:                {primary: colors.FromRGB(0x45, 0x89, 0xf5), isDark: true},
	Light:               {primary: colors.FromRGB(0x45, 0x89, 0xf5), isDark: false},
	GithubDark:          {primary: colors.FromRGB(0x1f, 0x6f, 0xeb), isDark: true},
	GithubLight:         {primary: colors.FromRGB(0x09, 0x69, 0xda), isDark: false},
	CatppuccinLatte:     {primary: colors.FromRGB(0x1e, 0x66, 0xf5), isDark: false},
	CatppuccinFrappe:    {primary: colors.FromRGB(0x8c, 0xaa, 0xee), isDark: true},
	CatppuccinMacchiato: {primary: colors.FromRGB(0x8a, 0xad, 0xf4), isDark: true},
	CatppuccinMocha:     {primary: colors.FromRGB(0x89, 0xb4, 0xfa), isDark: true},
}

func schemaFor(name Name) (schema, bool) {
	def, ok := schemas[name]
	return def, ok
}
