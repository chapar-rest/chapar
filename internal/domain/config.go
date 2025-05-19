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
	AutoCloseBrackets bool   `yaml:"autoCloseBrackets"`
	AutoCloseQuotes   bool   `yaml:"autoCloseQuotes"`
}

func (e EditorConfig) Changed(other EditorConfig) bool {
	return e.FontFamily != other.FontFamily ||
		e.FontSize != other.FontSize ||
		e.Indentation != other.Indentation ||
		e.TabWidth != other.TabWidth ||
		e.AutoCloseBrackets != other.AutoCloseBrackets ||
		e.AutoCloseQuotes != other.AutoCloseQuotes
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

// GetDefaultGlobalConfig returns a default global config
func GetDefaultGlobalConfig() *GlobalConfig {
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
				AutoCloseBrackets: true,
				AutoCloseQuotes:   true,
			},
			Scripting: ScriptingConfig{
				Enabled:     false,
				UseDocker:   true,
				DockerImage: "chapar/python-executor:latest",
				Language:    "python",
				Port:        2397,
			},
			Data: DataConfig{
				WorkspacePath: "",
			},
		},
	}
}

func (g *GlobalConfig) ValuesMap() map[string]any {
	return map[string]any{
		"general": map[string]any{
			"httpVersion":           g.Spec.General.HTTPVersion,
			"timeoutSec":            g.Spec.General.RequestTimeoutSec,
			"responseSizeMb":        g.Spec.General.ResponseSizeMb,
			"sendNoCacheHeader":     g.Spec.General.SendNoCacheHeader,
			"sendChaparAgentHeader": g.Spec.General.SendChaparAgentHeader,
		},
		"editor": map[string]any{
			"fontFamily":        g.Spec.Editor.FontFamily,
			"fontSize":          g.Spec.Editor.FontSize,
			"indentation":       g.Spec.Editor.Indentation,
			"tabWidth":          g.Spec.Editor.TabWidth,
			"autoCloseBrackets": g.Spec.Editor.AutoCloseBrackets,
			"autoCloseQuotes":   g.Spec.Editor.AutoCloseQuotes,
		},
		"scripting": map[string]any{
			"enabled":          g.Spec.Scripting.Enabled,
			"language":         g.Spec.Scripting.Language,
			"useDocker":        g.Spec.Scripting.UseDocker,
			"dockerImage":      g.Spec.Scripting.DockerImage,
			"executablePath":   g.Spec.Scripting.ExecutablePath,
			"serverScriptPath": g.Spec.Scripting.ServerScriptPath,
			"port":             g.Spec.Scripting.Port,
		},
		"data": map[string]any{
			"workspacePath": g.Spec.Data.WorkspacePath,
		},
	}
}

// GetDefaultAppState returns a default app state
func GetDefaultAppState() *AppState {
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
