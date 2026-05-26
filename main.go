package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/chapar-rest/chapar/ui/uiv1"
	"github.com/chapar-rest/chapar/uiv2"
)

var (
	enablePprof = flag.Bool("pprof", false, "enable pprof")
	uiVersion   = flag.String("ui", "v1", "ui version to use")
)

func main() {
	flag.Parse()

	if *enablePprof {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	switch *uiVersion {
	case "v1":
		uiv1.Run()
	case "v2":
		uiv2.Run()
	default:
		log.Fatalf("invalid ui version: %s", *uiVersion)
	}
}
