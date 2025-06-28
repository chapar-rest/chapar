package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/util"
)

const (
	environmentsDir = "envs"
	protoFilesDir   = "protofiles"
	collectionsDir  = "collections"
	requestsDir     = "requests"
	preferencesDir  = "preferences"
)

var _ Repository = &Filesystem{}

type Filesystem struct {
	dataDir string

	ActiveWorkspace  *domain.Workspace
	requestPaths     map[string]string
	collectionPaths  map[string]string
	environmentPaths map[string]string
	protoFilePaths   map[string]string
	workspacePaths   map[string]string
}

func (f *Filesystem) SetActiveWorkspace(workspace *domain.Workspace) {
	f.ActiveWorkspace = workspace
}

func NewFilesystem(dataDir string, appState domain.AppStateSpec) (*Filesystem, error) {
	fs := &Filesystem{
		dataDir:          dataDir,
		requestPaths:     make(map[string]string),
		collectionPaths:  make(map[string]string),
		environmentPaths: make(map[string]string),
		protoFilePaths:   make(map[string]string),
		workspacePaths:   make(map[string]string),
	}

	// make sure the data directory exists
	if err := util.MakeDir(dataDir); err != nil {
		return nil, err
	}

	if appState.ActiveWorkspace != nil {
		ws, err := fs.GetWorkspace(filepath.Join(dataDir, appState.ActiveWorkspace.Name))
		if err != nil {
			return nil, err
		}
		fs.ActiveWorkspace = ws

		return fs, nil
	}

	// check if the default workspace directory exists
	defaultWorkspaceDir := filepath.Join(dataDir, domain.DefaultWorkspaceName)
	if _, err := os.Stat(defaultWorkspaceDir); os.IsNotExist(err) {
		// make the default workspace directory
		if err := util.MakeDir(defaultWorkspaceDir); err != nil {
			return nil, err
		}

		ws := domain.NewDefaultWorkspace()
		defaultPath := filepath.Join(dataDir, ws.MetaData.Name, "_workspace.yaml")
		fs.workspacePaths[ws.MetaData.ID] = defaultPath
		if err := fs.updateWorkspace(ws); err != nil {
			return nil, err
		}

		fs.ActiveWorkspace = ws
	} else {
		// if it exists, load the default workspace
		ws, err := fs.GetWorkspace(filepath.Join(dataDir, domain.DefaultWorkspaceName))
		if err != nil {
			return nil, err
		}
		fs.ActiveWorkspace = ws
	}

	return fs, nil
}

func (f *Filesystem) getEntityDirectoryInWorkspace(entityType string) (string, error) {
	p := filepath.Join(f.dataDir, f.ActiveWorkspace.MetaData.Name, entityType)
	if err := util.MakeDir(p); err != nil {
		return "", err
	}

	return p, nil
}

func (f *Filesystem) LoadProtoFiles() ([]*domain.ProtoFile, error) {
	dir, err := f.getEntityDirectoryInWorkspace(protoFilesDir)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]*domain.ProtoFile, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dir, file.Name())

		protoFile, err := LoadFromYaml[domain.ProtoFile](filePath)
		if err != nil {
			return nil, err
		}
		f.protoFilePaths[protoFile.MetaData.ID] = filePath
		out = append(out, protoFile)
	}

	return out, err
}

func (f *Filesystem) updateProtoFile(protoFile *domain.ProtoFile) error {
	filePath, exists := f.protoFilePaths[protoFile.MetaData.ID]
	if !exists {
		// this is a new protoFile
		fileName, err := f.getNewProtoFilePath(protoFile.MetaData.Name)
		if err != nil {
			return err
		}
		filePath = fileName.Path
		f.protoFilePaths[protoFile.MetaData.ID] = filePath
	}

	if err := SaveToYaml(filePath, protoFile); err != nil {
		return err
	}

	// rename the file to the new name
	if protoFile.MetaData.Name != filepath.Base(filePath) {
		newFilePath := filepath.Join(filepath.Dir(filePath), protoFile.MetaData.Name+".yaml")
		if err := os.Rename(filePath, newFilePath); err != nil {
			return err
		}
		f.protoFilePaths[protoFile.MetaData.ID] = newFilePath
	}
	return nil
}

func (f *Filesystem) getNewProtoFilePath(name string) (*FilePath, error) {
	dir, err := f.getEntityDirectoryInWorkspace(protoFilesDir)
	if err != nil {
		return nil, err
	}

	return getNewFilePath(dir, name), nil
}

