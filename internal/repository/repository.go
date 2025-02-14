package repository

import (
	"os"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
)

// Repository defines the main storage interface
type Repository interface {
	// Generic CRUD operations
	Create(entity interface{}) error
	Update(entity interface{}) error
	Delete(entity interface{}) error

	// Collection operations
	LoadCollections() ([]*domain.Collection, error)
	CreateRequestInCollection(collection *domain.Collection, request *domain.Request) error

	// Environment operations
	LoadEnvironments() ([]*domain.Environment, error)
	GetEnvironment(id string) (*domain.Environment, error)

	// Request operations
	LoadRequests() ([]*domain.Request, error)
	GetRequest(id string) (*domain.Request, error)

	// Workspace operations
	LoadWorkspaces() ([]*domain.Workspace, error)
	GetWorkspace(id string) (*domain.Workspace, error)
	SetActiveWorkspace(workspace *domain.Workspace) error

	// ProtoFile operations
	LoadProtoFiles() ([]*domain.ProtoFile, error)

	// Configuration
	GetConfig() (*domain.Config, error)
	UpdateConfig(config *domain.Config) error

	// Preferences
	ReadPreferences() (*domain.Preferences, error)
	UpdatePreferences(pref *domain.Preferences) error
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
