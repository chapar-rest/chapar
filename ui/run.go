package ui

import (
	"log"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/theme"

	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/update"
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
	if err := yoga.Run(cfg, BuildApp); err != nil {
		return err
	}
	if restartAfterExit.Load() {
		if err := update.Restart(); err != nil {
			log.Printf("restarting after the update: %v", err)
		}
	}
	return nil
}
