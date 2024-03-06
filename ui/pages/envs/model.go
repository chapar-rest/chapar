package envs

import (
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/manager"
)

type EnvChangeListener func(environment *domain.Environment)

type Model struct {
	*manager.Manager

	listeners []EnvChangeListener
}

func NewModel(m *manager.Manager) *Model {
	return &Model{
		Manager: m,
	}
}

func (m *Model) AddEnvChangeListener(listener EnvChangeListener) {
	m.listeners = append(m.listeners, listener)
}

func (m *Model) notifyEnvChange(env *domain.Environment) {
	for _, listener := range m.listeners {
		listener(env)
	}
}