func (f *Filesystem) GetConfig() (*domain.Config, error) {
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

func (f *Filesystem) LoadWorkspaces() ([]*domain.Workspace, error) {
	dirs, err := os.ReadDir(f.dataDir)
	if err != nil {
		return nil, err
	}

	out := make([]*domain.Workspace, 0)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		dirPath := filepath.Join(f.dataDir, dir.Name())
		if ws, err := f.GetWorkspace(dirPath); err != nil {
			return nil, err
		} else {
			out = append(out, ws)
		}
	}

	return out, nil
}

func (f *Filesystem) GetWorkspace(dirPath string) (*domain.Workspace, error) {
	// if directory is not exist, create it
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, err
		}
	}

	filePath := filepath.Join(dirPath, "_workspace.yaml")

	// if workspace file does not exist, create it
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ws := domain.NewWorkspace(filepath.Base(dirPath))
		f.workspacePaths[ws.MetaData.ID] = filePath
		if err := SaveToYaml(filePath, ws); err != nil {
			return nil, err
		}

		return ws, nil
	}

	ws, err := LoadFromYaml[domain.Workspace](filePath)
	if err != nil {
		return nil, err
	}

	f.workspacePaths[ws.MetaData.ID] = filePath
	return ws, nil
}

func (f *Filesystem) updateWorkspace(workspace *domain.Workspace) error {
	filePath, exists := f.workspacePaths[workspace.MetaData.ID]
	if !exists {
		return fmt.Errorf("workspace path not found for %s", workspace.MetaData.ID)
	}

	if err := SaveToYaml(filePath, workspace); err != nil {
		return err
	}

	// Get the directory name
	dirName := filepath.Dir(filePath)
	// Change the directory name to the workspace name
	if workspace.MetaData.Name != filepath.Base(dirName) {
		// replace last part of the path with the new name
		newDirName := filepath.Join(filepath.Dir(dirName), workspace.MetaData.Name)
		if err := os.Rename(dirName, newDirName); err != nil {
			return err
		}
		f.workspacePaths[workspace.MetaData.ID] = filepath.Join(newDirName, "_workspace.yaml")
	}

	return nil
}

func (f *Filesystem) GetNewWorkspaceDir(name string) (*FilePath, error) {
	dir := filepath.Join(f.dataDir, name)
	if !dirExist(dir) {
		return &FilePath{
			Path:    dir,
			NewName: name,
		}, nil
	}

	// If the file exists, append a number to the filename.
	for i := 1; ; i++ {
		newDirName := fmt.Sprintf("%s%d", dir, i)
		if !dirExist(newDirName) {
			return &FilePath{
				Path:    newDirName,
				NewName: fmt.Sprintf("%s%d", name, i),
			}, nil
		}
	}
}

func (f *Filesystem) GetCollectionRequestNewFilePath(collection *domain.Collection, name string) (*FilePath, error) {
	dir, exists := f.collectionPaths[collection.MetaData.ID]
	if !exists {
		return nil, fmt.Errorf("collection path not found")
	}
	return getNewFilePath(filepath.Dir(dir), name), nil
}

func (f *Filesystem) LoadCollections() ([]*domain.Collection, error) {
	dir, err := f.getEntityDirectoryInWorkspace(collectionsDir)
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
			col, err := f.loadCollection(path)
			if err != nil {
				return fmt.Errorf("failed to load collection, path: %s, %w", path, err)
			}
			out = append(out, col)
		}
		return nil
	})

	return out, err
}

func (f *Filesystem) loadCollection(collectionPath string) (*domain.Collection, error) {
	// Read the collection metadata
	collectionMetadataPath := filepath.Join(collectionPath, "_collection.yaml")
	collectionMetadata, err := os.ReadFile(collectionMetadataPath)
	if err != nil {
		return nil, err
	}

	collection := &domain.Collection{}
	if err = yaml.Unmarshal(collectionMetadata, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection %s, %w", collectionMetadata, err)
	}

	f.collectionPaths[collection.MetaData.ID] = collectionMetadataPath
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
		req, err := LoadFromYaml[domain.Request](requestPath)
		if err != nil {
			return nil, err
		}

		// set request default values
		req.SetDefaultValues()
		f.requestPaths[req.MetaData.ID] = requestPath
		req.CollectionName = collection.MetaData.Name
		collection.Spec.Requests = append(collection.Spec.Requests, req)
	}
	return collection, nil
}

