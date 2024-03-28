package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"gioui.org/app"
	"gioui.org/unit"
	"github.com/mirzakhany/chapar/ui"
	mainApp "github.com/mirzakhany/chapar/ui/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	a := ui.NewApplication(ctx)
	mainUI, err := mainApp.New(a)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		a.NewWindow("Chapar", mainUI, app.Size(unit.Dp(1200), unit.Dp(800)))
		a.Wait()
		os.Exit(0)
	}()

	app.Main()
}
