package uiv2

import "github.com/mirzakhany/yoga/theme"

func applyChaparTheme(name string) {
	theme.Use(mapChaparTheme(name))
}

func mapChaparTheme(name string) string {
	switch name {
	case "light":
		return "yoga-light"
	case "github-light":
		return "github-light"
	case "dark":
		return "yoga-dark"
	case "github-dark":
		return "github-dark"
	case "catppuccin-frappe", "catppuccin-macchiato", "catppuccin-mocha", "catppuccin":
		return "catppuccin"
	case "catppuccin-latte":
		return "yoga-light"
	default:
		if name == "" {
			return "yoga-dark"
		}
		for _, n := range theme.Names() {
			if n == name {
				return name
			}
		}
		return "yoga-dark"
	}
}
