package repository

import (
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/safemap"
)

type (
	EnvironmentChangeListener       func(*domain.Environment)
	ActiveEnvironmentChangeListener func(*domain.Environment)
)

type Environments struct {
	environmentChangeListeners       []EnvironmentChangeListener
	activeEnvironmentChangeListeners []ActiveEnvironmentChangeListener
	environments                     *safemap.Map[*domain.Environment]

	repository Repository
}

func NewEnvironments(repository Repository) *Environments {
	return &Environments{
		repository:   repository,
		environments: safemap.New[*domain.Environment](),
	}
}

func (m *Environments) AddEnvironmentChangeListener(listener EnvironmentChangeListener) {
	m.environmentChangeListeners = append(m.environmentChangeListeners, listener)
}

func (m *Environments) AddActiveEnvironmentChangeListener(listener ActiveEnvironmentChangeListener) {
	m.activeEnvironmentChangeListeners = append(m.activeEnvironmentChangeListeners, listener)
}

func (m *Environments) notifyEnvironmentChange(environment *domain.Environment) {
	for _, listener := range m.environmentChangeListeners {
		listener(environment)
	}
}

func (m *Environments) notifyActiveEnvironmentChange(environment *domain.Environment) {
	for _, listener := range m.activeEnvironmentChangeListeners {
		listener(environment)
	}
}
