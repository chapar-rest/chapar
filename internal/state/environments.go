package state

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/safemap"
)

type (
	EnvironmentChangeListener       func(environment *domain.Environment, source Source, action Action)
	ActiveEnvironmentChangeListener func(*domain.Environment)
)

type Environments struct {
	environmentChangeListeners       []EnvironmentChangeListener
	activeEnvironmentChangeListeners []ActiveEnvironmentChangeListener
	environments                     *safemap.Map[*domain.Environment]

	activeEnvironment *domain.Environment

	repository repository.RepositoryV2
}

func NewEnvironments(repository repository.RepositoryV2) *Environments {
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

func (m *Environments) notifyEnvironmentChange(environment *domain.Environment, source Source, action Action) {
	for _, listener := range m.environmentChangeListeners {
		listener(environment, source, action)
	}
}

func (m *Environments) notifyActiveEnvironmentChange(environment *domain.Environment) {
	for _, listener := range m.activeEnvironmentChangeListeners {
		listener(environment)
	}
}

func (m *Environments) AddEnvironment(environment *domain.Environment, source Source) {
	m.environments.Set(environment.MetaData.ID, environment)
	m.notifyEnvironmentChange(environment, source, ActionAdd)
}

func (m *Environments) RemoveEnvironment(environment *domain.Environment, source Source, stateOnly bool) error {
	if _, ok := m.environments.Get(environment.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.DeleteEnvironment(environment); err != nil {
			return err
		}
	}

	m.environments.Delete(environment.MetaData.ID)
	m.notifyEnvironmentChange(environment, source, ActionDelete)
	return nil
}

func (m *Environments) GetEnvironment(id string) *domain.Environment {
	env, _ := m.environments.Get(id)
	return env
}

func (m *Environments) UpdateEnvironment(env *domain.Environment, source Source, stateOnly bool) error {
	if _, ok := m.environments.Get(env.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.UpdateEnvironment(env); err != nil {
			return err
		}
	}

	m.environments.Set(env.MetaData.ID, env)
	m.notifyEnvironmentChange(env, source, ActionUpdate)

	return nil
}

func (m *Environments) SetActiveEnvironment(environment *domain.Environment) {
	if _, ok := m.environments.Get(environment.MetaData.ID); !ok {
		return
	}

	m.activeEnvironment = environment
	m.notifyActiveEnvironmentChange(environment)
}

func (m *Environments) ClearActiveEnvironment() {
	m.activeEnvironment = nil
	m.notifyActiveEnvironmentChange(nil)
}

func (m *Environments) GetActiveEnvironment() *domain.Environment {
	return m.activeEnvironment
}

func (m *Environments) GetPersistedEnvironment(id string) (*domain.Environment, error) {
	environments, err := m.repository.LoadEnvironments()
	if err != nil {
		return nil, err
	}

	for _, e := range environments {
		if e.MetaData.ID == id {
			return e, nil
		}
	}

	return nil, ErrNotFound
}

func (m *Environments) ReloadEnvironment(id string, source Source) {
	_, ok := m.environments.Get(id)
	if !ok {
		// log error and handle it
		return
	}

	env, err := m.GetPersistedEnvironment(id)
	if err != nil {
		return
	}

	m.environments.Set(id, env)
	m.notifyEnvironmentChange(env, source, ActionUpdate)
}

func (m *Environments) GetEnvironments() []*domain.Environment {
	return m.environments.Values()
}

func (m *Environments) LoadEnvironments() ([]*domain.Environment, error) {
	envs, err := m.repository.LoadEnvironments()
	if err != nil {
		return nil, err
	}

	m.environments.Clear()

	for _, env := range envs {
		m.environments.Set(env.MetaData.ID, env)
	}

	return envs, nil
}
