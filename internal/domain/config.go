package domain

type Config struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	MetaData   MetaData   `yaml:"metadata"`
	Spec       ConfigSpec `yaml:"spec"`
}

type ConfigSpec struct {
	ActiveWorkspace   *ActiveWorkspace   `yaml:"activeWorkspace"`
	ActiveEnvironment *ActiveEnvironment `yaml:"activeEnvironment"`
	Scripting         Scripting          `yaml:"scripting"`
}

type ActiveWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ActiveEnvironment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Scripting struct {
	Plugins []ScriptingPlugin `yaml:"plugins"`
}

type ScriptingPlugin struct {
	Language string `yaml:"language"`

	BinPath              string   `yaml:"binPath"`
	ServerScriptPathPath string   `yaml:"scriptPath"`
	Args                 []string `yaml:"args"`
	ServerPort           int      `yaml:"serverPort"`
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
			Scripting: Scripting{
				Plugins: []ScriptingPlugin{
					{
						BinPath:              "/Users/mohsen/.chapar/.venv/bin/python",
						ServerScriptPathPath: "/Users/mohsen/.chapar/scripts/chapar.py",
						Args: []string{
							"/Users/mohsen/.chapar/scripts/chapar.py",
							"--port", "8090",
						},
						ServerPort: 8090,
					},
				},
			},
		},
	}
}
