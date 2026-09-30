package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/chapar-rest/chapar/ui"
)

var enablePprof = flag.Bool("pprof", false, "enable pprof")

func main() {
	flag.Parse()

	if *enablePprof {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
