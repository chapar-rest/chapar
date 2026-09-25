package ui

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"

	"github.com/chapar-rest/chapar/internal/prefs"
)

// Run starts the Yoga UI. It must be called from the main goroutine.
func Run() error {
	applyChaparTheme(prefs.GetGlobalConfig().Spec.General.Theme)
	cfg := yoga.Config{
		Title:          "Chapar",
		Width:          1200,
		Height:         800,
		CustomTitleBar: true,
	}
	_ = theme.Current() // theme selected before Run so the first clear matches
	return yoga.Run(cfg, BuildApp)
}
