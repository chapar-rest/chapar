package workspaces

import (
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/state"
)

type Controller struct {
	view *View

	state *state.Workspaces

	repo repository.Repository
}

func NewController(view *View, state *state.Workspaces, repo repository.Repository) *Controller {
	return &Controller{
		view:  view,
		state: state,
		repo:  repo,
	}
}

func (c *Controller) LoadData() error {
	data, err := c.state.LoadWorkspacesFromDisk()
	if err != nil {
		return err
	}

	c.view.SetItems(data)
	return nil
}
