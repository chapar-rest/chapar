package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mirzakhany/chapar/ui/importer"
)

var (
	fileType = flag.String("t", "collection", "type of input file (collection or environment)")
	filePath = flag.String("p", "example.json", "path to the input file")
)

func main() {
	flag.Parse()

	if *fileType == "collection" {
		if err := importer.ImportPostmanCollectionFromFile(*filePath); err != nil {
			fmt.Printf("Error importing Postman collection: %v\n", err)
			os.Exit(1)
		}
	} else if *fileType == "environment" {
		if err := importer.ImportPostmanEnvironmentFromFile(*filePath); err != nil {
			fmt.Printf("Error importing Postman environment	: %v\n", err)
		}
	}
}
