package state

import (
	"path"

	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/repository"
	"github.com/mirzakhany/chapar/internal/safemap"
)

type (
	RequestChangeListener    func(request *domain.Request, action Action)
	CollectionChangeListener func(collection *domain.Collection, action Action)
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

func (m *Requests) notifyRequestChange(request *domain.Request, action Action) {
	for _, listener := range m.requestChangeListeners {
		listener(request, action)
	}
}

func (m *Requests) notifyCollectionChange(collection *domain.Collection, action Action) {
	for _, listener := range m.collectionChangeListeners {
		listener(collection, action)
	}
}

func (m *Requests) AddRequest(request *domain.Request) {
	m.requests.Set(request.MetaData.ID, request)
	m.notifyRequestChange(request, ActionAdd)
}

func (m *Requests) RemoveRequest(request *domain.Request, stateOnly bool) error {
	if _, ok := m.requests.Get(request.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.DeleteRequest(request); err != nil {
			return err
		}
	}

	m.requests.Delete(request.MetaData.ID)
	m.notifyRequestChange(request, ActionDelete)
	return nil
}

func (m *Requests) AddCollection(collection *domain.Collection) {
	m.collections.Set(collection.MetaData.ID, collection)
	m.notifyCollectionChange(collection, ActionAdd)
}

func (m *Requests) RemoveCollection(collection *domain.Collection, stateOnly bool) error {
	if _, ok := m.collections.Get(collection.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.DeleteCollection(collection); err != nil {
			return err
		}
	}

	m.collections.Delete(collection.MetaData.ID)
	m.notifyCollectionChange(collection, ActionDelete)

	return nil
}
func (m *Requests) AddRequestToCollection(collection *domain.Collection, request *domain.Request) {
	collection.AddRequest(request)
}

func (m *Requests) GetRequest(id string) *domain.Request {
	req, _ := m.requests.Get(id)
	return req
}

func (m *Requests) GetCollection(id string) *domain.Collection {
	collection, _ := m.collections.Get(id)
	return collection
}

func (m *Requests) UpdateRequest(request *domain.Request, stateOnly bool) error {
	if _, ok := m.requests.Get(request.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.UpdateRequest(request); err != nil {
			return err
		}
	}

	m.requests.Set(request.MetaData.ID, request)
	m.notifyRequestChange(request, ActionUpdate)

	return nil
}

func (m *Requests) UpdateCollection(collection *domain.Collection, stateOnly bool) error {
	if _, ok := m.collections.Get(collection.MetaData.ID); !ok {
		return ErrNotFound
	}

	oldCollectionFilePath := collection.FilePath

	if !stateOnly {
		if err := m.repository.UpdateCollection(collection); err != nil {
			return err
		}
	}

	// update the request collection name and id and file path
	for _, req := range collection.Spec.Requests {
		req.CollectionName = collection.MetaData.Name
		req.CollectionID = collection.MetaData.ID

		if oldCollectionFilePath != collection.FilePath {
			req.FilePath = fixRequestFilePath(req, collection)
		}

		m.requests.Set(req.MetaData.ID, req)
	}

	m.collections.Set(collection.MetaData.ID, collection)
	m.notifyCollectionChange(collection, ActionUpdate)

	return nil
}

func fixRequestFilePath(request *domain.Request, collection *domain.Collection) string {
	collectionDir, _ := path.Split(collection.FilePath)
	requestFileName := path.Base(request.FilePath)
	return path.Join(collectionDir, requestFileName)
}

func (m *Requests) GetRequests() []*domain.Request {
	return m.requests.Values()
}

func (m *Requests) GetCollections() []*domain.Collection {
	return m.collections.Values()
}

func (m *Requests) GetRequestFromDisc(id string) (*domain.Request, error) {
	req, ok := m.requests.Get(id)
	if !ok {
		return nil, ErrNotFound
	}

	freshReq, err := m.repository.GetRequest(req.FilePath)
	if err != nil {
		return nil, err
	}

	// update the file path in case if its a collection request
	freshReq.FilePath = req.FilePath

	freshReq.CollectionID = req.CollectionID
	freshReq.CollectionName = req.CollectionName

	return freshReq, nil
}

func (m *Requests) ReloadRequestFromDisc(id string) {
	env, ok := m.requests.Get(id)
	if !ok {
		// log error and handle it
		return
	}

	env, err := m.GetRequestFromDisc(id)
	if err != nil {
		return
	}

	m.requests.Set(id, env)
	m.notifyRequestChange(env, ActionUpdate)
}

func (m *Requests) LoadRequestsFromDisk() ([]*domain.Request, error) {
	reqs, err := m.repository.LoadRequests()
	if err != nil {
		return nil, err
	}

	for _, req := range reqs {
		m.requests.Set(req.MetaData.ID, req)
	}

	return reqs, nil
}

func (m *Requests) LoadCollectionsFromDisk() ([]*domain.Collection, error) {
	cols, err := m.repository.LoadCollections()
	if err != nil {
		return nil, err
	}

	for _, col := range cols {
		m.collections.Set(col.MetaData.ID, col)

		for _, req := range col.Spec.Requests {
			req.CollectionName = col.MetaData.Name
			req.CollectionID = col.MetaData.ID
			m.requests.Set(req.MetaData.ID, req)
		}
	}

	return cols, nil
}
