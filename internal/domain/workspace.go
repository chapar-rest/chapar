package domain

import (
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

const DefaultWorkspaceName = "Default Workspace"

type Workspace struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
}

func (w *Workspace) ID() string {
	return w.MetaData.ID
}

func (w *Workspace) GetKind() string {
	return w.Kind
}

func (w *Workspace) SetName(name string) {
	w.MetaData.Name = name
}

func (w *Workspace) GetName() string {
	return w.MetaData.Name
}

func (w *Workspace) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(w)
}

func NewWorkspace(name string) *Workspace {
	return &Workspace{
		ApiVersion: ApiVersion,
		Kind:       KindWorkspace,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: name,
		},
	}
}

func NewDefaultWorkspace() *Workspace {
	return &Workspace{
		ApiVersion: ApiVersion,
		Kind:       KindWorkspace,
		MetaData: MetaData{
			ID:   "default",
			Name: DefaultWorkspaceName,
		},
	}
}
