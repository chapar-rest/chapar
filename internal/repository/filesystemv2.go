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

func NewFilesystemV2(dataDir, workspaceName string) (*FilesystemV2, error) {
	fs := &FilesystemV2{
		dataDir:       dataDir,
		workspaceName: workspaceName,
		entities:      make(map[string]string),
	}

	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create directory %s: %w", dataDir, err)
	}

	// Ensure the workspace directory exists
	workspacePath := filepath.Join(dataDir, workspaceName)
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			return nil, fmt.Errorf("could not create workspace directory %s: %w", workspacePath, err)
		}
	}

	return fs, nil
}

func (f *FilesystemV2) SetActiveWorkspace(workspaceName string) {
	f.workspaceName = workspaceName
}

func (f *FilesystemV2) LoadProtoFiles() ([]*domain.ProtoFile, error) {
	dir, err := f.EntityPath(domain.KindProtoFile)
	if err != nil {
		return nil, err
	}

	return loadList[domain.ProtoFile](dir, func(n *domain.ProtoFile) {
		f.entities[n.ID()] = n.GetName()
	})
}

func (f *FilesystemV2) CreateProtoFile(protoFile *domain.ProtoFile) error {
	f.entities[protoFile.ID()] = protoFile.GetName()
	return f.writeProtoFile(protoFile, false)
}

