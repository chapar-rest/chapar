package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
)

const (
	configDir = "chapar"

	environmentsDir = "envs"
	protoFilesDir   = "protofiles"
	collectionsDir  = "collections"
	requestsDir     = "requests"
	preferencesDir  = "preferences"
)

var _ Repository = &Filesystem{}

type Filesystem struct {
	ActiveWorkspace *domain.Workspace
}

func NewFilesystem() (*Filesystem, error) {
	fs := &Filesystem{}
	config, err := fs.GetConfig()
	if err != nil {
		return nil, err
	}

	cDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	if config.Spec.ActiveWorkspace != nil {
		ws, err := fs.GetWorkspace(filepath.Join(cDir, config.Spec.ActiveWorkspace.Name))
		if err != nil {
			return nil, err
		}
		fs.ActiveWorkspace = ws
	}

	// if there is no active workspace, create default workspace
	if fs.ActiveWorkspace == nil {
		ws := domain.NewDefaultWorkspace()
		ws.FilePath = filepath.Join(cDir, "default")
		if err := fs.UpdateWorkspace(ws); err != nil {
			return nil, err
		}

		fs.ActiveWorkspace = ws
	}

	return fs, nil
}

func (f *Filesystem) GetProtoFilesDir() (string, error) {
	dir, err := CreateConfigDir()
	if err != nil {
		return "", err
	}

	protoDir := filepath.Join(dir, f.ActiveWorkspace.MetaData.Name, protoFilesDir)
	if err := makeDir(protoDir); err != nil {
		return "", err
	}

	return protoDir, nil
}

func (f *Filesystem) LoadProtoFiles() ([]*domain.ProtoFile, error) {
	dir, err := f.GetProtoFilesDir()
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
		protoFile.FilePath = filePath
		out = append(out, protoFile)
	}

	return out, err
}

func (f *Filesystem) DeleteProtoFile(protoFile *domain.ProtoFile) error {
	return os.Remove(protoFile.FilePath)
}

func (f *Filesystem) UpdateProtoFile(protoFile *domain.ProtoFile) error {
	if protoFile.FilePath == "" {
		// this is a new protoFile
		fileName, err := f.GetNewProtoFilePath(protoFile.MetaData.Name)
		if err != nil {
			return err
		}

		protoFile.FilePath = fileName.Path
	}

	if err := SaveToYaml(protoFile.FilePath, protoFile); err != nil {
		return err
	}

	// rename the file to the new name
	if protoFile.MetaData.Name != filepath.Base(protoFile.FilePath) {
		newFilePath := filepath.Join(filepath.Dir(protoFile.FilePath), protoFile.MetaData.Name+".yaml")
		if err := os.Rename(protoFile.FilePath, newFilePath); err != nil {
			return err
		}
		protoFile.FilePath = newFilePath
	}
	return nil
}

func (f *Filesystem) GetNewProtoFilePath(name string) (*FilePath, error) {
	dir, err := f.GetProtoFilesDir()
	if err != nil {
		return nil, err
	}

	return getNewFilePath(dir, name), nil
}

func (f *Filesystem) SetActiveWorkspace(workspace *domain.Workspace) error {
	config, err := f.GetConfig()
	if err != nil {
		return err
	}

	f.ActiveWorkspace = workspace
	config.Spec.ActiveWorkspace = &domain.ActiveWorkspace{
		ID:   workspace.MetaData.ID,
		Name: workspace.MetaData.Name,
	}
	return f.UpdateConfig(config)
}

func (f *Filesystem) GetConfig() (*domain.Config, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(dir, "config.yaml")

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

func (f *Filesystem) UpdateConfig(config *domain.Config) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(dir, "config.yaml")
	return SaveToYaml(filePath, config)
}

func (f *Filesystem) LoadWorkspaces() ([]*domain.Workspace, error) {
	wdir, err := f.GetWorkspacesDir()
	if err != nil {
		return nil, err
	}

	dirs, err := os.ReadDir(wdir)
	if err != nil {
		return nil, err
	}

	out := make([]*domain.Workspace, 0)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		dirPath := filepath.Join(wdir, dir.Name())
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
		ws.FilePath = filePath
		if err := SaveToYaml(filePath, ws); err != nil {
			return nil, err
		}

		return ws, nil
	}

	ws, err := LoadFromYaml[domain.Workspace](filePath)
	if err != nil {
		return nil, err
	}

	ws.FilePath = filePath
	return ws, nil
}

func (f *Filesystem) GetWorkspacesDir() (string, error) {
	dir, err := CreateConfigDir()
	if err != nil {
		return "", err
	}

	// all folders in the config directory are workspaces
	return dir, nil
}

