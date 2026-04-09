package repository

import (
	"fmt"
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
)

// Entity is the interface that all repository-managed domain objects must satisfy.
type Entity interface {
	ID() string
	GetKind() string
	GetName() string
	SetName(name string)
	MarshalYaml() ([]byte, error)
}

// FilesystemV2 implements RepositoryV2 by delegating all file I/O to a
// StorageBackend. By default (via NewFilesystemV2) it uses LocalStorage, but
// any StorageBackend can be injected via NewFilesystemV2WithBackend.
type FilesystemV2 struct {
	dataDir       string
	workspaceName string

	// entities tracks the stored name of each entity by ID so that renames can
	// be detected and the old file removed atomically during updates.
	entities *safemap.Map[string]

	storage StorageBackend
}

// NewFilesystemV2 creates a FilesystemV2 backed by the local OS filesystem.
// The external signature is unchanged for backward compatibility.
func NewFilesystemV2(dataDir, workspaceName string) (*FilesystemV2, error) {
	return NewFilesystemV2WithBackend(dataDir, workspaceName, NewLocalStorage())
}

// NewFilesystemV2WithBackend creates a FilesystemV2 using the provided StorageBackend.
// Use this in tests (with an in-memory backend) or for alternative storage targets.
func NewFilesystemV2WithBackend(dataDir, workspaceName string, storage StorageBackend) (*FilesystemV2, error) {
	fs := &FilesystemV2{
		dataDir:       dataDir,
		workspaceName: workspaceName,
		entities:      safemap.New[string](),
		storage:       storage,
	}

	if err := storage.MkdirAll(dataDir); err != nil {
		return nil, fmt.Errorf("could not create directory %s: %w", dataDir, err)
	}

	workspacePath := filepath.Join(dataDir, workspaceName)
	exists, err := storage.Stat(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("could not stat workspace directory: %w", err)
	}
	if !exists {
		if err := storage.MkdirAll(workspacePath); err != nil {
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

	return loadList[domain.ProtoFile](dir, f.storage, func(n *domain.ProtoFile) {
		f.entities.Set(n.ID(), n.GetName())
	})
}

func (f *FilesystemV2) CreateProtoFile(protoFile *domain.ProtoFile) error {
	f.entities.Set(protoFile.ID(), protoFile.GetName())
	return f.writeProtoFile(protoFile, false, "")
}

func (f *FilesystemV2) UpdateProtoFile(protoFile *domain.ProtoFile) error {
	oldEntityName, ok := f.entities.Get(protoFile.ID())
	if !ok {
		return fmt.Errorf("proto file with ID %s not found", protoFile.ID())
	}

	if oldEntityName != protoFile.GetName() {
		path, err := f.EntityPath(protoFile.GetKind())
		if err != nil {
			return err
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(f.storage, filepath.Join(path, protoFile.GetName()+".yaml"), protoFile.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			n := f.ensureUniqueName(path, protoFile.GetName(), ".yaml")
			protoFile.SetName(n)
		}

		f.entities.Set(protoFile.ID(), protoFile.GetName())
	}

	return f.writeProtoFile(protoFile, true, oldEntityName)
}

func (f *FilesystemV2) DeleteProtoFile(protoFile *domain.ProtoFile) error {
	path, err := f.EntityPath(protoFile.GetKind())
	if err != nil {
		return err
	}

	if err := f.deleteEntity(path, protoFile); err != nil {
		return err
	}

	f.entities.Delete(protoFile.ID())
	return nil
}

// LoadRequests loads standalone requests (not part of any collection).
func (f *FilesystemV2) LoadRequests() ([]*domain.Request, error) {
	dir, err := f.EntityPath(domain.KindRequest)
	if err != nil {
		return nil, err
	}

	return loadList[domain.Request](dir, f.storage, func(n *domain.Request) {
		n.SetDefaultValues() // Bug fix: apply defaults on load, consistent with collection requests
		f.entities.Set(n.ID(), n.GetName())
	})
}

func (f *FilesystemV2) CreateRequest(request *domain.Request, collection *domain.Collection) error {
	f.entities.Set(request.ID(), request.GetName())
	return f.writeStandaloneRequest(request, collection, false, "")
}

func (f *FilesystemV2) UpdateRequest(request *domain.Request, collection *domain.Collection) error {
	oldEntityName, ok := f.entities.Get(request.ID())
	if !ok {
		return fmt.Errorf("request with ID %s not found", request.ID())
	}

	if oldEntityName != request.GetName() {
		kind := domain.KindRequest
		if collection != nil {
			kind = domain.KindCollection
		}
		path, err := f.EntityPath(kind)
		if err != nil {
			return err
		}

		if collection != nil {
			path = filepath.Join(path, collection.GetName())
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(f.storage, filepath.Join(path, request.GetName()+".yaml"), request.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			request.SetName(f.ensureUniqueName(path, request.GetName(), ".yaml"))
		}

		f.entities.Set(request.ID(), request.GetName())
	}

	return f.writeStandaloneRequest(request, collection, true, oldEntityName)
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

	if collection != nil {
		dir = filepath.Join(dir, collection.GetName())
	}

	if err := f.deleteEntity(dir, request); err != nil {
		return err
	}

	f.entities.Delete(request.ID())
	return nil
}

func (f *FilesystemV2) LoadCollections() ([]*domain.Collection, error) {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return nil, err
	}

	entries, err := f.storage.ListEntries(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection directory: %w", err)
	}

	collections := make([]*domain.Collection, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir {
			continue
		}

		collectionFile := filepath.Join(path, entry.Name, "_collection.yaml")
		exists, err := f.storage.Stat(collectionFile)
		if err != nil {
			return nil, fmt.Errorf("failed to stat collection file: %w", err)
		}
		if !exists {
			continue
		}

		collection, err := loadFromYaml[domain.Collection](f.storage, collectionFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load collection %s: %w", entry.Name, err)
		}

		requests, err := f.loadCollectionRequests(filepath.Join(path, entry.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to load requests for collection %s: %w", entry.Name, err)
		}
		collection.Spec.Requests = requests

		collections = append(collections, collection)
		f.entities.Set(collection.ID(), collection.GetName())
	}

	return collections, nil
}

func (f *FilesystemV2) loadCollectionRequests(path string) ([]*domain.Request, error) {
	return loadList(path, f.storage, func(n *domain.Request) {
		n.SetDefaultValues()
		f.entities.Set(n.ID(), n.GetName())
	})
}

func (f *FilesystemV2) CreateCollection(collection *domain.Collection) error {
	f.entities.Set(collection.ID(), collection.GetName())
	return f.writeCollection(collection, false)
}

func (f *FilesystemV2) UpdateCollection(collection *domain.Collection) error {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	oldEntityName, ok := f.entities.Get(collection.ID())
	if ok && oldEntityName != collection.GetName() {
		potentialExistingCollectionPath := filepath.Join(path, collection.GetName(), "_collection.yaml")
		anotherFileExists, err := doesFileNameExistWithDifferentID(f.storage, potentialExistingCollectionPath, collection.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			collection.SetName(f.ensureUniqueName(path, collection.GetName(), ""))
		}

		if err := f.renameEntity(path, oldEntityName, collection.GetName()); err != nil {
			return fmt.Errorf("cannot rename collection with ID %s: %w", collection.ID(), err)
		}

		f.entities.Set(collection.ID(), collection.GetName())
	}

	collectionPath := filepath.Join(path, collection.GetName())
	exists, err := f.storage.Stat(filepath.Join(collectionPath, "_collection.yaml"))
	if err != nil {
		return fmt.Errorf("failed to check collection metadata file: %w", err)
	}
	if exists {
		return f.writeMetadataFile(collectionPath, "_collection", collection)
	}

	return f.writeCollection(collection, true)
}

func (f *FilesystemV2) DeleteCollection(collection *domain.Collection) error {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	if err := f.deleteEntity(path, collection); err != nil {
		return err
	}

	f.entities.Delete(collection.ID())
	return nil
}

func (f *FilesystemV2) LoadEnvironments() ([]*domain.Environment, error) {
	path, err := f.EntityPath(domain.KindEnv)
	if err != nil {
		return nil, err
	}

	return loadList[domain.Environment](path, f.storage, func(n *domain.Environment) {
		f.entities.Set(n.ID(), n.GetName())
	})
}

func (f *FilesystemV2) CreateEnvironment(environment *domain.Environment) error {
	f.entities.Set(environment.ID(), environment.GetName())
	return f.writeEnvironmentFile(environment, false, "")
}

func (f *FilesystemV2) UpdateEnvironment(environment *domain.Environment) error {
	oldEntityName, ok := f.entities.Get(environment.ID())
	if !ok {
		return fmt.Errorf("environment with ID %s not found", environment.ID())
	}

	if oldEntityName != environment.GetName() {
		path, err := f.EntityPath(domain.KindEnv)
		if err != nil {
			return err
		}

		anotherFileExists, err := doesFileNameExistWithDifferentID(f.storage, filepath.Join(path, environment.GetName()+".yaml"), environment.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			environment.SetName(f.ensureUniqueName(path, environment.GetName(), ".yaml"))
		}

		f.entities.Set(environment.ID(), environment.GetName())
	}

	return f.writeEnvironmentFile(environment, true, oldEntityName)
}

func (f *FilesystemV2) DeleteEnvironment(environment *domain.Environment) error {
	path, err := f.EntityPath(environment.GetKind())
	if err != nil {
		return err
	}

	if err := f.deleteEntity(path, environment); err != nil {
		return err
	}

	f.entities.Delete(environment.ID())
	return nil
}

func (f *FilesystemV2) LoadWorkspaces() ([]*domain.Workspace, error) {
	path := f.dataDir

	entries, err := f.storage.ListEntries(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	workspaces := make([]*domain.Workspace, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir {
			continue
		}

		workspaceFile := filepath.Join(path, entry.Name, "_workspace.yaml")
		exists, err := f.storage.Stat(workspaceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to stat workspace file: %w", err)
		}
		if !exists {
			continue
		}

		workspace, err := loadFromYaml[domain.Workspace](f.storage, workspaceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load workspace %s: %w", entry.Name, err)
		}

		workspaces = append(workspaces, workspace)
		f.entities.Set(workspace.ID(), workspace.GetName())
	}

	return workspaces, nil
}

func (f *FilesystemV2) CreateWorkspace(workspace *domain.Workspace) error {
	f.entities.Set(workspace.ID(), workspace.GetName())
	return f.writeWorkspace(workspace, false)
}

func (f *FilesystemV2) UpdateWorkspace(workspace *domain.Workspace) error {
	oldEntityName, ok := f.entities.Get(workspace.ID())
	if !ok {
		return fmt.Errorf("workspace with ID %s not found", workspace.ID())
	}

	if oldEntityName != workspace.GetName() {
		potentialExistingWorkspacePath := filepath.Join(f.dataDir, workspace.GetName(), "_workspace.yaml")
		anotherFileExists, err := doesFileNameExistWithDifferentID(f.storage, potentialExistingWorkspacePath, workspace.ID())
		if err != nil {
			return fmt.Errorf("failed to check if another file with the same name exists: %w", err)
		}
		if anotherFileExists {
			workspace.SetName(f.ensureUniqueName(f.dataDir, workspace.GetName(), ""))
		}

		if err := f.renameEntity(f.dataDir, oldEntityName, workspace.GetName()); err != nil {
			return fmt.Errorf("cannot rename workspace with ID %s: %w", workspace.ID(), err)
		}

		f.entities.Set(workspace.ID(), workspace.GetName())
	}

	return f.writeWorkspace(workspace, true)
}

func (f *FilesystemV2) DeleteWorkspace(workspace *domain.Workspace) error {
	if err := f.deleteEntity(f.dataDir, workspace); err != nil {
		return err
	}

	f.entities.Delete(workspace.ID())
	return nil
}

// ── private write helpers ──────────────────────────────────────────────────

func (f *FilesystemV2) writeWorkspace(workspace *domain.Workspace, override bool) error {
	path, err := f.EntityPath(domain.KindWorkspace)
	if err != nil {
		return err
	}

	if !override {
		uniqueName := f.ensureUniqueName(path, workspace.GetName(), "")
		workspace.SetName(uniqueName)
	}

	workspaceDir := filepath.Join(path, workspace.GetName())
	exists, err := f.storage.Stat(workspaceDir)
	if err != nil {
		return fmt.Errorf("failed to check workspace directory: %w", err)
	}
	if !exists {
		if err := f.storage.MkdirAll(workspaceDir); err != nil {
			return fmt.Errorf("failed to create workspace directory: %w", err)
		}
	}

	if err := f.writeMetadataFile(workspaceDir, "_workspace", workspace); err != nil {
		return err
	}

	f.entities.Set(workspace.ID(), workspace.GetName())
	return nil
}

func (f *FilesystemV2) writeCollection(collection *domain.Collection, override bool) error {
	path, err := f.EntityPath(domain.KindCollection)
	if err != nil {
		return err
	}

	if !override {
		uniqueName := f.ensureUniqueName(path, collection.GetName(), "")
		collection.SetName(uniqueName)
	}

	collectionDir := filepath.Join(path, collection.GetName())
	exists, err := f.storage.Stat(collectionDir)
	if err != nil {
		return fmt.Errorf("failed to check collection directory: %w", err)
	}
	if !exists {
		if err := f.storage.MkdirAll(collectionDir); err != nil {
			return fmt.Errorf("failed to create collection directory: %w", err)
		}
	}

	return f.writeMetadataFile(collectionDir, "_collection", collection)
}

// writeMetadataFile atomically writes the metadata file name.yaml inside path.
func (f *FilesystemV2) writeMetadataFile(path, name string, e Entity) error {
	data, err := e.MarshalYaml()
	if err != nil {
		return fmt.Errorf("failed to marshal entity %s: %w", e.GetName(), err)
	}

	filePath := filepath.Join(path, name+".yaml")
	tmpPath := filePath + ".tmp"

	if err := f.storage.WriteFile(tmpPath, data); err != nil {
		return fmt.Errorf("failed to write metadata for entity %s: %w", e.GetName(), err)
	}

	if err := f.storage.Rename(tmpPath, filePath); err != nil {
		_ = f.storage.Remove(tmpPath)
		return fmt.Errorf("failed to finalize metadata write for entity %s: %w", e.GetName(), err)
	}

	return nil
}

func (f *FilesystemV2) writeStandaloneRequest(request *domain.Request, collection *domain.Collection, override bool, oldName string) error {
	kind := domain.KindRequest
	if collection != nil {
		kind = domain.KindCollection
	}

	path, err := f.EntityPath(kind)
	if err != nil {
		return err
	}

	if collection != nil {
		path = filepath.Join(path, collection.GetName())
	}

	return f.writeFile(path, request, override, oldName)
}

func (f *FilesystemV2) writeProtoFile(protoFile *domain.ProtoFile, override bool, oldName string) error {
	path, err := f.EntityPath(protoFile.GetKind())
	if err != nil {
		return err
	}
	return f.writeFile(path, protoFile, override, oldName)
}

func (f *FilesystemV2) writeEnvironmentFile(environment *domain.Environment, override bool, oldName string) error {
	path, err := f.EntityPath(domain.KindEnv)
	if err != nil {
		return err
	}
	return f.writeFile(path, environment, override, oldName)
}

func (f *FilesystemV2) deleteEntity(path string, e Entity) error {
	if e.GetKind() == domain.KindWorkspace || e.GetKind() == domain.KindCollection {
		dir := filepath.Join(path, e.GetName())
		if err := f.storage.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to delete entity directory: %w", err)
		}
		return nil
	}
	filePath := filepath.Join(path, e.GetName()+".yaml")
	if err := f.storage.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}
	return nil
}

// writeFile writes e to dir/name.yaml atomically using a tmp-then-rename strategy.
//
//   - If override is false, a unique name is enforced before writing (create path).
//   - oldName: previous file name without extension. If non-empty and different from
//     e.GetName(), the old file is removed after the new one is in place.
func (f *FilesystemV2) writeFile(dir string, e Entity, override bool, oldName string) error {
	if !override {
		uniqueName := f.ensureUniqueName(dir, e.GetName(), ".yaml")
		e.SetName(uniqueName)
		oldName = "" // creates never have an old file
	}

	data, err := e.MarshalYaml()
	if err != nil {
		return fmt.Errorf("failed to marshal entity %s: %w", e.GetName(), err)
	}

	finalPath := filepath.Join(dir, e.GetName()+".yaml")
	tmpPath := finalPath + ".tmp"

	if err := f.storage.WriteFile(tmpPath, data); err != nil {
		return fmt.Errorf("failed to write entity %s: %w", e.GetName(), err)
	}

	if err := f.storage.Rename(tmpPath, finalPath); err != nil {
		_ = f.storage.Remove(tmpPath) // best-effort cleanup
		return fmt.Errorf("failed to finalize write for entity %s: %w", e.GetName(), err)
	}

	// Remove the old file only when the name actually changed.
	if oldName != "" && oldName != e.GetName() {
		oldPath := filepath.Join(dir, oldName+".yaml")
		if err := f.storage.Remove(oldPath); err != nil {
			return fmt.Errorf("failed to remove old file %s: %w", oldPath, err)
		}
	}

	return nil
}

func (f *FilesystemV2) renameEntity(path, oldName, newName string) error {
	oldFilePath := filepath.Join(path, oldName)
	newFilePath := filepath.Join(path, newName)

	exists, err := f.storage.Stat(oldFilePath)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", oldFilePath, err)
	}
	if !exists {
		return fmt.Errorf("path %s does not exist", oldFilePath)
	}

	if err := f.storage.Rename(oldFilePath, newFilePath); err != nil {
		return fmt.Errorf("failed to rename from %s to %s: %w", oldFilePath, newFilePath, err)
	}

	return nil
}

func (f *FilesystemV2) ensureUniqueName(path, name, extension string) string {
	filePath := filepath.Join(path, name+extension)
	exists, _ := f.storage.Stat(filePath)
	if !exists {
		return name
	}

	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s_%d", name, i)
		newFilePath := filepath.Join(path, newName+extension)
		exists, _ = f.storage.Stat(newFilePath)
		if !exists {
			return newName
		}
	}
}

// EntityPath returns the directory path for the given entity kind, creating it
// if necessary.
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
		// workspace and legacy config files live in dataDir directly
		path = f.dataDir
	}

	if err := f.storage.MkdirAll(path); err != nil {
		return "", err
	}

	return path, nil
}