func (f *Filesystem) updateCollection(collection *domain.Collection) error {
	filePath, exists := f.collectionPaths[collection.MetaData.ID]
	if !exists {
		// if directory is not exist, create it
		dirPath := filepath.Join(collection.MetaData.Name, "_collection.yaml")
		if err := os.MkdirAll(filepath.Dir(dirPath), 0755); err != nil {
			return err
		}
		filePath = dirPath
		f.collectionPaths[collection.MetaData.ID] = filePath
	}

	if err := SaveToYaml(filePath, collection); err != nil {
		return err
	}

	// Get the directory name
	dirName := filepath.Dir(filePath)
	// Change the directory name to the collection name
	if collection.MetaData.Name != filepath.Base(dirName) {
		// replace last part of the path with the new name
		newDirName := filepath.Join(filepath.Dir(dirName), collection.MetaData.Name)
		if err := os.Rename(dirName, newDirName); err != nil {
			return err
		}
		f.collectionPaths[collection.MetaData.ID] = filepath.Join(newDirName, "_collection.yaml")
	}

	return nil
}

func (f *Filesystem) LoadEnvironments() ([]*domain.Environment, error) {
	dir, err := f.getEntityDirectoryInWorkspace(environmentsDir)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]*domain.Environment, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dir, file.Name())
		env, err := LoadFromYaml[domain.Environment](filePath)
		if err != nil {
			return nil, err
		}
		f.environmentPaths[env.MetaData.ID] = filePath
		out = append(out, env)
	}

	return out, nil
}

func (f *Filesystem) GetEnvironment(id string) (*domain.Environment, error) {
	filePath, exists := f.environmentPaths[id]
	if !exists {
		return nil, fmt.Errorf("environment path not found")
	}

	env, err := LoadFromYaml[domain.Environment](filePath)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (f *Filesystem) updateEnvironment(env *domain.Environment) error {
	filePath, exists := f.environmentPaths[env.MetaData.ID]
	if !exists {
		// This is a new environment
		fileName, err := f.getNewEnvironmentFilePath(env.MetaData.Name)
		if err != nil {
			return err
		}
		filePath = fileName.Path
		f.environmentPaths[env.MetaData.ID] = filePath
	}

	if err := SaveToYaml(filePath, env); err != nil {
		return err
	}

	// rename the file to the new name
	if env.MetaData.Name != filepath.Base(filePath) {
		newFilePath := filepath.Join(filepath.Dir(filePath), env.MetaData.Name+".yaml")
		if err := os.Rename(filePath, newFilePath); err != nil {
			return err
		}
		f.environmentPaths[env.MetaData.ID] = newFilePath
	}

	return nil
}

func (f *Filesystem) getNewEnvironmentFilePath(name string) (*FilePath, error) {
	dir, err := f.getEntityDirectoryInWorkspace(environmentsDir)
	if err != nil {
		return nil, err
	}

	return getNewFilePath(dir, name), nil
}

func (f *Filesystem) ReadPreferences() (*domain.Preferences, error) {
	pdir := filepath.Join(f.dataDir, f.ActiveWorkspace.MetaData.Name, preferencesDir)
	filePath := filepath.Join(pdir, "preferences.yaml")

	return LoadFromYaml[domain.Preferences](filePath)
}

func (f *Filesystem) LoadRequests() ([]*domain.Request, error) {
	dir, err := f.getEntityDirectoryInWorkspace(requestsDir)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]*domain.Request, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dir, file.Name())
		req, err := f.loadRequest(filePath)
		if err != nil {
			return nil, err
		}
		out = append(out, req)
	}

	return out, nil
}

func (f *Filesystem) loadRequest(filePath string) (*domain.Request, error) {
	req, err := LoadFromYaml[domain.Request](filePath)
	if err != nil {
		return nil, err
	}

	req.SetDefaultValues()
	f.requestPaths[req.MetaData.ID] = filePath
	return req, nil
}

