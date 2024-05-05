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

	repo repository.Repository
}

func NewController(view *View, state *state.Workspaces, repo repository.Repository) *Controller {
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
	data, err := c.state.LoadWorkspacesFromDisk()
	if err != nil {
		return err
	}

	c.view.SetItems(data)
	return nil
}

func (c *Controller) onNew() {
	ws := domain.NewWorkspace("New Workspace")
	filePath, err := c.repo.GetNewWorkspaceDir(ws.MetaData.Name)
	if err != nil {
		fmt.Println("failed to get new workspace path", err)
		return
	}

	ws.FilePath = filePath.Path
	ws.MetaData.Name = filePath.NewName

	c.state.AddWorkspace(ws, state.SourceController)
}

func (c *Controller) onDelete(w *domain.Workspace) {
	ws := c.state.GetWorkspace(w.MetaData.ID)
	if ws == nil {
		fmt.Println("failed to get workspace", w.MetaData.ID)
		return
	}

	if err := c.state.RemoveWorkspace(w, state.SourceController, false); err != nil {
		fmt.Println("failed to remove workspace", err)
		return
	}
}

func (c *Controller) onUpdate(w *domain.Workspace) {
	ws := c.state.GetWorkspace(w.MetaData.ID)
	if ws == nil {
		fmt.Println("failed to get workspace", w.MetaData.ID)
		return
	}

	if err := c.state.UpdateWrorkspace(w, state.SourceController, false); err != nil {
		fmt.Println("failed to update workspace", err)
		return
	}
}

func (c *Controller) saveWorkspaceToDisc(id string) {
	ws := c.state.GetWorkspace(id)
	if ws == nil {
		fmt.Println("failed to get workspace", id)
		return
	}

	if err := c.state.UpdateWrorkspace(ws, state.SourceController, false); err != nil {
		fmt.Println("failed to update workspace", err)
		return
	}
}
