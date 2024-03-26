package state

import (
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/repository"
	"github.com/mirzakhany/chapar/internal/safemap"
)

type (
	RequestChangeListener    func(*domain.Request)
	CollectionChangeListener func(**domain.Collection)
)

type Requests struct {
	requestChangeListeners    []RequestChangeListener
	collectionChangeListeners []CollectionChangeListener

	requests    *safemap.Map[*domain.Request]
	collections *safemap.Map[*domain.Collection]

	repository repository.Repository
}

func NewRequests(repository repository.Repository) *Requests {
	return &Requests{
		repository:  repository,
		requests:    safemap.New[*domain.Request](),
		collections: safemap.New[*domain.Collection](),
	}
}

func (m *Requests) AddRequestChangeListener(listener RequestChangeListener) {
	m.requestChangeListeners = append(m.requestChangeListeners, listener)
}

func (m *Requests) AddCollectionChangeListener(listener CollectionChangeListener) {
	m.collectionChangeListeners = append(m.collectionChangeListeners, listener)
}

func (m *Requests) notifyRequestChange(request *domain.Request) {
	for _, listener := range m.requestChangeListeners {
		listener(request)
	}
}

func (m *Requests) notifyCollectionChange(collection **domain.Collection) {
	for _, listener := range m.collectionChangeListeners {
		listener(collection)
	}
}
