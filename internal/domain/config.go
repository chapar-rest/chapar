package domain

type Config struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	MetaData   MetaData   `yaml:"metadata"`
	Spec       ConfigSpec `yaml:"spec"`
}

type ConfigSpec struct {
	ActiveWorkspace *ActiveWorkspace `yaml:"activeWorkspace"`
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
