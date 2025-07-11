package workspaces

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/state"
)

type Controller struct {
	view *View

	state *state.Workspaces

	repo repository.RepositoryV2
}

func NewController(view *View, state *state.Workspaces, repo repository.RepositoryV2) *Controller {
	c := &Controller{
		view:  view,
		state: state,
		repo:  repo,
	}

	view.SetOnNew(c.onNew)
	view.SetOnDelete(c.onDelete)
	view.SetOnUpdate(c.onUpdate)

	return c
}

func (c *Controller) LoadData() error {
	data, err := c.state.LoadWorkspaces()
	if err != nil {
		return err
	}

	c.view.SetItems(data)
	return nil
}

func (c *Controller) onNew() {
	ws := domain.NewWorkspace("New Workspace")
	if err := c.repo.CreateWorkspace(ws); err != nil {
		c.view.showError(fmt.Errorf("failed to create workspace: %w", err))
		return
	}

	c.state.AddWorkspace(ws, state.SourceController)
	c.view.AddItem(ws)
}

func (c *Controller) onDelete(w *domain.Workspace) {
	ws := c.state.GetWorkspace(w.MetaData.ID)
	if ws == nil {
		c.view.showError(fmt.Errorf("failed to get workspace, %s", w.MetaData.ID))
		return
	}

	if err := c.state.RemoveWorkspace(w, state.SourceController, false); err != nil {
		c.view.showError(fmt.Errorf("failed to remove workspace, %w", err))
		return
	}

	c.view.RemoveItem(w)
}

func (c *Controller) onUpdate(w *domain.Workspace) {
	ws := c.state.GetWorkspace(w.MetaData.ID)
	if ws == nil {
		c.view.showError(fmt.Errorf("failed to get workspace, %s", w.MetaData.ID))
		return
	}

	if err := c.state.UpdateWorkspace(w, state.SourceController, false); err != nil {
		c.view.showError(fmt.Errorf("failed to update workspace, %w", err))
		return
	}
}
