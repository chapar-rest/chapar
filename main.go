package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/chapar-rest/chapar/internal/testcli"
	"github.com/chapar-rest/chapar/ui"
)

var enablePprof = flag.Bool("pprof", false, "enable pprof")

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		os.Exit(testcli.Main(os.Args[2:], os.Stdout, os.Stderr))
	}

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