func (f *Filesystem) UpdateWorkspace(workspace *domain.Workspace) error {
	if !strings.HasSuffix(workspace.FilePath, "_workspace.yaml") {
		// if directory is not exist, create it
		if _, err := os.Stat(workspace.FilePath); os.IsNotExist(err) {
			if err := os.MkdirAll(workspace.FilePath, 0755); err != nil {
				return err
			}
		}

		workspace.FilePath = filepath.Join(workspace.FilePath, "_workspace.yaml")
	}

	if err := SaveToYaml(workspace.FilePath, workspace); err != nil {
		return err
	}

	// Get the directory name
	dirName := filepath.Dir(workspace.FilePath)
	// Change the directory name to the collection name
	if workspace.MetaData.Name != filepath.Base(dirName) {
		// replace last part of the path with the new name
		newDirName := filepath.Join(filepath.Dir(dirName), workspace.MetaData.Name)
		if err := os.Rename(dirName, newDirName); err != nil {
			return err
		}
		workspace.FilePath = filepath.Join(newDirName, "_workspace.yaml")
	}

	return nil
}

func (f *Filesystem) DeleteWorkspace(workspace *domain.Workspace) error {
	return os.RemoveAll(filepath.Dir(workspace.FilePath))
}

func (f *Filesystem) GetNewWorkspaceDir(name string) (*FilePath, error) {
	wDir, err := f.GetWorkspacesDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(wDir, name)
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
	dir := filepath.Dir(collection.FilePath)
	return getNewFilePath(dir, name), nil
}

func (f *Filesystem) LoadCollections() ([]*domain.Collection, error) {
	dir, err := f.GetCollectionsDir()
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

		// Skip further processing since we're only interested in directories here
		return filepath.SkipDir
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
		req, err := LoadFromYaml[domain.Request](requestPath)
		if err != nil {
			return nil, err
		}

		// set request default values
		req.SetDefaultValues()

		req.FilePath = requestPath
		req.CollectionName = collection.MetaData.Name
		collection.Spec.Requests = append(collection.Spec.Requests, req)
	}
	return collection, nil
}

func (f *Filesystem) GetCollectionsDir() (string, error) {
	dir, err := CreateConfigDir()
	if err != nil {
		return "", err
	}

	cdir := filepath.Join(dir, f.ActiveWorkspace.MetaData.Name, collectionsDir)
	if err := makeDir(cdir); err != nil {
		return "", err
	}

	return cdir, nil
}

func (f *Filesystem) UpdateCollection(collection *domain.Collection) error {
	if !strings.HasSuffix(collection.FilePath, "_collection.yaml") {
		// if directory is not exist, create it
		if _, err := os.Stat(collection.FilePath); os.IsNotExist(err) {
			if err := os.MkdirAll(collection.FilePath, 0755); err != nil {
				return err
			}
		}

		collection.FilePath = filepath.Join(collection.FilePath, "_collection.yaml")
	}

	if err := SaveToYaml(collection.FilePath, collection); err != nil {
		return err
	}

	// Get the directory name
	dirName := filepath.Dir(collection.FilePath)
	// Change the directory name to the collection name
	if collection.MetaData.Name != filepath.Base(dirName) {
		// replace last part of the path with the new name
		newDirName := filepath.Join(filepath.Dir(dirName), collection.MetaData.Name)
		if err := os.Rename(dirName, newDirName); err != nil {
			return err
		}
		collection.FilePath = filepath.Join(newDirName, "_collection.yaml")
	}

	return nil
}

func (f *Filesystem) DeleteCollection(collection *domain.Collection) error {
	return os.RemoveAll(filepath.Dir(collection.FilePath))
}

func (f *Filesystem) GetNewCollectionDir(name string) (*FilePath, error) {
	collectionDir, err := f.GetCollectionsDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(collectionDir, name)
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

func (f *Filesystem) LoadEnvironments() ([]*domain.Environment, error) {
	dir, err := f.GetEnvironmentDir()
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
		env.FilePath = filePath
		out = append(out, env)
	}

	return out, nil
}

func (f *Filesystem) GetEnvironment(filepath string) (*domain.Environment, error) {
	env, err := LoadFromYaml[domain.Environment](filepath)
	if err != nil {
		return nil, err
	}

	env.FilePath = filepath
	return env, nil
}

func (f *Filesystem) GetEnvironmentDir() (string, error) {
	dir, err := CreateConfigDir()
	if err != nil {
		return "", err
	}

	envDir := filepath.Join(dir, f.ActiveWorkspace.MetaData.Name, environmentsDir)
	if err := makeDir(envDir); err != nil {
		return "", err
	}

	return envDir, nil
}

