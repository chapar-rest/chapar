package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/chapar-rest/chapar/internal/gitx"
	"github.com/chapar-rest/chapar/ui"
)

var enablePprof = flag.Bool("pprof", false, "enable pprof")

func main() {
	// git runs chapar as its askpass helper to get a remote's credentials.
	if os.Getenv(gitx.AskPassEnv) == "1" {
		os.Exit(gitx.AskPass(os.Args[1:], os.Stdout))
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
