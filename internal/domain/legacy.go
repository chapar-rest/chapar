package domain

import "gopkg.in/yaml.v2"

type Config struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	MetaData   MetaData   `yaml:"metadata"`
	Spec       ConfigSpec `yaml:"spec"`
}

func (c *Config) SetName(name string) {
	c.MetaData.Name = name
}

func (c *Config) GetName() string {
	return c.MetaData.Name
}

// ConfigSpec holds the configuration for the application.
// its deprecated and its data will be moved to GlobalConfig on the first run
type ConfigSpec struct {
	ActiveWorkspace     *ActiveWorkspace     `yaml:"activeWorkspace"`
	SelectedEnvironment *SelectedEnvironment `yaml:"selectedEnvironment"`
}

type ActiveWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewConfig() *Config {
	return &Config{
		ApiVersion: ApiVersion,
		Kind:       KindConfig,
		MetaData: MetaData{
			Name: "config",
		},
		Spec: ConfigSpec{
			ActiveWorkspace: &ActiveWorkspace{
				ID:   "default",
				Name: "Default Workspace",
			},
		},
	}
}

// Preferences struct is deprecated and will be removed in the future.
// Its deprecated and its data will be moved to AppState on the first run
type Preferences struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       PrefSpec `yaml:"spec"`
}

func (c *Preferences) ID() string {
	return c.MetaData.ID
}

func (c *Preferences) GetKind() string {
	return c.Kind
}

func (c *Preferences) SetName(name string) {
	c.MetaData.Name = name
}

func (c *Preferences) GetName() string {
	return c.MetaData.Name
}

func (c *Preferences) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(c)
}

type PrefSpec struct {
	DarkMode            bool                `yaml:"darkMode"`
	SelectedEnvironment SelectedEnvironment `yaml:"selectedEnvironment"`
}

type SelectedEnvironment struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}