func (f *Filesystem) UpdateEnvironment(env *domain.Environment) error {
	if err := SaveToYaml(env.FilePath, env); err != nil {
		return err
	}

	// rename the file to the new name
	if env.MetaData.Name != filepath.Base(env.FilePath) {
		newFilePath := filepath.Join(filepath.Dir(env.FilePath), env.MetaData.Name+".yaml")
		if err := os.Rename(env.FilePath, newFilePath); err != nil {
			return err
		}
		env.FilePath = newFilePath
	}

	return nil
}

func (f *Filesystem) GetNewEnvironmentFilePath(name string) (*FilePath, error) {
	dir, err := f.GetEnvironmentDir()
	if err != nil {
		return nil, err
	}

	return getNewFilePath(dir, name), nil
}

func (f *Filesystem) DeleteEnvironment(env *domain.Environment) error {
	return os.Remove(env.FilePath)
}

func (f *Filesystem) ReadPreferencesData() (*domain.Preferences, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	pdir := filepath.Join(dir, f.ActiveWorkspace.MetaData.Name, preferencesDir)
	filePath := filepath.Join(pdir, "preferences.yaml")
	return LoadFromYaml[domain.Preferences](filePath)
}

func (f *Filesystem) UpdatePreferences(pref *domain.Preferences) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	pdir := filepath.Join(dir, f.ActiveWorkspace.MetaData.Name, preferencesDir)
	if err := makeDir(pdir); err != nil {
		return err
	}

	filePath := filepath.Join(pdir, "preferences.yaml")
	return SaveToYaml[domain.Preferences](filePath, pref)
}

func (f *Filesystem) LoadRequests() ([]*domain.Request, error) {
	dir, err := f.GetRequestsDir()
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

	req.FilePath = filePath
	return req, nil
}

func (f *Filesystem) GetRequest(filepath string) (*domain.Request, error) {
	req, err := LoadFromYaml[domain.Request](filepath)
	if err != nil {
		return nil, err
	}

	req.FilePath = filepath
	return req, nil
}

func (f *Filesystem) GetRequestsDir() (string, error) {
	dir, err := CreateConfigDir()
	if err != nil {
		return "", err
	}

	rdir := filepath.Join(dir, f.ActiveWorkspace.MetaData.Name, requestsDir)
	if err := makeDir(rdir); err != nil {
		return "", err
	}

	return rdir, nil
}

func (f *Filesystem) UpdateRequest(request *domain.Request) error {
	if request.FilePath == "" {
		// this is a new request
		fileName, err := f.GetNewRequestFilePath(request.MetaData.Name)
		if err != nil {
			return err
		}

		request.FilePath = fileName.Path
	}

	if err := SaveToYaml(request.FilePath, request); err != nil {
		return err
	}

	// rename the file to the new name
	if request.MetaData.Name != filepath.Base(request.FilePath) {
		newFilePath := filepath.Join(filepath.Dir(request.FilePath), request.MetaData.Name+".yaml")
		if err := os.Rename(request.FilePath, newFilePath); err != nil {
			return err
		}
		request.FilePath = newFilePath
	}
	return nil
}

func (f *Filesystem) GetNewRequestFilePath(name string) (*FilePath, error) {
	dir, err := f.GetRequestsDir()
	if err != nil {
		return nil, err
	}
	return getNewFilePath(dir, name), nil
}

func (f *Filesystem) DeleteRequest(request *domain.Request) error {
	return os.Remove(request.FilePath)
}

func getNewFilePath(dir, name string) *FilePath {
	fileName := filepath.Join(dir, name)
	fName := generateNewFileName(fileName, "yaml")

	return &FilePath{
		Path:    fName,
		NewName: GetFileNameWithoutExt(fName),
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

func GetConfigDir() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, configDir)
	return path, makeDir(path)
}

func CreateConfigDir() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	if err := makeDir(dir); err != nil {
		return "", err
	}

	return dir, nil
}

func makeDir(dir string) error {
	dir = filepath.FromSlash(dir)
	fnMakeDir := func() error { return os.MkdirAll(dir, os.ModePerm) }
	info, err := os.Stat(dir)
	switch {
	case err == nil:
		if info.IsDir() {
			return nil // The directory exists
		} else {
			return fmt.Errorf("path exists but is not a directory: %s", dir)
		}
	case os.IsNotExist(err):
		return fnMakeDir()
	default:
		return err
	}
}

func userConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("AppData")
		if dir == "" {
			return "", errors.New("%AppData% is not defined")
		}

	case "plan9":
		dir = os.Getenv("home")
		if dir == "" {
			return "", errors.New("$home is not defined")
		}
		dir += "/lib"

	default: // Unix
		dir = os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
			}
			dir += "/.config"
		}
	}

	return dir, nil
}
