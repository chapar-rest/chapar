package settings

import (
	"github.com/chapar-rest/chapar/internal/domain"
)

// ScriptingSettings configures the scripting engine.
type ScriptingSettings struct {
	Enabled          bool
	Language         string
	UseDocker        bool
	DockerImage      string
	ExecutablePath   string
	ServerScriptPath string
	Port             int
}

func (s *ScriptingSettings) Defaults() {}

func (s *ScriptingSettings) FromConfig(cfg domain.ScriptingConfig) {
	s.Enabled = cfg.Enabled
	s.Language = cfg.Language
	s.UseDocker = cfg.UseDocker
	s.DockerImage = cfg.DockerImage
	s.ExecutablePath = cfg.ExecutablePath
	s.ServerScriptPath = cfg.ServerScriptPath
	s.Port = cfg.Port
}

func (s *ScriptingSettings) ToConfig(cfg *domain.ScriptingConfig) {
	cfg.Enabled = s.Enabled
	cfg.Language = s.Language
	cfg.UseDocker = s.UseDocker
	cfg.DockerImage = s.DockerImage
	cfg.ExecutablePath = s.ExecutablePath
	cfg.ServerScriptPath = s.ServerScriptPath
	cfg.Port = s.Port
}