func (f *Filesystem) GetRequest(id string) (*domain.Request, error) {
	filePath, exists := f.requestPaths[id]
	if !exists {
		return nil, fmt.Errorf("request file path not found")
	}

	req, err := LoadFromYaml[domain.Request](filePath)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (f *Filesystem) updateRequest(request *domain.Request) error {
	filePath, exists := f.requestPaths[request.MetaData.ID]
	if !exists {
		// This is a new request
		fileName, err := f.getNewRequestFilePath(request.MetaData.Name)
		if err != nil {
			return err
		}
		filePath = fileName.Path
		f.requestPaths[request.MetaData.ID] = filePath
	}

	if err := SaveToYaml(filePath, request); err != nil {
		return err
	}

	// rename the file to the new name
	if request.MetaData.Name != filepath.Base(filePath) {
		newFilePath := filepath.Join(filepath.Dir(filePath), request.MetaData.Name+".yaml")
		if err := os.Rename(filePath, newFilePath); err != nil {
			return err
		}
		f.requestPaths[request.MetaData.ID] = newFilePath
	}
	return nil
}

func (f *Filesystem) getNewRequestFilePath(name string) (*FilePath, error) {
	dir, err := f.getEntityDirectoryInWorkspace(requestsDir)
	if err != nil {
		return nil, err
	}
	return getNewFilePath(dir, name), nil
}

func getNewFilePath(dir, name string) *FilePath {
	fileName := filepath.Join(dir, name)
	fName := generateNewFileName(fileName, "yaml")

	return &FilePath{
		Path:    fName,
		NewName: getFileNameWithoutExt(fName),
	}
}

// generateNewFileName takes the original file name and generates a new file name
// with the first possible numeric postfix if the original file exists.
func generateNewFileName(filename, ext string) string {
	if !fileExists(filename + "." + ext) {
		return filename + "." + ext
	}

	// If the file exists, append a number to the filename.
	for i := 1; ; i++ {
		newFilename := fmt.Sprintf("%s%d.%s", filename, i, ext)
		if !fileExists(newFilename) {
			return newFilename
		}
	}
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func dirExist(dirname string) bool {
	info, err := os.Stat(dirname)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func (f *Filesystem) Create(entity interface{}) error {
	switch e := entity.(type) {
	case *domain.Request:
		return f.createRequest(e)
	case *domain.Collection:
		return f.createCollection(e)
	case *domain.Environment:
		return f.createEnvironment(e)
	case *domain.Workspace:
		return f.createWorkspace(e)
	case *domain.ProtoFile:
		return f.createProtoFile(e)
	default:
		return fmt.Errorf("unsupported entity type: %T", entity)
	}
}

func (f *Filesystem) CreateRequestInCollection(collection *domain.Collection, request *domain.Request) error {
	// Generate unique name if needed
	request.MetaData.Name = f.generateUniqueName(request.MetaData.Name)

	// Set collection metadata
	request.CollectionID = collection.MetaData.ID
	request.CollectionName = collection.MetaData.Name

	// Get collection directory path
	collectionDir := filepath.Dir(f.collectionPaths[collection.MetaData.ID])
	filePath := filepath.Join(collectionDir, request.MetaData.Name+".yaml")
	f.requestPaths[request.MetaData.ID] = filePath

	return f.updateRequest(request)
}

// Helper function to generate unique names
func (f *Filesystem) generateUniqueName(name string) string {
	// Start with the original name
	newName := name
	counter := 1

	// Keep trying new names until we find one that doesn't exist
	for {
		// Check if this name exists in various locations
		exists, err := f.nameExists(newName)
		if err != nil || !exists {
			break
		}

		// If it exists, try the next number
		newName = fmt.Sprintf("%s%d", name, counter)
		counter++
	}

	return newName
}

// Helper function to check if a name exists across different types
func (f *Filesystem) nameExists(name string) (bool, error) {
	// Get all directories we need to check
	reqDir, err := f.getEntityDirectoryInWorkspace(requestsDir)
	if err != nil {
		return false, err
	}

	cDir, err := f.getEntityDirectoryInWorkspace(collectionsDir)
	if err != nil {
		return false, err
	}

	envDir, err := f.getEntityDirectoryInWorkspace(environmentsDir)
	if err != nil {
		return false, err
	}

	// Check in requests directory
	if fileExists(filepath.Join(reqDir, name+".yaml")) {
		return true, nil
	}

	// Check in collections directory
	if dirExist(filepath.Join(cDir, name)) {
		return true, nil
	}

	// Check in environments directory
	if fileExists(filepath.Join(envDir, name+".yaml")) {
		return true, nil
	}

	return false, nil
}

func (f *Filesystem) createProtoFile(protoFile *domain.ProtoFile) error {
	// Get proto files directory
	protoDir, err := f.getEntityDirectoryInWorkspace(protoFilesDir)
	if err != nil {
		return err
	}

	// Generate file path internally
	filePath := filepath.Join(protoDir, protoFile.MetaData.Name+".yaml")
	f.protoFilePaths[protoFile.MetaData.ID] = filePath

	return f.updateProtoFile(protoFile)
}

func (f *Filesystem) createRequest(request *domain.Request) error {
	// Get requests directory
	reqDir, err := f.getEntityDirectoryInWorkspace(requestsDir)
	if err != nil {
		return err
	}

	// Generate file path internally
	filePath := filepath.Join(reqDir, request.MetaData.Name+".yaml")
	f.requestPaths[request.MetaData.ID] = filePath

	return f.updateRequest(request)
}

func (f *Filesystem) createCollection(collection *domain.Collection) error {
	// Get collections directory
	collectionDir, err := f.getEntityDirectoryInWorkspace(collectionsDir)
	if err != nil {
		return err
	}

	// Generate directory path internally
	dirPath := filepath.Join(collectionDir, collection.MetaData.Name)
	f.collectionPaths[collection.MetaData.ID] = filepath.Join(dirPath, "_collection.yaml")

	// Create the collection directory
	if err := util.MakeDir(dirPath); err != nil {
		return fmt.Errorf("failed to create collection directory: %w", err)
	}

	return f.updateCollection(collection)
}

func (f *Filesystem) createEnvironment(env *domain.Environment) error {
	// Get environments directory
	envDir, err := f.getEntityDirectoryInWorkspace(environmentsDir)
	if err != nil {
		return err
	}

	// Generate file path internally
	filePath := filepath.Join(envDir, env.MetaData.Name+".yaml")
	f.environmentPaths[env.MetaData.ID] = filePath

	return f.updateEnvironment(env)
}

func (f *Filesystem) createWorkspace(workspace *domain.Workspace) error {
	// Generate directory path internally
	dirPath := filepath.Join(f.dataDir, workspace.MetaData.Name)
	f.workspacePaths[workspace.MetaData.ID] = filepath.Join(dirPath, "_workspace.yaml")

	// Create the workspace directory
	if err := util.MakeDir(dirPath); err != nil {
		return fmt.Errorf("failed to create collection directory: %w", err)
	}

	return f.updateWorkspace(workspace)
}

func (f *Filesystem) Delete(entity interface{}) error {
	deleteFn := func(mp map[string]string, id string, isDir bool) error {
		filePath, exists := mp[id]
		if !exists {
			return fmt.Errorf("entiry path not found")
		}

		var err error
		if isDir {
			dirName := filepath.Dir(filePath)
			// If it's a directory, remove it and all its content
			err = os.RemoveAll(dirName)
		} else {
			// If it's a regular or symlink file, remove it
			err = os.Remove(filePath)
		}
		if err == nil {
			delete(mp, id)
		}
		return err
	}

	switch e := entity.(type) {
	case *domain.Request:
		return deleteFn(f.requestPaths, e.MetaData.ID, false)
	case *domain.Collection:
		return deleteFn(f.collectionPaths, e.MetaData.ID, true)
	case *domain.Environment:
		return deleteFn(f.environmentPaths, e.MetaData.ID, false)
	case *domain.Workspace:
		return deleteFn(f.workspacePaths, e.MetaData.ID, true)
	case *domain.ProtoFile:
		return deleteFn(f.protoFilePaths, e.MetaData.ID, false)
	default:
		return fmt.Errorf("unsupported entity type: %T", entity)
	}
}

func getFileNameWithoutExt(filePath string) string {
	_, file := filepath.Split(filePath)
	extension := filepath.Ext(file)
	return file[:len(file)-len(extension)]
}

func (f *Filesystem) Update(entity interface{}) error {
	switch e := entity.(type) {
	case *domain.Request:
		return f.updateRequest(e)
	case *domain.Collection:
		return f.updateCollection(e)
	case *domain.Environment:
		return f.updateEnvironment(e)
	case *domain.Workspace:
		return f.updateWorkspace(e)
	case *domain.ProtoFile:
		return f.updateProtoFile(e)
	default:
		return fmt.Errorf("unsupported entity type: %T", entity)
	}
}
