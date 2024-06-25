package main

import (
	"fmt"
	"os"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		panic("no command provided")
	}

	if args[0] == "fix-request-types" {
		fixRequestTypes()
	}
}

func fixRequestTypes() {
	filesystem, err := repository.NewFilesystem()
	if err != nil {
		fmt.Printf("Error creating filesystem: %v\n", err)
		os.Exit(1)
	}

	workspaces, err := filesystem.LoadWorkspaces()
	if err != nil {
		fmt.Printf("Error loading workspaces: %v\n", err)
		os.Exit(1)
	}

	for _, ws := range workspaces {
		if err := filesystem.SetActiveWorkspace(ws); err != nil {
			fmt.Printf("Error setting active workspace: %v\n", err)
			os.Exit(1)
		}

		requests, err := filesystem.LoadRequests()
		if err != nil {
			fmt.Printf("Error loading requests: %v\n", err)
			os.Exit(1)
		}

		updateRequest := func(req *domain.Request) {
			if req.Spec.HTTP != nil {
				req.MetaData.Type = domain.RequestTypeHTTP
			}

			if req.Spec.GRPC != nil {
				req.MetaData.Type = domain.RequestTypeGRPC
			}

			fmt.Println("Updating request", req.MetaData.Name, "type to", req.MetaData.Type)

			if err := filesystem.UpdateRequest(req); err != nil {
				fmt.Printf("Error updating request: %v\n", err)
				os.Exit(1)
			}
		}

		for _, req := range requests {
			updateRequest(req)
		}

		collections, err := filesystem.LoadCollections()
		if err != nil {
			fmt.Printf("Error loading collections: %v\n", err)
			os.Exit(1)
		}

		for _, col := range collections {
			for _, req := range col.Spec.Requests {
				updateRequest(req)
			}
		}

	}

	fmt.Println("Requests fixed")
}
