package main

import (
	"example.com/gio_test/ui"
	"gioui.org/app"
	"gioui.org/unit"
	"log"
	"os"
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
			app.Size(unit.Dp(800), unit.Dp(600)),
		)
		if err := appUI.Run(w); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()

}
