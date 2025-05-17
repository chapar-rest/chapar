package domain

import "github.com/google/uuid"

// GlobalConfig hold the global configuration for the application.
// it will eventually replace the Config struct
type GlobalConfig struct {
	ApiVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	MetaData   MetaData         `yaml:"metadata"`
	Spec       GlobalConfigSpec `yaml:"spec"`
}

type GlobalConfigSpec struct {
	General   GeneralConfig   `yaml:"general"`
	Editor    EditorConfig    `yaml:"editor"`
	Scripting ScriptingConfig `yaml:"scripting"`
	Data      DataConfig      `yaml:"data"`
}

type GeneralConfig struct {
	HTTPVersion           string `yaml:"httpVersion"`
	RequestTimeoutSec     int    `yaml:"timeoutSec"`
	ResponseSizeMb        int    `yaml:"responseSizeMb"`
	SendNoCacheHeader     bool   `yaml:"sendNoCacheHeader"`
	SendChaparAgentHeader bool   `yaml:"sendChaparAgentHeader"`
}

const (
	IndentationSpaces = "spaces"
	IndentationTabs   = "tabs"
)

type EditorConfig struct {
	FontFamily        string `yaml:"fontFamily"`
	FontSize          int    `yaml:"FontSize"`
	Indentation       string `yaml:"indentation"` // spaces or tabs
	TabWidth          int    `yaml:"tabWidth"`
	ShowLineNumbers   bool   `yaml:"showLineNumbers"`
	AutoCloseBrackets bool   `yaml:"autoCloseBrackets"`
	AutoCloseQuotes   bool   `yaml:"autoCloseQuotes"`
}

type ScriptingConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Language  string `yaml:"language"` // python or javascript
	UseDocker bool   `yaml:"useDocker"`
	// DockerImage is the docker image to use for the scripting engine
	DockerImage string `yaml:"dockerImage"`

	ExecutablePath   string `yaml:"executablePath"`
	ServerScriptPath string `yaml:"serverScriptPath"`
	Port             int    `yaml:"port"`
}

type DataConfig struct {
	WorkspacePath string `yaml:"workspacePath"`
}

type AppState struct {
	ApiVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	MetaData   MetaData     `yaml:"metadata"`
	Spec       AppStateSpec `yaml:"spec"`
}

type AppStateSpec struct {
	ActiveWorkspace     *ActiveWorkspace     `yaml:"activeWorkspace"`
	SelectedEnvironment *SelectedEnvironment `yaml:"selectedEnvironment"`
	DarkMode            bool                 `yaml:"darkMode"`
}

// getDefaultGlobalConfig returns a default global config
func NewGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		ApiVersion: ApiVersion,
		Kind:       "GlobalConfig",
		MetaData: MetaData{
			Name: "global-config",
		},
		Spec: GlobalConfigSpec{
			General: GeneralConfig{
				HTTPVersion:           "HTTP/1.1",
				RequestTimeoutSec:     30,
				ResponseSizeMb:        10,
				SendNoCacheHeader:     true,
				SendChaparAgentHeader: true,
			},
			Editor: EditorConfig{
				FontFamily:        "JetBrains Mono",
				FontSize:          12,
				Indentation:       IndentationSpaces,
				TabWidth:          4,
				ShowLineNumbers:   true,
				AutoCloseBrackets: true,
				AutoCloseQuotes:   true,
			},
			Scripting: ScriptingConfig{
				Enabled:  false,
				Language: "javascript",
				Port:     9090,
			},
			Data: DataConfig{
				WorkspacePath: "",
			},
		},
	}
}

// getDefaultAppState returns a default app state
func getDefaultAppState() *AppState {
	return &AppState{
		ApiVersion: ApiVersion,
		Kind:       "AppState",
		MetaData: MetaData{
			Name: "app-state",
		},
		Spec: AppStateSpec{
			ActiveWorkspace: &ActiveWorkspace{
				ID:   "default",
				Name: "Default Workspace",
			},
			SelectedEnvironment: &SelectedEnvironment{},
		},
	}
}

type Config struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	MetaData   MetaData   `yaml:"metadata"`
	Spec       ConfigSpec `yaml:"spec"`
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
