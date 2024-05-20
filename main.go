package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"gioui.org/app"
	"gioui.org/unit"

	mainApp "github.com/chapar-rest/chapar/ui/app"
)

var (
	enablePprof = flag.Bool("pprof", false, "enable pprof")
)

func main() {
	flag.Parse()

	if *enablePprof {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

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
