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

func CompareProtoFiles(a, b *ProtoFile) bool {
	return a.MetaData.ID == b.MetaData.ID &&
		a.MetaData.Name == b.MetaData.Name &&
		CompareProtoFileSpecs(a.Spec, b.Spec)
}

func CompareProtoFileSpecs(a, b ProtoFileSpec) bool {
	return a.Path == b.Path &&
		a.Package == b.Package &&
		a.IsImportPath == b.IsImportPath &&
		compareStringSlices(a.Services, b.Services)
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}
