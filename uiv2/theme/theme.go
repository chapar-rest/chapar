package theme

import (
	"image/color"

	"cogentcore.org/core/colors"
)

// Name identifies a color scheme.
type Name string

const (
	Dark                Name = "dark"
	Light               Name = "light"
	GithubDark          Name = "github-dark"
	GithubLight         Name = "github-light"
	CatppuccinLatte     Name = "catppuccin-latte"
	CatppuccinFrappe    Name = "catppuccin-frappe"
	CatppuccinMacchiato Name = "catppuccin-macchiato"
	CatppuccinMocha     Name = "catppuccin-mocha"
)

// AllNames returns every available theme name in display order.
func AllNames() []Name {
	return []Name{
		Dark,
		Light,
		GithubDark,
		GithubLight,
		CatppuccinLatte,
		CatppuccinFrappe,
		CatppuccinMacchiato,
		CatppuccinMocha,
	}
}

// Label returns a human-readable name for the theme.
func (n Name) Label() string {
	switch n {
	case Dark:
		return "Dark"
	case Light:
		return "Light"
	case GithubDark:
		return "GitHub Dark"
	case GithubLight:
		return "GitHub Light"
	case CatppuccinLatte:
		return "Catppuccin Latte"
	case CatppuccinFrappe:
		return "Catppuccin Frappé"
	case CatppuccinMacchiato:
		return "Catppuccin Macchiato"
	case CatppuccinMocha:
		return "Catppuccin Mocha"
	default:
		return string(n)
	}
}

// Theme describes the active Material color scheme.
type Theme struct {
	Name    Name
	IsDark  bool
	Primary color.RGBA
}

var current Theme

// Current returns the active theme.
func Current() Theme {
	return current
}

// Init configures fonts and global widget styles.
// Call [settings.Init] and [core.LoadAllSettings] before core.NewBody.
func Init() {
	setupFonts()
	setupGlobalStyles()
}

// Apply activates the theme with the given name using Cogent Core's
// Material Design 3 color generation.
func Apply(name Name) Theme {
	def, ok := schemaFor(name)
	if !ok {
		return Apply(Dark)
	}

	current = Theme{
		Name:    name,
		IsDark:  def.isDark,
		Primary: def.primary,
	}
	applyToCogentCore(def.primary, def.isDark)
	return current
}

// OnSurface returns the current on-surface color from the active scheme.
func OnSurface() color.RGBA {
	return colors.ToUniform(colors.Scheme.OnSurface)
}

// Surface returns the current surface color from the active scheme.
func Surface() color.RGBA {
	return colors.ToUniform(colors.Scheme.Surface)
}

// Outline returns the current outline color from the active scheme.
func Outline() color.RGBA {
	return colors.ToUniform(colors.Scheme.Outline)
}
