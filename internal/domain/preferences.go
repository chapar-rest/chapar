package domain

import "github.com/google/uuid"

type Preferences struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       PrefSpec `yaml:"spec"`
}

type PrefSpec struct {
	DarkMode            bool                `yaml:"darkMode"`
	SelectedEnvironment SelectedEnvironment `yaml:"selectedEnvironment"`
}

type SelectedEnvironment struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

func NewPreferences() *Preferences {
	return &Preferences{
		ApiVersion: ApiVersion,
		Kind:       KindPreferences,
		MetaData: MetaData{
			ID:   uuid.NewString(),
			Name: "Preferences",
		},
		Spec: PrefSpec{
			DarkMode:            true,
			SelectedEnvironment: SelectedEnvironment{},
		},
	}
}
