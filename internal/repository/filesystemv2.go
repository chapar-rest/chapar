package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/domain"
)

type Entity interface {
	ID() string
	GetKind() string
	GetName() string
	SetName(name string)
	MarshalYaml() ([]byte, error)
}

type FilesystemV2 struct {
	dataDir       string
	workspaceName string

	// entities is a map to hold loaded entities so filesystem can name changes
	entities map[string]string
}

func NewFilesystemV2(dataDir, workspaceName string) *FilesystemV2 {
	return &FilesystemV2{
		dataDir:       dataDir,
		workspaceName: workspaceName,
		entities:      make(map[string]string),
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
	f.entities[protoFile.ID()] = protoFile.GetName()
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

	if err := f.deleteEntity(path, protoFile); err != nil {
		return err
	}

	// Remove the proto file from the entities map
	delete(f.entities, protoFile.ID())
	return nil
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
	// add the request to the entities map but break the pointer to avoid sharing the same object
	f.entities[request.ID()] = request.GetName()
	return f.writeStandaloneRequest(request, false)
}

func (f *FilesystemV2) UpdateRequest(request *domain.Request, collection *domain.Collection) error {
	oldEntityName, ok := f.entities[request.ID()]
	if !ok {
		return fmt.Errorf("request with ID %s not found", request.ID())
	}

	// did the request change its name?
	if oldEntityName != request.GetName() {
		// as name has changed, we need to rename the file
		path, err := f.EntityPath(domain.KindRequest)
		if err != nil {
			return err
		}

		if err := f.renameEntity(path, oldEntityName+".yaml", request.GetName()+".yaml"); err != nil {
			return fmt.Errorf("cannot rename request with ID %s: %v", request.ID(), err)
		}

		// Update the name in the entities map
		f.entities[request.ID()] = request.GetName()
	}

	return f.writeStandaloneRequest(request, true)
}

func (f *FilesystemV2) DeleteRequest(request *domain.Request, collection *domain.Collection) error {
	dir, err := f.EntityPath(domain.KindRequest)
	if err != nil {
		return err
	}

	// TODO: Handle collection-specific requests if here

	if err := f.deleteEntity(dir, request); err != nil {
		return err
	}

	// Remove the request from the entities map
	delete(f.entities, request.ID())
	return nil
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
	f.entities[collection.ID()] = collection.GetName()
	return f.writeCollection(collection)
}

func (f *FilesystemV2) UpdateCollection(collection *domain.Collection) error {
	// if the collection already exists, we can just update it otherwise, it means the name has changed
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	// Check if the collection name has changed
	oldEntityName, ok := f.entities[collection.ID()]
	if ok && oldEntityName != collection.GetName() {
		// Rename the collection directory if the name has changed
		if err := f.renameEntity(path, oldEntityName, collection.GetName()); err != nil {
			return fmt.Errorf("cannot rename collection with ID %s: %v", collection.ID(), err)
		}

		// Update the name in the entities map
		f.entities[collection.ID()] = collection.GetName()
	}

	collectionPath := filepath.Join(path, collection.GetName())
	// if collection already exists, we can just update it
	if _, err := os.Stat(filepath.Join(collectionPath, "_collection.yaml")); err == nil {
		return f.writeMetadataFile(collectionPath, "_collection", collection)
	}

	return f.writeCollection(collection)
}

func (f *FilesystemV2) DeleteCollection(collection *domain.Collection) error {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	// Delete the collection directory
	if err := f.deleteEntity(path, collection); err != nil {
		return err
	}

	// Remove the collection from the entities map
	delete(f.entities, collection.ID())
	return nil
}

func (f *FilesystemV2) writeCollection(collection *domain.Collection) error {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	collectionDir := filepath.Join(path, collection.GetName())
	// Ensure the collection directory exists
	if _, err := os.Stat(collectionDir); os.IsNotExist(err) {
		// Create the collection directory if it does not exist
		if err := os.MkdirAll(collectionDir, 0755); err != nil {
			return fmt.Errorf("failed to create collection directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check collection directory: %w", err)
	}

	// Update the collection metadata file
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

func (f *FilesystemV2) renameEntity(path string, oldName, newName string) error {
	oldFilePath := filepath.Join(path, oldName)
	newFilePath := filepath.Join(path, newName)

	// Check if the old file exists
	if _, err := os.Stat(oldFilePath); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", oldFilePath)
	}

	// Rename the entity
	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		return fmt.Errorf("failed to rename file from %s to %s: %w", oldFilePath, newFilePath, err)
	}

	return nil
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
