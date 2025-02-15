package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
)

var (
	fileType = flag.String("t", "collection", "type of input file (collection or environment)")
	filePath = flag.String("p", "example.json", "path to the input file")
)

func main() {
	flag.Parse()

	repo, err := repository.NewFilesystem()
	if err != nil {
		fmt.Printf("Error creating repository: %v\n", err)
		os.Exit(1)
	}

	if *fileType == "collection" {
		if err := importer.ImportPostmanCollectionFromFile(*filePath, repo); err != nil {
			fmt.Printf("Error importing Postman collection: %v\n", err)
			os.Exit(1)
		}
	} else if *fileType == "environment" {
		if err := importer.ImportPostmanEnvironmentFromFile(*filePath, repo); err != nil {
			fmt.Printf("Error importing Postman environment	: %v\n", err)
		}
	}
}
