package repository

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
)

type Repository interface {
	LoadCollections() ([]*domain.Collection, error)
	GetCollectionsDir() (string, error)
	UpdateCollection(collection *domain.Collection) error
	DeleteCollection(collection *domain.Collection) error

	LoadEnvironments() ([]*domain.Environment, error)
	GetEnvironment(filepath string) (*domain.Environment, error)
	GetEnvironmentDir() (string, error)
	UpdateEnvironment(env *domain.Environment) error
	DeleteEnvironment(env *domain.Environment) error

	ReadPreferencesData() (*domain.Preferences, error)
	UpdatePreferences(pref *domain.Preferences) error

	LoadRequests() ([]*domain.Request, error)
	GetRequest(filepath string) (*domain.Request, error)
	GetRequestsDir() (string, error)
	UpdateRequest(request *domain.Request) error
	DeleteRequest(request *domain.Request) error

	LoadWorkspaces() ([]*domain.Workspace, error)
	GetWorkspace(filepath string) (*domain.Workspace, error)
	GetWorkspacesDir() (string, error)
	UpdateWorkspace(workspace *domain.Workspace) error
	DeleteWorkspace(workspace *domain.Workspace) error

	GetProtoFilesDir() (string, error)
	LoadProtoFiles() ([]*domain.ProtoFile, error)
	DeleteProtoFile(protoFile *domain.ProtoFile) error
	UpdateProtoFile(protoFile *domain.ProtoFile) error
	CreateProtoFile(protoFile *domain.ProtoFile) error

	SetActiveWorkspace(workspace *domain.Workspace) error

	GetConfig() (*domain.Config, error)
	UpdateConfig(config *domain.Config) error

	CreateRequest(request *domain.Request) error
	CreateRequestInCollection(collection *domain.Collection, request *domain.Request) error
	CreateCollection(collection *domain.Collection) error
	CreateEnvironment(env *domain.Environment) error
	CreateWorkspace(workspace *domain.Workspace) error
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
