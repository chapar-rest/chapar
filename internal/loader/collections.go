package loader

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/mirzakhany/chapar/internal/domain"
)

// Directory structure of collections
// /collections
//   /collection1
//  	 _collection.yaml  # Metadata file
//		 request1.yaml
//       request2.yaml
//   /collection2
//  	 _collection.yaml  # Metadata file
//		 request1.yaml
//       request2.yaml

func LoadCollections() ([]*domain.Collection, error) {
	dir, err := GetCollectionsDir()
	if err != nil {
		return nil, err
	}

	out := make([]*domain.Collection, 0)

	// Walk through the collections directory
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the root directory
		if path == dir {
			return nil
		}

		// If it's a directory, it's a collection
		if info.IsDir() {
			col, err := loadCollection(path)
			if err != nil {
				return err
			}
			out = append(out, col)
		}

		// Skip further processing since we're only interested in directories here
		return filepath.SkipDir
	})

	return out, err
}

func loadCollection(collectionPath string) (*domain.Collection, error) {
	// Read the collection metadata
	collectionMetadataPath := filepath.Join(collectionPath, "_collection.yaml")
	collectionMetadata, err := os.ReadFile(collectionMetadataPath)
	if err != nil {
		return nil, err
	}

	collection := &domain.Collection{}
	err = yaml.Unmarshal(collectionMetadata, collection)
	if err != nil {
		return nil, err
	}

	collection.FilePath = collectionMetadataPath
	collection.Spec.Requests = make([]*domain.Request, 0)

	// Load requests in the collection
	files, err := os.ReadDir(collectionPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() || file.Name() == "_collection.yaml" {
			continue // Skip directories and the collection metadata file
		}

		requestPath := filepath.Join(collectionPath, file.Name())
		requestData, err := os.ReadFile(requestPath)
		if err != nil {
			return nil, err
		}

		var request = new(domain.Request)
		err = yaml.Unmarshal(requestData, request)
		if err != nil {
			return nil, err
		}

		request.FilePath = requestPath
		collection.Spec.Requests = append(collection.Spec.Requests, request)
	}
	return collection, nil
}

func GetCollectionsDir() (string, error) {
	dir, err := CreateConfigDir()
	if err != nil {
		return "", err
	}

	cdir := path.Join(dir, collectionsDir)
	if _, err := os.Stat(cdir); os.IsNotExist(err) {
		if err := os.Mkdir(cdir, 0755); err != nil {
			return "", err
		}
	}

	return cdir, nil
}

func UpdateCollection(collection *domain.Collection) error {
	if collection.FilePath == "" {
		dirName, err := getNewCollectionDirName(collection.MetaData.Name)
		if err != nil {
			return err
		}

		fmt.Println(dirName)

		collection.FilePath = filepath.Join(dirName, "_collection.yaml")
		fmt.Println(collection.FilePath)
	}

	return SaveToYaml(collection.FilePath, collection)
}
