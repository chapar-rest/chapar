package settings

import (
	"os"
	"path/filepath"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/uiv2/theme"
)

var (
	App          = &Data{}
	registerOnce sync.Once
)

// Data holds Chapar settings edited in the settings dialog.
type Data struct {
	Appearance AppearanceSettings
	General    GeneralSettings
	Scripting  ScriptingSettings
	Editor     EditorSettings
	Data       DataSettings
}

func (d *Data) Defaults() {
	d.fromConfig(*domain.GetDefaultGlobalConfig())
}

func (d *Data) Apply() {
	d.Appearance.Apply()
}

func (d *Data) fromConfig(cfg domain.GlobalConfig) {
	d.Appearance.FromConfig(cfg.Spec.General)
	d.General.FromConfig(cfg.Spec.General)
	d.Scripting.FromConfig(cfg.Spec.Scripting)
	d.Editor.FromConfig(cfg.Spec.Editor)
	d.Data.FromConfig(cfg.Spec.Data)
}

func (d *Data) toConfig() domain.GlobalConfig {
	cfg := prefs.GetGlobalConfig()
	d.Appearance.ToConfig(&cfg.Spec.General)
	d.General.ToConfig(&cfg.Spec.General)
	d.Scripting.ToConfig(&cfg.Spec.Scripting)
	d.Editor.ToConfig(&cfg.Spec.Editor)
	d.Data.ToConfig(&cfg.Spec.Data)
	return cfg
}

// Init prepares settings. Call before [Load].
func Init() {
	registerOnce.Do(func() {
		errors.Log(redirectCogentCoreSettings())
	})
}

func redirectCogentCoreSettings() error {
	chaparDir, err := prefs.GetConfigDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(chaparDir, "cogentcore")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	core.AppearanceSettings.File = filepath.Join(dir, "appearance-settings.toml")
	core.SystemSettings.File = filepath.Join(dir, "system-settings.toml")
	core.TimingSettings.File = filepath.Join(dir, "timing-settings.toml")
	core.DebugSettings.File = filepath.Join(dir, "debug-settings.toml")
	return nil
}

// Load loads Cogent Core platform settings and Chapar config from prefs.
func Load() error {
	if err := core.LoadAllSettings(); err != nil {
		return err
	}
	prefs.GetInstance()
	App.fromConfig(prefs.GetGlobalConfig())
	App.Apply()
	return nil
}

// LoadOrLog loads settings and logs any error.
func LoadOrLog() {
	errors.Log(Load())
}

// Save persists Chapar settings to disk.
func Save() error {
	return prefs.UpdateGlobalConfig(App.toConfig())
}

// SaveOrLog saves settings and logs any error.
func SaveOrLog() {
	errors.Log(Save())
}

// SetTheme applies the given theme and saves it.
func SetTheme(name theme.Name) {
	App.Appearance.Theme = name
	theme.Apply(name)
	SaveOrLog()
}

func defaultData() Data {
	d := Data{}
	d.Defaults()
	return d
}
