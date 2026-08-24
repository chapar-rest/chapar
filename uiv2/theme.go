package uiv2

import "github.com/mirzakhany/yoga/theme"

func applyChaparTheme(name string) {
	if name == "" || !theme.Use(name) {
		theme.Use("yoga-dark")
	}
}
