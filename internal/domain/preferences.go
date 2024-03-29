package domain

import "github.com/google/uuid"

type Preferences struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       PrefSpec `yaml:"spec"`
}

type PrefSpec struct {
	DarkMode              bool   `yaml:"darkMode"`
	SelectedEnvironmentID string `yaml:"selectedEnvironmentID"`
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
			DarkMode:              true,
			SelectedEnvironmentID: "",
		},
	}
}
