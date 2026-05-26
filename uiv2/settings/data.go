package settings

import (
	"github.com/chapar-rest/chapar/internal/domain"
)

// DataSettings configures workspace storage.
type DataSettings struct {
	WorkspacePath string `doc:"Absolute path to the workspace folder."`
}

func (d *DataSettings) Defaults() {}

func (d *DataSettings) FromConfig(cfg domain.DataConfig) {
	d.WorkspacePath = cfg.WorkspacePath
}

func (d *DataSettings) ToConfig(cfg *domain.DataConfig) {
	cfg.WorkspacePath = d.WorkspacePath
}
