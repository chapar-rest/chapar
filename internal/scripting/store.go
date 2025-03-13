package scripting

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

var _ VariableStore = &Store{}

// Store is a wrapper around repository that provides methods for accessing and modifying data
// this is the first draft of the store, implementation will be improved
type Store struct {
	repo repository.Repository
}

// NewStore creates a new Store
func NewStore(repo repository.Repository) *Store {
	return &Store{
		repo: repo,
	}
}

func (s *Store) Get(name string) (interface{}, bool) {
	env, err := s.getActiveEnvironment()
	if err != nil {
		return nil, false
	}

	for _, v := range env.Spec.Values {
		if v.Key == name {
			return v.Value, true
		}
	}

	return nil, false
}

func (s *Store) Set(name string, value interface{}) {
	env, err := s.getActiveEnvironment()
	if err != nil {
		fmt.Println("error getting active environment", err)
		return
	}

	needUpdate := false
	for i, v := range env.Spec.Values {
		if v.Key == name {
			env.Spec.Values[i].Value = value.(string)
			needUpdate = true
			break
		}
	}

	if needUpdate {
		if err := s.repo.Update(env); err != nil {
			fmt.Println("error updating environment", err)
		}
	}
}

func (s *Store) GetAll() map[string]interface{} {
	env, err := s.getActiveEnvironment()
	if err != nil {
		return nil
	}

	values := make(map[string]interface{})
	for _, v := range env.Spec.Values {
		values[v.Key] = v.Value
	}

	return values
}

func (s *Store) getActiveEnvironment() (*domain.Environment, error) {
	pr, err := s.repo.ReadPreferences()
	if err != nil {
		return nil, err
	}

	return s.repo.GetEnvironment(pr.Spec.SelectedEnvironment.ID)
}
