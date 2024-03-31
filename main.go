package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/unit"
	mainApp "github.com/mirzakhany/chapar/ui/app"
)

func main() {
	mainUI, err := mainApp.New()
	if err != nil {
		log.Fatal(err)
	}

	w := app.NewWindow(app.Title("Chapar"), app.Size(unit.Dp(1200), unit.Dp(800)))
	go func() {
		if err := mainUI.Run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}
