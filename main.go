package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/unit"
	mainApp "github.com/mirzakhany/chapar/ui/app"
)

func main() {
	go func() {
		var w app.Window
		w.Option(app.Title("Chapar"), app.Size(unit.Dp(1200), unit.Dp(800)))

		mainUI, err := mainApp.New(&w)
		if err != nil {
			log.Fatal(err)
		}
		if err := mainUI.Run(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}
