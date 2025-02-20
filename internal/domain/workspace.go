package domain

import (
	"github.com/google/uuid"
)

const DefaultWorkspaceName = "Default Workspace"

type Workspace struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
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
