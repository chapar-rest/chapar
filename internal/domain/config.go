package domain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/util"
)

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

func (g *GlobalConfig) Changed(other *GlobalConfig) bool {
	return g.Spec.General.Changed(other.Spec.General) ||
		g.Spec.Editor.Changed(other.Spec.Editor) ||
		g.Spec.Scripting.Changed(other.Spec.Scripting) ||
		g.Spec.Data.Changed(other.Spec.Data)
}

type GeneralConfig struct {
	HTTPVersion           string `yaml:"httpVersion"`
	RequestTimeoutSec     int    `yaml:"timeoutSec"`
	ResponseSizeMb        int    `yaml:"responseSizeMb"`
	SendNoCacheHeader     bool   `yaml:"sendNoCacheHeader"`
	SendChaparAgentHeader bool   `yaml:"sendChaparAgentHeader"`
}

func (g GeneralConfig) Changed(other GeneralConfig) bool {
	return g.HTTPVersion != other.HTTPVersion ||
		g.RequestTimeoutSec != other.RequestTimeoutSec ||
		g.ResponseSizeMb != other.ResponseSizeMb ||
		g.SendNoCacheHeader != other.SendNoCacheHeader ||
		g.SendChaparAgentHeader != other.SendChaparAgentHeader
}

const (
	IndentationSpaces = "spaces"
	IndentationTabs   = "tabs"
)

type EditorConfig struct {
	FontFamily        string `yaml:"fontFamily"`
	FontSize          int    `yaml:"FontSize"`
	Indentation       string `yaml:"indentation"` // spaces or tabs
	Theme             string `yaml:"theme"`       // theme name, e.q dracula, github, etc.
	TabWidth          int    `yaml:"tabWidth"`
	AutoCloseBrackets bool   `yaml:"autoCloseBrackets"`
	AutoCloseQuotes   bool   `yaml:"autoCloseQuotes"`
}

func (e EditorConfig) Changed(other EditorConfig) bool {
	return e.FontFamily != other.FontFamily ||
		e.FontSize != other.FontSize ||
		e.Indentation != other.Indentation ||
		e.Theme != other.Theme ||
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

func (s ScriptingConfig) Changed(other ScriptingConfig) bool {
	return s.Enabled != other.Enabled ||
		s.Language != other.Language ||
		s.UseDocker != other.UseDocker ||
		s.DockerImage != other.DockerImage ||
		s.ExecutablePath != other.ExecutablePath ||
		s.ServerScriptPath != other.ServerScriptPath ||
		s.Port != other.Port
}

type DataConfig struct {
	WorkspacePath string `yaml:"workspacePath"`
}

func (d DataConfig) Changed(other DataConfig) bool {
	return d.WorkspacePath != other.WorkspacePath
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
	dataDir, _ := LegacyConfigDir()

	return &GlobalConfig{
		ApiVersion: ApiVersion,
		Kind:       "GlobalConfig",
		MetaData: MetaData{
			Name: "global-config",
		},
		Spec: GlobalConfigSpec{
			General: GeneralConfig{
				HTTPVersion:           "http/1.1",
				RequestTimeoutSec:     30,
				ResponseSizeMb:        10,
				SendNoCacheHeader:     true,
				SendChaparAgentHeader: true,
			},
			Editor: EditorConfig{
				FontFamily:        "JetBrains Mono",
				FontSize:          12,
				Indentation:       IndentationSpaces,
				Theme:             "dracula",
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
				WorkspacePath: dataDir,
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
			"theme":             g.Spec.Editor.Theme,
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

func GlobalConfigFromValues(initial GlobalConfig, values map[string]any) GlobalConfig {
	if values == nil {
		fmt.Println("values is nil")
		return initial
	}

	g := initial

	g.Spec.General.HTTPVersion = getOrDefault(values, "httpVersion", g.Spec.General.HTTPVersion).(string)
	g.Spec.General.RequestTimeoutSec = getOrDefault(values, "requestTimeoutSec", g.Spec.General.RequestTimeoutSec).(int)
	g.Spec.General.ResponseSizeMb = getOrDefault(values, "responseSizeMb", g.Spec.General.ResponseSizeMb).(int)
	g.Spec.General.SendNoCacheHeader = getOrDefault(values, "sendNoCacheHeader", g.Spec.General.SendNoCacheHeader).(bool)
	g.Spec.General.SendChaparAgentHeader = getOrDefault(values, "sendChaparAgentHeader", g.Spec.General.SendChaparAgentHeader).(bool)

	g.Spec.Editor.FontFamily = getOrDefault(values, "fontFamily", g.Spec.Editor.FontFamily).(string)
	g.Spec.Editor.FontSize = getOrDefault(values, "fontSize", g.Spec.Editor.FontSize).(int)
	g.Spec.Editor.Indentation = getOrDefault(values, "indentation", g.Spec.Editor.Indentation).(string)
	g.Spec.Editor.Theme = getOrDefault(values, "theme", g.Spec.Editor.Theme).(string)
	g.Spec.Editor.TabWidth = getOrDefault(values, "tabWidth", g.Spec.Editor.TabWidth).(int)
	g.Spec.Editor.AutoCloseBrackets = getOrDefault(values, "autoCloseBrackets", g.Spec.Editor.AutoCloseBrackets).(bool)
	g.Spec.Editor.AutoCloseQuotes = getOrDefault(values, "autoCloseQuotes", g.Spec.Editor.AutoCloseQuotes).(bool)

	g.Spec.Scripting.Enabled = getOrDefault(values, "enable", g.Spec.Scripting.Enabled).(bool)
	g.Spec.Scripting.Language = getOrDefault(values, "language", g.Spec.Scripting.Language).(string)
	g.Spec.Scripting.UseDocker = getOrDefault(values, "useDocker", g.Spec.Scripting.UseDocker).(bool)
	g.Spec.Scripting.DockerImage = getOrDefault(values, "dockerImage", g.Spec.Scripting.DockerImage).(string)
	g.Spec.Scripting.ExecutablePath = getOrDefault(values, "executablePath", g.Spec.Scripting.ExecutablePath).(string)
	g.Spec.Scripting.ServerScriptPath = getOrDefault(values, "serverScriptPath", g.Spec.Scripting.ServerScriptPath).(string)
	g.Spec.Scripting.Port = getOrDefault(values, "port", g.Spec.Scripting.Port).(int)

	g.Spec.Data.WorkspacePath = getOrDefault(values, "workspacePath", g.Spec.Data.WorkspacePath).(string)

	return g
}

func getOrDefault(m map[string]any, key string, defaultValue any) any {
	if v, ok := m[key]; ok {
		return v
	}
	return defaultValue
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
				Name: DefaultWorkspaceName,
			},
			SelectedEnvironment: &SelectedEnvironment{},
		},
	}
}

func LegacyConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("AppData")
		if dir == "" {
			return "", errors.New("%AppData% is not defined")
		}

	case "plan9":
		dir = os.Getenv("home")
		if dir == "" {
			return "", errors.New("$home is not defined")
		}
		dir += "/lib"

	default: // Unix
		dir = os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_CONFIG_HOME nor $HOME are defined")
			}
			dir += "/.config"
		}
	}

	path := filepath.Join(dir, "chapar")
	return path, util.MakeDir(path)
}

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
