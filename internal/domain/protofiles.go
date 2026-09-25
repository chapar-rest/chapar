package domain

import (
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type ProtoFile struct {
	ApiVersion string        `yaml:"apiVersion"`
	Kind       string        `yaml:"kind"`
	MetaData   MetaData      `yaml:"metadata"`
	Spec       ProtoFileSpec `yaml:"spec"`
}

func NewProtoFile(name string) *ProtoFile {
	return &ProtoFile{
		ApiVersion: ApiVersion,
		Kind:       KindProtoFile,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: name,
		},
		Spec: ProtoFileSpec{
			Path: "",
		},
	}
}

func (p *ProtoFile) ID() string {
	return p.MetaData.ID
}

func (p *ProtoFile) GetKind() string {
	return p.Kind
}

func (p *ProtoFile) SetName(name string) {
	p.MetaData.Name = name
}

func (p *ProtoFile) GetName() string {
	return p.MetaData.Name
}

func (p *ProtoFile) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(p)
}

type ProtoFileSpec struct {
	Path string `yaml:"path"`
	// TODO should it be a dedicated type?
	IsImportPath bool     `yaml:"isImportPath"`
	Package      string   `yaml:"package"`
	Services     []string `yaml:"services"`
}
