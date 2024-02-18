package domain

import "github.com/google/uuid"

const EnvKind = "Environment"

type Environment struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Meta       EnvMeta    `yaml:"meta"`
	Values     []KeyValue `yaml:"values"`

	FilePath string `yaml:"-"`
}

type EnvMeta struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

func NewEnvironment(name string) *Environment {
	return &Environment{
		ApiVersion: ApiVersion,
		Kind:       EnvKind,
		Meta: EnvMeta{
			ID:   uuid.NewString(),
			Name: name,
		},
		Values:   make([]KeyValue, 0),
		FilePath: "",
	}
}

func CompareEnvValue(a, b KeyValue) bool {
	// compare length of the values
	if len(a.Key) != len(b.Key) || len(a.Value) != len(b.Value) || len(a.ID) != len(b.ID) {
		return false
	}

	if a.Key != b.Key || a.Value != b.Value || a.Enable != b.Enable || a.ID != b.ID {
		return false
	}

	return true
}

func (e *Environment) Clone() *Environment {
	clone := &Environment{
		ApiVersion: e.ApiVersion,
		Kind:       e.Kind,
		Meta:       e.Meta,
		Values:     make([]KeyValue, len(e.Values)),
		FilePath:   e.FilePath,
	}

	for i, v := range e.Values {
		clone.Values[i] = v
	}

	return clone
}

func CompareEnvValues(a, b []KeyValue) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if !CompareEnvValue(v, b[i]) {
			return false
		}
	}

	return true
}
