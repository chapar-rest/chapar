package repository

import (
	"os"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
)

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
