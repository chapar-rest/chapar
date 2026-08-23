//go:build !nogpu

package main

import (
	"github.com/chapar-rest/chapar/uiv2"
	"github.com/mirzakhany/yoga"
)

func main() {
	cfg := yoga.Config{
		Title:  "Chapar",
		Width:  1100,
		Height: 720,
	}
	if err := yoga.Run(cfg, uiv2.BuildChaparUI); err != nil {
		panic(err)
	}
}