func (f *FilesystemV2) UpdateProtoFile(protoFile *domain.ProtoFile) error {
	oldEntityName, ok := f.entities[protoFile.ID()]
	if !ok {
		return fmt.Errorf("proto file with ID %s not found", protoFile.ID())
	}

	// did the proto file change its name?
	if oldEntityName != protoFile.GetName() {
		// as name has changed, we need to rename the file
		path, err := f.EntityPath(protoFile.GetKind())
		if err != nil {
			return err
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(filepath.Join(path, protoFile.GetName()+".yaml"), protoFile.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			n := f.ensureUniqueName(path, protoFile.GetName(), ".yaml")
			protoFile.SetName(n)
		}

		if err := f.renameEntity(path, oldEntityName+".yaml", protoFile.GetName()+".yaml"); err != nil {
			return fmt.Errorf("cannot rename proto file with ID %s: %v", protoFile.ID(), err)
		}

		// Update the name in the entities map
		f.entities[protoFile.ID()] = protoFile.GetName()
	}

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

	return loadList[domain.Request](dir, func(n *domain.Request) {
		f.entities[n.ID()] = n.GetName()
	})
}

func (f *FilesystemV2) CreateRequest(request *domain.Request, collection *domain.Collection) error {
	// add the request to the entities map but break the pointer to avoid sharing the same object
	f.entities[request.ID()] = request.GetName()
	return f.writeStandaloneRequest(request, collection, false)
}

func (f *FilesystemV2) UpdateRequest(request *domain.Request, collection *domain.Collection) error {
	oldEntityName, ok := f.entities[request.ID()]
	if !ok {
		return fmt.Errorf("request with ID %s not found", request.ID())
	}

	// did the request change its name?
	if oldEntityName != request.GetName() {
		kind := domain.KindRequest
		if collection != nil {
			kind = domain.KindCollection
		}
		// as name has changed, we need to rename the file
		path, err := f.EntityPath(kind)
		if err != nil {
			return err
		}

		if collection != nil {
			// if request is part of a collection, we need to add the collection name to the path
			path = filepath.Join(path, collection.GetName())
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(filepath.Join(path, request.GetName()+".yaml"), request.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			request.SetName(f.ensureUniqueName(path, request.GetName(), ".yaml"))
		}

		if err := f.renameEntity(path, oldEntityName+".yaml", request.GetName()+".yaml"); err != nil {
			return fmt.Errorf("cannot rename request with ID %s: %v", request.ID(), err)
		}

		// Update the name in the entities map
		f.entities[request.ID()] = request.GetName()
	}

	return f.writeStandaloneRequest(request, collection, true)
}

func (f *FilesystemV2) DeleteRequest(request *domain.Request, collection *domain.Collection) error {
	kind := domain.KindRequest
	if collection != nil {
		kind = domain.KindCollection
	}

	dir, err := f.EntityPath(kind)
	if err != nil {
		return err
	}

	// If the request is part of a collection, we need to delete it from the collection directory
	if collection != nil {
		dir = filepath.Join(dir, collection.GetName())
	}

	if err := f.deleteEntity(dir, request); err != nil {
		return err
	}

	// Remove the request from the entities map
	delete(f.entities, request.ID())
	return nil
}

func (f *FilesystemV2) GetCollectionByID(id string) (*domain.Collection, error) {
	return getById(f.LoadCollections, id)
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

	collections := make([]*domain.Collection, 0, len(dirs))
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

		requests, err := f.loadCollectionRequests(filepath.Join(path, dir.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to load requests for collection %s: %w", dir.Name(), err)
		}
		collection.Spec.Requests = requests

		collections = append(collections, collection)
		f.entities[collection.ID()] = collection.GetName()
	}

	return collections, nil
}

func (f *FilesystemV2) loadCollectionRequests(path string) ([]*domain.Request, error) {
	return loadList[domain.Request](path, func(n *domain.Request) {
		// set request default values
		n.SetDefaultValues()
		f.entities[n.ID()] = n.GetName()
	})
}

func (f *FilesystemV2) CreateCollection(collection *domain.Collection) error {
	f.entities[collection.ID()] = collection.GetName()
	return f.writeCollection(collection, false)
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
		potentialExistingCollectionPath := filepath.Join(path, collection.GetName(), "_collection.yaml")
		anotherFileExists, err := doesFileNameExistWithDifferentID(potentialExistingCollectionPath, collection.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			collection.SetName(f.ensureUniqueName(path, collection.GetName(), ""))
		}

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

	return f.writeCollection(collection, true)
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

func (f *FilesystemV2) LoadEnvironments() ([]*domain.Environment, error) {
	path, err := f.EntityPath(domain.KindEnv)
	if err != nil {
		return nil, err
	}

	return loadList[domain.Environment](path, func(n *domain.Environment) {
		f.entities[n.ID()] = n.GetName()
	})
}

func (f *FilesystemV2) CreateEnvironment(environment *domain.Environment) error {
	f.entities[environment.ID()] = environment.GetName()
	return f.writeEnvironmentFile(environment, false)
}

func (f *FilesystemV2) UpdateEnvironment(environment *domain.Environment) error {
	oldEntityName, ok := f.entities[environment.ID()]
	if !ok {
		return fmt.Errorf("environment with ID %s not found", environment.ID())
	}

	// did the environment change its name?
	if oldEntityName != environment.GetName() {
		// as name has changed, we need to rename the file
		path, err := f.EntityPath(domain.KindEnv)
		if err != nil {
			return err
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(filepath.Join(path, environment.GetName()+".yaml"), environment.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			environment.SetName(f.ensureUniqueName(path, environment.GetName(), ".yaml"))
		}

		if err := f.renameEntity(path, oldEntityName+".yaml", environment.GetName()+".yaml"); err != nil {
			return fmt.Errorf("cannot rename environment with ID %s: %v", environment.ID(), err)
		}

		// Update the name in the entities map
		f.entities[environment.ID()] = environment.GetName()
	}

	return f.writeEnvironmentFile(environment, true)
}

func (f *FilesystemV2) DeleteEnvironment(environment *domain.Environment) error {
	path, err := f.EntityPath(environment.GetKind())
	if err != nil {
		return err
	}

	if err := f.deleteEntity(path, environment); err != nil {
		return err
	}

	// Remove the environment from the entities map
	delete(f.entities, environment.ID())
	return nil
}

func (f *FilesystemV2) LoadWorkspaces() ([]*domain.Workspace, error) {
	// Workspaces are stored in the dataDir directly
	path := f.dataDir

	// Load all workspace directories
	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	workspaces := make([]*domain.Workspace, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue // Skip non-directory entries
		}

		// Each workspace directory should have a "workspace.yaml" file
		workspaceFile := filepath.Join(path, dir.Name(), "_workspace.yaml")
		if _, err := os.Stat(workspaceFile); os.IsNotExist(err) {
			continue // Skip if the workspace file does not exist
		}

		// Load the workspace from the YAML file
		workspace, err := LoadFromYaml[domain.Workspace](workspaceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load workspace %s: %w", dir.Name(), err)
		}

		workspaces = append(workspaces, workspace)
		f.entities[workspace.ID()] = workspace.GetName()
	}

	return workspaces, nil
}

// CreateWorkspace creates a new workspace and writes it to the filesystem.
func (f *FilesystemV2) CreateWorkspace(workspace *domain.Workspace) error {
	f.entities[workspace.ID()] = workspace.GetName()
	return f.writeWorkspace(workspace, false)
}

// UpdateWorkspace updates an existing workspace and writes it to the filesystem.
func (f *FilesystemV2) UpdateWorkspace(workspace *domain.Workspace) error {
	oldEntityName, ok := f.entities[workspace.ID()]
	if !ok {
		return fmt.Errorf("workspace with ID %s not found", workspace.ID())
	}

	// did the workspace change its name?
	if oldEntityName != workspace.GetName() {
		potentialExistingWorkspacePath := filepath.Join(f.dataDir, workspace.GetName(), "_workspace.yaml")
		anotherFileExists, err := doesFileNameExistWithDifferentID(potentialExistingWorkspacePath, workspace.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			workspace.SetName(f.ensureUniqueName(f.dataDir, workspace.GetName(), ""))
		}

		// as name has changed, we need to rename the file
		if err := f.renameEntity(f.dataDir, oldEntityName, workspace.GetName()); err != nil {
			return fmt.Errorf("cannot rename workspace with ID %s: %v", workspace.ID(), err)
		}

		// Update the name in the entities map
		f.entities[workspace.ID()] = workspace.GetName()
	}

	return f.writeWorkspace(workspace, true)
}

func (f *FilesystemV2) DeleteWorkspace(workspace *domain.Workspace) error {
	// Delete the workspace directory
	if err := f.deleteEntity(f.dataDir, workspace); err != nil {
		return err
	}

	// Remove the workspace from the entities map
	delete(f.entities, workspace.ID())
	return nil
}

func (f *FilesystemV2) writeWorkspace(workspace *domain.Workspace, override bool) error {
	path, err := f.EntityPath(domain.KindWorkspace)
	if err != nil {
		return err
	}

	if !override {
		// Ensure the workspace name is unique
		uniqueName := f.ensureUniqueName(path, workspace.GetName(), "")
		workspace.SetName(uniqueName)
	}

	workspaceDir := filepath.Join(path, workspace.GetName())
	// Ensure the workspace directory exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		// Create the workspace directory if it does not exist
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			return fmt.Errorf("failed to create collection directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check collection directory: %w", err)
	}

	// Update the collection metadata file
	if err := f.writeMetadataFile(workspaceDir, "_workspace", workspace); err != nil {
		return err
	}

	f.entities[workspace.ID()] = workspace.GetName()
	return nil
}

func (f *FilesystemV2) writeCollection(collection *domain.Collection, override bool) error {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	if !override {
		// Ensure the collection name is unique
		uniqueName := f.ensureUniqueName(path, collection.GetName(), "")
		collection.SetName(uniqueName)
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

func (f *FilesystemV2) writeStandaloneRequest(request *domain.Request, collection *domain.Collection, override bool) error {
	kind := domain.KindRequest
	if collection != nil {
		kind = domain.KindCollection
	}

	path, err := f.EntityPath(kind)
	if err != nil {
		return err
	}

	if collection != nil {
		// if the request is part of a collection, we need to add the collection name to the path
		path = filepath.Join(path, collection.GetName())
	}

	return f.writeFile(path, request, override)
}

func (f *FilesystemV2) writeProtoFile(protoFile *domain.ProtoFile, override bool) error {
	path, err := f.EntityPath(protoFile.GetKind())
	if err != nil {
		return err
	}
	return f.writeFile(path, protoFile, override)
}

func (f *FilesystemV2) writeEnvironmentFile(environment *domain.Environment, override bool) error {
	path, err := f.EntityPath(domain.KindEnv)
	if err != nil {
		return err
	}
	return f.writeFile(path, environment, override)
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

// writeFile, writes the protofile, request and environment files to the filesystem.
func (f *FilesystemV2) writeFile(path string, e Entity, override bool) error {
	if !override {
		uniqueName := f.ensureUniqueName(path, e.GetName(), ".yaml")
		// Set the unique name to the entity
		e.SetName(uniqueName)
	}

	data, err := e.MarshalYaml()
	if err != nil {
		return fmt.Errorf("failed to marshal entity %s", e.GetName())
	}

	filePath := filepath.Join(path, e.GetName()+".yaml")
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

func (f *FilesystemV2) ensureUniqueName(path, name, extension string) string {
	filePath := filepath.Join(path, name+extension)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return name
	}

	// If the file already exists, append a suffix to make it unique
	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s_%d", name, i)
		newFilePath := filepath.Join(path, newName+extension)
		if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
			return newName
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
		path = filepath.Join(f.dataDir, f.workspaceName, "envs")
	case domain.KindRequest:
		path = filepath.Join(f.dataDir, f.workspaceName, "requests")
	default:
		// workspace and old config files are living in the dataDir directly
		path = f.dataDir
	}

	// Ensure the directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}

	return path, nil
}

// GetLegacyConfig gets the legacy config from the filesystem.
func (f *FilesystemV2) GetLegacyConfig() (*domain.Config, error) {
	filePath := filepath.Join(f.dataDir, "config.yaml")

	// if config file does not exist, create it
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		config := domain.NewConfig()
		if err := SaveToYaml(filePath, config); err != nil {
			return nil, err
		}

		return config, nil
	}

	return LoadFromYaml[domain.Config](filePath)
}

func doesFileNameExistWithDifferentID(filePath, id string) (bool, error) {
	type Dummy struct {
		MetaData domain.MetaData `yaml:"metadata"`
	}

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// If the file does not exist, we can safely return false
		return false, nil
	}

	v, err := LoadFromYaml[Dummy](filePath)
	if err != nil {
		return false, fmt.Errorf("failed to load metadata from file %s: %w", filePath, err)
	}

	if v == nil {
		return false, fmt.Errorf("loaded metadata from file %s is nil", filePath)
	}

	if v.MetaData.ID != id {
		return true, nil
	}

	return false, nil
}

// ReadLegacyPreferences reads the preferences
func (f *FilesystemV2) ReadLegacyPreferences() (*domain.Preferences, error) {
	pdir := filepath.Join(f.dataDir, f.workspaceName, "preferences")
	filePath := filepath.Join(pdir, "preferences.yaml")
	return LoadFromYaml[domain.Preferences](filePath)
}

func loadList[T any](dir string, fallback func(n *T)) ([]*T, error) {
	var out []*T

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// skip the metadata files
		if filepath.Base(file) == "_collection.yaml" || filepath.Base(file) == "_workspace.yaml" {
			continue
		}

		if item, err := LoadFromYaml[T](file); err != nil {
			return nil, err
		} else {
			out = append(out, item)
			if fallback != nil {
				fallback(item)
			}
		}
	}

	return out, nil
}

func getById[T Entity](list func() ([]T, error), id string) (T, error) {
	items, err := list()
	var zero T

	if err != nil {
		return zero, err
	}

	for _, item := range items {
		if item.ID() == id {
			return item, nil
		}
	}

	return zero, fmt.Errorf("item with ID %s not found", id)
}
