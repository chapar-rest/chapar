package domain

import (
	"github.com/google/uuid"
)

type Environment struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       EnvSpec  `yaml:"spec"`
	FilePath   string   `yaml:"-"`
}

type EnvSpec struct {
	Values []KeyValue `yaml:"values"`
}

func (e *EnvSpec) Clone() EnvSpec {
	clone := EnvSpec{
		Values: make([]KeyValue, len(e.Values)),
	}

	for i, v := range e.Values {
		clone.Values[i] = KeyValue{
			ID:     uuid.NewString(),
			Key:    v.Key,
			Value:  v.Value,
			Enable: v.Enable,
		}
	}

	return clone
}

func NewEnvironment(name string) *Environment {
	return &Environment{
		ApiVersion: ApiVersion,
		Kind:       KindEnv,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: name,
		},
		Spec: EnvSpec{
			Values: make([]KeyValue, 0),
		},
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
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: e.MetaData.Name,
		},
		Spec:     e.Spec.Clone(),
		FilePath: e.FilePath,
	}

	return clone
}

func (e *Environment) SetKey(key string, value string) {
	for i, v := range e.Spec.Values {
		if v.Key == key {
			e.Spec.Values[i].Value = value
			return
		}
	}

	e.Spec.Values = append(e.Spec.Values, KeyValue{
		ID:     uuid.NewString(),
		Key:    key,
		Value:  value,
		Enable: true,
	})
}