// GetLegacyConfig returns the legacy config, creating a default one if absent.
func (f *FilesystemV2) GetLegacyConfig() (*domain.Config, error) {
	filePath := filepath.Join(f.dataDir, "config.yaml")

	exists, err := f.storage.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat legacy config: %w", err)
	}

	if !exists {
		config := domain.NewConfig()
		if err := saveToYaml(f.storage, filePath, config); err != nil {
			return nil, err
		}
		return config, nil
	}

	return loadFromYaml[domain.Config](f.storage, filePath)
}

// ReadLegacyPreferences reads the legacy preferences file.
func (f *FilesystemV2) ReadLegacyPreferences() (*domain.Preferences, error) {
	pdir := filepath.Join(f.dataDir, f.workspaceName, "preferences")
	filePath := filepath.Join(pdir, "preferences.yaml")
	return loadFromYaml[domain.Preferences](f.storage, filePath)
}

// ── package-level helpers ──────────────────────────────────────────────────

// loadList loads all *.yaml files (excluding metadata files) from dir via the
// given StorageBackend, unmarshalling each into *T and calling fallback if set.
func loadList[T any](dir string, storage StorageBackend, fallback func(n *T)) ([]*T, error) {
	var out []*T

	files, err := storage.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		base := filepath.Base(file)
		if base == "_collection.yaml" || base == "_workspace.yaml" {
			continue
		}

		item, err := loadFromYaml[T](storage, file)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
		if fallback != nil {
			fallback(item)
		}
	}

	return out, nil
}

// doesFileNameExistWithDifferentID checks whether filePath already exists and
// belongs to a different entity (different ID). Returns (false, nil) when the
// file does not exist, so callers can treat that as "safe to use this name".
func doesFileNameExistWithDifferentID(storage StorageBackend, filePath, id string) (bool, error) {
	type dummy struct {
		MetaData domain.MetaData `yaml:"metadata"`
	}

	exists, err := storage.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	if !exists {
		return false, nil
	}

	v, err := loadFromYaml[dummy](storage, filePath)
	if err != nil {
		return false, fmt.Errorf("failed to load metadata from file %s: %w", filePath, err)
	}

	if v == nil {
		return false, fmt.Errorf("loaded metadata from file %s is nil", filePath)
	}

	return v.MetaData.ID != id, nil
}
