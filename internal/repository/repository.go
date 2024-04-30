package repository

import (
	"os"
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/domain"
	"gopkg.in/yaml.v2"
)

type Repository interface {
	LoadCollections() ([]*domain.Collection, error)
	GetCollectionsDir() (string, error)
	UpdateCollection(collection *domain.Collection) error
	DeleteCollection(collection *domain.Collection) error
	GetNewCollectionDir(name string) (*FilePath, error)
	GetCollectionRequestNewFilePath(collection *domain.Collection, name string) (*FilePath, error)

	LoadEnvironments() ([]*domain.Environment, error)
	GetEnvironment(filepath string) (*domain.Environment, error)
	GetEnvironmentDir() (string, error)
	UpdateEnvironment(env *domain.Environment) error
	DeleteEnvironment(env *domain.Environment) error
	GetNewEnvironmentFilePath(name string) (*FilePath, error)

	ReadPreferencesData() (*domain.Preferences, error)
	UpdatePreferences(pref *domain.Preferences) error

	LoadRequests() ([]*domain.Request, error)
	GetRequest(filepath string) (*domain.Request, error)
	GetRequestsDir() (string, error)
	UpdateRequest(request *domain.Request) error
	DeleteRequest(request *domain.Request) error
	GetNewRequestFilePath(name string) (*FilePath, error)

	LoadWorkspaces() ([]*domain.Workspace, error)
	GetWorkspace(filepath string) (*domain.Workspace, error)
	GetWorkspacesDir() (string, error)
	UpdateWorkspace(workspace *domain.Workspace) error
	DeleteWorkspace(workspace *domain.Workspace) error
	GetNewWorkspaceDir(name string) (*FilePath, error)

	GetConfig() (*domain.Config, error)
	UpdateConfig(config *domain.Config) error
}

type FilePath struct {
	Path    string
	NewName string
}

func LoadFromYaml[T any](filename string) (*T, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	env := new(T)
	if err := yaml.Unmarshal(data, env); err != nil {
		return nil, err
	}
	return env, nil
}

func SaveToYaml[T any](filename string, data *T) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, out, 0644); err != nil {
		return err
	}
	return nil
}

// AddSuffixBeforeExt to add a suffix before the file extension
func AddSuffixBeforeExt(filePath, suffix string) string {
	dir, file := filepath.Split(filePath)
	extension := filepath.Ext(file)
	baseName := file[:len(file)-len(extension)]
	newBaseName := baseName + suffix + extension
	return filepath.Join(dir, newBaseName)
}

func GetFileNameWithoutExt(filePath string) string {
	_, file := filepath.Split(filePath)
	extension := filepath.Ext(file)
	return file[:len(file)-len(extension)]
}
