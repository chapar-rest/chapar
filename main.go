package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/unit"
	"github.com/mirzakhany/chapar/ui"
)

func main() {
	appUI, err := ui.New()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		// create main window
		w := app.NewWindow(
			app.Title("Chapar"),
			app.Size(unit.Dp(1200), unit.Dp(800)),
		)
		if err := appUI.Run(w); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
