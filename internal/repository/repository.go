package repository

import (
	"os"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
)

// loadFromYaml reads and unmarshals a YAML file via the given StorageBackend.
func loadFromYaml[T any](backend StorageBackend, path string) (*T, error) {
	data, err := backend.ReadFile(path)
	if err != nil {
		return nil, err
	}
	v := new(T)
	if err := yaml.Unmarshal(data, v); err != nil {
		return nil, err
	}
	return v, nil
}

// saveToYaml marshals data to YAML and writes it via the given StorageBackend.
func saveToYaml[T any](backend StorageBackend, path string, data *T) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return backend.WriteFile(path, out)
}

// RepositoryV2 defines the main storage interface
type RepositoryV2 interface {
	SetActiveWorkspace(workspaceName string)

	LoadProtoFiles() ([]*domain.ProtoFile, error)
	CreateProtoFile(protoFile *domain.ProtoFile) error
	UpdateProtoFile(protoFile *domain.ProtoFile) error
	DeleteProtoFile(protoFile *domain.ProtoFile) error

	LoadRequests() ([]*domain.Request, error)
	CreateRequest(request *domain.Request, collection *domain.Collection) error
	UpdateRequest(request *domain.Request, collection *domain.Collection) error
	DeleteRequest(request *domain.Request, collection *domain.Collection) error

	LoadCollections() ([]*domain.Collection, error)
	CreateCollection(collection *domain.Collection) error
	UpdateCollection(collection *domain.Collection) error
	DeleteCollection(collection *domain.Collection) error

	LoadEnvironments() ([]*domain.Environment, error)
	CreateEnvironment(environment *domain.Environment) error
	UpdateEnvironment(environment *domain.Environment) error
	DeleteEnvironment(environment *domain.Environment) error

	LoadWorkspaces() ([]*domain.Workspace, error)
	CreateWorkspace(workspace *domain.Workspace) error
	UpdateWorkspace(workspace *domain.Workspace) error
	DeleteWorkspace(workspace *domain.Workspace) error
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
