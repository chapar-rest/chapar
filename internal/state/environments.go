package state

import (
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/repository"
	"github.com/mirzakhany/chapar/internal/safemap"
)

type (
	EnvironmentChangeListener       func(environment *domain.Environment, action Action)
	ActiveEnvironmentChangeListener func(*domain.Environment)
)

type Environments struct {
	environmentChangeListeners       []EnvironmentChangeListener
	activeEnvironmentChangeListeners []ActiveEnvironmentChangeListener
	environments                     *safemap.Map[*domain.Environment]

	activeEnvironment *domain.Environment

	repository repository.Repository
}

func NewEnvironments(repository repository.Repository) *Environments {
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

func (m *Environments) notifyEnvironmentChange(environment *domain.Environment, action Action) {
	for _, listener := range m.environmentChangeListeners {
		listener(environment, action)
	}
}

func (m *Environments) notifyActiveEnvironmentChange(environment *domain.Environment) {
	for _, listener := range m.activeEnvironmentChangeListeners {
		listener(environment)
	}
}

func (m *Environments) AddEnvironment(environment *domain.Environment) {
	m.environments.Set(environment.MetaData.ID, environment)
	m.notifyEnvironmentChange(environment, ActionAdd)
}

func (m *Environments) RemoveEnvironment(environment *domain.Environment) {
	if _, ok := m.environments.Get(environment.MetaData.ID); !ok {
		return
	}

	m.environments.Delete(environment.MetaData.ID)
	m.notifyEnvironmentChange(environment, ActionDelete)
}

func (m *Environments) GetEnvironment(id string) *domain.Environment {
	env, _ := m.environments.Get(id)
	return env
}

func (m *Environments) UpdateEnvironment(env *domain.Environment) {
	if _, ok := m.environments.Get(env.MetaData.ID); !ok {
		return
	}

	m.environments.Set(env.MetaData.ID, env)
	m.notifyEnvironmentChange(env, ActionUpdate)
}

func (m *Environments) SetActiveEnvironment(environment *domain.Environment) {
	if _, ok := m.environments.Get(environment.MetaData.ID); !ok {
		return
	}

	m.activeEnvironment = environment
	m.notifyActiveEnvironmentChange(environment)
}

func (m *Environments) GetActiveEnvironment() *domain.Environment {
	return m.activeEnvironment
}

func (m *Environments) GetEnvironmentFromDisc(id string) (*domain.Environment, error) {
	env, ok := m.environments.Get(id)
	if !ok {
		return nil, ErrNotFound
	}

	return m.repository.GetEnvironment(env.FilePath)
}

func (m *Environments) ReloadEnvironmentFromDisc(id string) {
	env, ok := m.environments.Get(id)
	if !ok {
		// log error and handle it
		return
	}

	env, err := m.GetEnvironmentFromDisc(id)
	if err != nil {
		return
	}

	m.environments.Set(id, env)
	m.notifyEnvironmentChange(env, ActionUpdate)
}

func (m *Environments) GetEnvironments() []*domain.Environment {
	return m.environments.Values()
}

func (m *Environments) LoadEnvironmentsFromDisk() ([]*domain.Environment, error) {
	envs, err := m.repository.LoadEnvironments()
	if err != nil {
		return nil, err
	}

	for _, env := range envs {
		m.environments.Set(env.MetaData.ID, env)
	}

	return envs, nil
}
