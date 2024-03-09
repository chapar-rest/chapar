package requests

import (
	"github.com/mirzakhany/chapar/ui/manager"
)

type Model struct {
	*manager.Manager
}

func NewModel(m *manager.Manager) *Model {
	return &Model{
		Manager: m,
	}
}
