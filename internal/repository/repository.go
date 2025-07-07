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
	loadCollectionRequests(path string) ([]*domain.Request, error)
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
	writeWorkspace(workspace *domain.Workspace, override bool) error
	writeCollection(collection *domain.Collection, override bool) error
}

// Repository defines the main storage interface
type Repository interface {
	// Create operations for entities
	Create(entity interface{}) error
	// Update operations for entities
	Update(entity interface{}) error
	// Delete operations for entities
	Delete(entity interface{}) error

	// LoadCollections loads all collections
	LoadCollections() ([]*domain.Collection, error)
	// CreateRequestInCollection creates a request in a collection
	CreateRequestInCollection(collection *domain.Collection, request *domain.Request) error

	// LoadEnvironments loads all environments
	LoadEnvironments() ([]*domain.Environment, error)
	// GetEnvironment gets an environment by id
	GetEnvironment(id string) (*domain.Environment, error)

	// LoadRequests loads all requests
	LoadRequests() ([]*domain.Request, error)
	// GetRequest gets a request by id
	GetRequest(id string) (*domain.Request, error)

	// LoadWorkspaces loads all workspaces
	LoadWorkspaces() ([]*domain.Workspace, error)
	// GetWorkspace gets a workspace by id
	GetWorkspace(id string) (*domain.Workspace, error)
	// LoadProtoFiles loads all proto files
	LoadProtoFiles() ([]*domain.ProtoFile, error)
	// SetActiveWorkspace sets the active workspace
	SetActiveWorkspace(workspace *domain.Workspace)

	// GetConfig gets the config
	GetConfig() (*domain.Config, error)

	// ReadPreferences reads the preferences
	ReadPreferences() (*domain.Preferences, error)
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
