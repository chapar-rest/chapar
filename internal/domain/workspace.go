package domain

import "github.com/google/uuid"

type Workspace struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	FilePath   string   `yaml:"-"`
}

func NewWorkspace(name string) *Workspace {
	return &Workspace{
		ApiVersion: ApiVersion,
		Kind:       KindWorkspace,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: name,
		},
		FilePath: "",
	}
}
