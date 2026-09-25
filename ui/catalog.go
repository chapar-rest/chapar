package ui

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
	Workspaces   []*domain.Workspace

	ActiveEnvID       string
	ActiveWorkspaceID string

	// skipped holds files the loads left out that nobody was told about yet;
	// told remembers which ones were already handed out.
	skipped []repository.SkippedFile
	told    map[string]bool
}

func newCatalog(repo repository.RepositoryV2) *Catalog {
	return &Catalog{repo: repo}
}

func (c *Catalog) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// A file that cannot be read is left out of the lists rather than failing
	// the whole load, so one malformed file does not keep the app from opening.
	cols, err := c.repo.LoadCollections()
	if !c.noteSkipped(err) {
		return err
	}
	reqs, err := c.repo.LoadRequests()
	if !c.noteSkipped(err) {
		return err
	}
	envs, err := c.repo.LoadEnvironments()
	if !c.noteSkipped(err) {
		return err
	}
	protos, err := c.repo.LoadProtoFiles()
	if !c.noteSkipped(err) {
		return err
	}
	workspaces, err := c.repo.LoadWorkspaces()
	if !c.noteSkipped(err) {
		return err
	}

	migrateProtoImportPaths(c.repo, cols, reqs, protos)

	c.Collections = cols
	c.Requests = reqs
	c.Environments = envs
	c.Workspaces = workspaces

	state := prefs.GetAppState()
	if aw := state.Spec.ActiveWorkspace; aw != nil {
		c.ActiveWorkspaceID = aw.ID
		// The repository opens the workspace by folder name, and the saved ID
		// can be stale (a fresh profile saves "default" while the folder's
		// workspace has its own ID). Trust the name when the ID matches none.
		if !hasWorkspace(workspaces, aw.ID) {
			for _, w := range workspaces {
				if w.MetaData.Name == aw.Name {
					c.ActiveWorkspaceID = w.MetaData.ID
					break
				}
			}
		}
	}
	if state.Spec.SelectedEnvironment != nil {
		c.ActiveEnvID = state.Spec.SelectedEnvironment.ID
	}
	return nil
}

// noteSkipped queues the files err reports as skipped. It returns false when
// err is a failure the load cannot get past.
func (c *Catalog) noteSkipped(err error) bool {
	if err == nil {
		return true
	}
	files, ok := repository.SkippedFiles(err)
	if !ok {
		return false
	}
	if c.told == nil {
		c.told = map[string]bool{}
	}
	for _, f := range files {
		key := f.Path + "\x00" + f.Err.Error()
		if c.told[key] {
			continue
		}
		c.told[key] = true
		c.skipped = append(c.skipped, f)
	}
	return true
}

// DrainSkipped returns the files left out of loads since the last call. Each
// problem is returned once, however often the catalog reloads.
func (c *Catalog) DrainSkipped() []repository.SkippedFile {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.skipped
	c.skipped = nil
	return out
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

func hasWorkspace(list []*domain.Workspace, id string) bool {
	for _, w := range list {
		if w.MetaData.ID == id {
			return true
		}
	}
	return false
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
