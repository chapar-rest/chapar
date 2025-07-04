package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/domain"
)

type Entity interface {
	GetKind() string
	GetName() string
	SetName(name string)
	MarshalYaml() ([]byte, error)
}

type FilesystemV2 struct {
	dataDir       string
	workspaceName string
}

func NewFilesystemV2(dataDir, workspaceName string) *FilesystemV2 {
	return &FilesystemV2{
		dataDir:       dataDir,
		workspaceName: workspaceName,
	}
}

func (f *FilesystemV2) LoadProtoFiles() ([]*domain.ProtoFile, error) {
	dir, err := f.EntityPath(domain.KindProtoFile)
	if err != nil {
		return nil, err
	}

	return loadList[domain.ProtoFile](dir)
}

func (f *FilesystemV2) CreateProtoFile(protoFile *domain.ProtoFile) error {
	return f.writeProtoFile(protoFile, false)
}

func (f *FilesystemV2) UpdateProtoFile(protoFile *domain.ProtoFile) error {
	return f.writeProtoFile(protoFile, true)
}

func (f *FilesystemV2) DeleteProtoFile(protoFile *domain.ProtoFile) error {
	path, err := f.EntityPath(protoFile.GetKind())
	if err != nil {
		return err
	}

	return f.deleteEntity(path, protoFile)
}

// LoadRequests loads standalone requests from the filesystem.
func (f *FilesystemV2) LoadRequests() ([]*domain.Request, error) {
	dir, err := f.EntityPath(domain.KindRequest)
	if err != nil {
		return nil, err
	}

	return loadList[domain.Request](dir)
}

func (f *FilesystemV2) CreateRequest(request *domain.Request, collection *domain.Collection) error {
	return f.writeStandaloneRequest(request, false)
}

func (f *FilesystemV2) UpdateRequest(request *domain.Request, collection *domain.Collection) error {
	return f.writeStandaloneRequest(request, true)
}

func (f *FilesystemV2) DeleteRequest(request *domain.Request, collection *domain.Collection) error {
	dir, err := f.EntityPath(domain.KindRequest)
	if err != nil {
		return err
	}

	// TODO: Handle collection-specific requests if here

	return f.deleteEntity(dir, request)
}

func (f *FilesystemV2) LoadCollections() ([]*domain.Collection, error) {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return nil, err
	}

	// Load all collection directories
	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection directory: %w", err)
	}

	var collections []*domain.Collection
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue // Skip non-directory entries
		}

		// Each collection directory should have a "_collection.yaml" file
		collectionFile := filepath.Join(path, dir.Name(), "_collection.yaml")
		if _, err := os.Stat(collectionFile); os.IsNotExist(err) {
			continue // Skip if the collection file does not exist
		}

		// Load the collection from the YAML file
		collection, err := LoadFromYaml[domain.Collection](collectionFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load collection %s: %w", dir.Name(), err)
		}

		collections = append(collections, collection)
	}

	return collections, nil
}

func (f *FilesystemV2) CreateCollection(collection *domain.Collection) error {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	collectionDir := filepath.Join(path, collection.GetName())
	// Ensure the collection directory exists
	if err := os.MkdirAll(collectionDir, 0755); err != nil {
		return fmt.Errorf("failed to create collection directory: %w", err)
	}

	// Collection folder always has a file named "_collection.yaml" which contains the collection metadata
	return f.writeMetadataFile(collectionDir, "_collection", collection)
}

func (f *FilesystemV2) writeMetadataFile(path, name string, e Entity) error {
	data, err := e.MarshalYaml()
	if err != nil {
		return fmt.Errorf("failed to marshal entity %s", e.GetName())
	}

	filePath := filepath.Join(path, name+".yaml")
	return os.WriteFile(filePath, data, 0644)
}

func (f *FilesystemV2) writeStandaloneRequest(request *domain.Request, override bool) error {
	path, err := f.EntityPath(domain.KindRequest)
	if err != nil {
		return err
	}
	return f.writeRequestOrProtoFile(path, request, override)
}

func (f *FilesystemV2) writeProtoFile(protoFile *domain.ProtoFile, override bool) error {
	path, err := f.EntityPath(protoFile.GetKind())
	if err != nil {
		return err
	}
	return f.writeRequestOrProtoFile(path, protoFile, override)
}

func (f *FilesystemV2) deleteEntity(path string, e Entity) error {
	// if the entity is a workspace or a collection, we need to delete the entire directory
	if e.GetKind() == domain.KindWorkspace || e.GetKind() == domain.KindCollection {
		dir := filepath.Join(path, e.GetName())
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to delete entity directory: %w", err)
		}
		return nil
	} else {
		// For other entities, we delete the specific file
		filePath := filepath.Join(path, e.GetName()+".yaml")
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to delete entity: %w", err)
		}
	}

	return nil
}

// writeRequestOrProtoFile, writes the protofile or request files to the filesystem.
func (f *FilesystemV2) writeRequestOrProtoFile(path string, e Entity, override bool) error {
	filename := e.GetName()
	if !override {
		uniqueName, err := f.ensureUniqueName(path, e.GetName())
		if err != nil {
			return err
		}

		// Set the unique name to the entity
		e.SetName(uniqueName)
		filename = uniqueName
	}

	data, err := e.MarshalYaml()
	if err != nil {
		return fmt.Errorf("failed to marshal entity %s", e.GetName())
	}

	filePath := filepath.Join(path, filename+".yaml")
	return os.WriteFile(filePath, data, 0644)
}

func (f *FilesystemV2) ensureUniqueName(path, name string) (string, error) {
	fileName := name + ".yaml"
	filePath := filepath.Join(path, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return name, nil
	}

	// If the file already exists, append a suffix to make it unique
	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s_%d", name, i)
		newFilePath := filepath.Join(path, newName+".yaml")
		if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
			return newName, nil
		}
	}
}

func (f *FilesystemV2) EntityPath(kind string) (string, error) {
	var path string
	switch kind {
	case domain.KindProtoFile:
		path = filepath.Join(f.dataDir, f.workspaceName, "protofiles")
	case domain.KindCollection:
		path = filepath.Join(f.dataDir, f.workspaceName, "collections")
	case domain.KindEnv:
		path = filepath.Join(f.dataDir, f.workspaceName, "environments")
	case domain.KindRequest:
		path = filepath.Join(f.dataDir, f.workspaceName, "requests")
	default:
		path = filepath.Join(f.dataDir)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}

	return path, nil
}

func loadList[T any](dir string) ([]*T, error) {
	var out []*T

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if item, err := LoadFromYaml[T](file); err != nil {
			return nil, err
		} else {
			out = append(out, item)
		}
	}

	return out, nil
}
