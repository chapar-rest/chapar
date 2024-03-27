package loader

import (
	"os"
	"path"
	"path/filepath"

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
		collection.FilePath = filepath.Join(dirName, "_collection.yaml")
	}

	if err := SaveToYaml(collection.FilePath, collection); err != nil {
		return err
	}

	// Get the directory name
	dirName := path.Dir(collection.FilePath)
	// Change the directory name to the collection name
	if collection.MetaData.Name != path.Base(dirName) {
		// replace last part of the path with the new name
		newDirName := path.Join(path.Dir(dirName), collection.MetaData.Name)
		if err := os.Rename(dirName, newDirName); err != nil {
			return err
		}
		collection.FilePath = filepath.Join(newDirName, "_collection.yaml")
	}

	return nil
}

func DeleteCollection(collection *domain.Collection) error {
	return os.RemoveAll(path.Dir(collection.FilePath))
}
