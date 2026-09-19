package uiv2

import (
	"sync"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
)

// Catalog holds list identity for sidebars. Open documents live on containers, not here.
type Catalog struct {
	mu sync.Mutex

	repo repository.RepositoryV2

	Collections  []*domain.Collection
	Requests     []*domain.Request
	Environments []*domain.Environment
	ProtoFiles   []*domain.ProtoFile
	Workspaces   []*domain.Workspace

	ActiveEnvID       string
	ActiveWorkspaceID string
}

func newCatalog(repo repository.RepositoryV2) *Catalog {
	return &Catalog{repo: repo}
}

func (c *Catalog) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cols, err := c.repo.LoadCollections()
	if err != nil {
		return err
	}
	reqs, err := c.repo.LoadRequests()
	if err != nil {
		return err
	}
	envs, err := c.repo.LoadEnvironments()
	if err != nil {
		return err
	}
	protos, err := c.repo.LoadProtoFiles()
	if err != nil {
		return err
	}
	workspaces, err := c.repo.LoadWorkspaces()
	if err != nil {
		return err
	}

	c.Collections = cols
	c.Requests = reqs
	c.Environments = envs
	c.ProtoFiles = protos
	c.Workspaces = workspaces

	state := prefs.GetAppState()
	if state.Spec.ActiveWorkspace != nil {
		c.ActiveWorkspaceID = state.Spec.ActiveWorkspace.ID
	}
	if state.Spec.SelectedEnvironment != nil {
		c.ActiveEnvID = state.Spec.SelectedEnvironment.ID
	}
	return nil
}

func (c *Catalog) RequestByID(id string) *domain.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.Requests {
		if r.MetaData.ID == id {
			return r
		}
	}
	for _, col := range c.Collections {
		for _, r := range col.Spec.Requests {
			if r.MetaData.ID == id {
				r.CollectionID = col.MetaData.ID
				r.CollectionName = col.MetaData.Name
				return r
			}
		}
	}
	return nil
}

func (c *Catalog) CollectionByID(id string) *domain.Collection {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, col := range c.Collections {
		if col.MetaData.ID == id {
			return col
		}
	}
	return nil
}

func (c *Catalog) EnvironmentByID(id string) *domain.Environment {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.Environments {
		if e.MetaData.ID == id {
			return e
		}
	}
	return nil
}

func (c *Catalog) WorkspaceByID(id string) *domain.Workspace {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, w := range c.Workspaces {
		if w.MetaData.ID == id {
			return w
		}
	}
	return nil
}

func (c *Catalog) ActiveEnvironment() *domain.Environment {
	if c.ActiveEnvID == "" {
		return nil
	}
	return c.EnvironmentByID(c.ActiveEnvID)
}

func (c *Catalog) ActiveWorkspace() *domain.Workspace {
	if c.ActiveWorkspaceID == "" {
		return nil
	}
	return c.WorkspaceByID(c.ActiveWorkspaceID)
}

func (c *Catalog) ProtoFileList() []*domain.ProtoFile {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*domain.ProtoFile(nil), c.ProtoFiles...)
}

func (c *Catalog) AllCollections() []*domain.Collection {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*domain.Collection(nil), c.Collections...)
}

func (c *Catalog) StandaloneRequests() []*domain.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*domain.Request(nil), c.Requests...)
}

func (c *Catalog) ReplaceEnvironment(env *domain.Environment) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, e := range c.Environments {
		if e.MetaData.ID == env.MetaData.ID {
			c.Environments[i] = env
			return
		}
	}
}

func (c *Catalog) SetActiveEnv(id string) error {
	c.ActiveEnvID = id
	state := prefs.GetAppState()
	if id == "" {
		state.Spec.SelectedEnvironment = &domain.SelectedEnvironment{}
	} else if env := c.EnvironmentByID(id); env != nil {
		state.Spec.SelectedEnvironment = &domain.SelectedEnvironment{ID: env.MetaData.ID, Name: env.MetaData.Name}
	}
	return prefs.UpdateAppState(state)
}

func (c *Catalog) SetActiveWorkspace(ws *domain.Workspace) error {
	c.ActiveWorkspaceID = ws.MetaData.ID
	c.repo.SetActiveWorkspace(ws.MetaData.Name)
	state := prefs.GetAppState()
	state.Spec.ActiveWorkspace = &domain.ActiveWorkspace{ID: ws.MetaData.ID, Name: ws.MetaData.Name}
	return prefs.UpdateAppState(state)
}
