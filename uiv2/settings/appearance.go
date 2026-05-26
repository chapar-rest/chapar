package settings

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/theme"
)

// AppearanceSettings configures visual presentation.
type AppearanceSettings struct {
	Theme theme.Name `default:"dark"`
}

func (a *AppearanceSettings) Defaults() {
	if a.Theme == "" {
		a.Theme = theme.Dark
	}
}

func (a *AppearanceSettings) FromConfig(cfg domain.GeneralConfig) {
	a.Theme = theme.Name(cfg.Theme)
	if a.Theme == "" {
		a.Theme = theme.Dark
	}
}

func (a *AppearanceSettings) ToConfig(cfg *domain.GeneralConfig) {
	cfg.Theme = string(a.Theme)
}

func (a *AppearanceSettings) Apply() {
	theme.Apply(a.Theme)
}
