package environments

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Controller struct {
	view  *View
	state *state.Environments

	repo repository.RepositoryV2

	explorer *explorer.Explorer

	activeTabID string
}

func NewController(view *View, repo repository.RepositoryV2, envState *state.Environments, explorer *explorer.Explorer) *Controller {
	c := &Controller{
		view:     view,
		state:    envState,
		repo:     repo,
		explorer: explorer,
	}

	view.SetOnNewEnv(c.onNewEnvironment)
	view.SetOnImportEnv(c.onImportEnvironment)
	view.SetOnTitleChanged(c.onTitleChanged)
	view.SetOnTreeViewNodeClicked(c.onTreeViewNodeDoubleClicked)
	view.SetOnTabSelected(c.onTabSelected)
	view.SetOnItemsChanged(c.onItemsChanged)
	view.SetOnSave(c.onSave)
	view.SetOnTabClose(c.onTabClose)
	view.SetOnTreeViewMenuClicked(c.onTreeViewMenuClicked)
	envState.AddEnvironmentChangeListener(c.onEnvironmentChange)

	return c
}

func (c *Controller) OpenEnvironment(id string) {
	c.openEnvironment(id)
}

func (c *Controller) onNewEnvironment() {
	env := domain.NewEnvironment("New Environment")
	if err := c.repo.CreateEnvironment(env); err != nil {
		c.view.showError(fmt.Errorf("failed to create environment: %w", err))
		return
	}

	env, err := c.state.GetPersistedEnvironment(env.MetaData.ID)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get environment from file %w", err))
		return
	}

	c.state.AddEnvironment(env, state.SourceController)
	c.view.AddTreeViewNode(env)
}

func (c *Controller) onImportEnvironment() {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Declined {
			return
		}

		if result.Error != nil {
			c.view.showError(fmt.Errorf("failed to get file %w", result.Error))
			return
		}

		if err := importer.ImportPostmanEnvironment(result.Data, c.repo); err != nil {
			c.view.showError(fmt.Errorf("failed to import postman environment %w", err))
			return
		}

		if err := c.LoadData(); err != nil {
			c.view.showError(fmt.Errorf("failed to load environments %w", err))
			return
		}

	}, "json")
}

func (c *Controller) onEnvironmentChange(env *domain.Environment, source state.Source, action state.Action) {
	if source == state.SourceController {
		// if the change is from controller then no need to update the view as it will be updated by the controller
		return
	}

	switch action {
	case state.ActionAdd:
		c.view.AddTreeViewNode(env)
		c.view.ReloadContainerData(env)
	case state.ActionUpdate:
		c.view.UpdateTreeViewNode(env)
		c.view.ReloadContainerData(env)
	case state.ActionDelete:
		c.view.RemoveTreeViewNode(env.MetaData.ID)
		if c.activeTabID == env.MetaData.ID {
			c.activeTabID = ""
			c.view.CloseTab(env.MetaData.ID)
		}
	}
}

func (c *Controller) onTitleChanged(id string, title string) {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return
	}

	env.MetaData.Name = title
	if err := c.state.UpdateEnvironment(env, state.SourceController, false); err != nil {
		c.view.showError(fmt.Errorf("failed to update environment, %w", err))
		return
	}

	c.view.SetTabDirty(id, false)
	c.view.UpdateTreeNodeTitle(id, env.MetaData.Name)
	c.view.UpdateTabTitle(id, env.MetaData.Name)
	c.view.SetContainerTitle(id, env.MetaData.Name)
}

func (c *Controller) onTreeViewNodeDoubleClicked(id string) {
	c.openEnvironment(id)
}

func (c *Controller) openEnvironment(id string) {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return
	}

	if c.view.IsTabOpen(id) {
		c.view.SwitchToTab(env.MetaData.ID)
		c.view.OpenContainer(env)
		return
	}

	c.view.OpenTab(env)
	c.view.OpenContainer(env)
}

func (c *Controller) LoadData() error {
	data, err := c.state.LoadEnvironments()
	if err != nil {
		return err
	}

	c.view.PopulateTreeView(data)
	return nil
}

func (c *Controller) onTabSelected(id string) {
	if c.activeTabID == id {
		return
	}
	c.activeTabID = id
	env := c.state.GetEnvironment(id)
	c.view.SwitchToTab(env.MetaData.ID)
	c.view.OpenContainer(env)
}

func (c *Controller) onItemsChanged(id string, items []domain.KeyValue) {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return
	}

	// is data changed?
	if domain.CompareKeyValues(env.Spec.Values, items) {
		return
	}

	env.Spec.Values = items
	if err := c.state.UpdateEnvironment(env, state.SourceController, true); err != nil {
		c.view.showError(fmt.Errorf("failed to update environment, %w", err))
		return
	}

	// set tab dirty if the in memory data is different from the file
	envFromFile, err := c.state.GetPersistedEnvironment(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get environment from file %w", err))
		return
	}

	c.view.SetTabDirty(id, !domain.CompareKeyValues(env.Spec.Values, envFromFile.Spec.Values))
}

func (c *Controller) onSave(id string) {
	c.saveEnvironment(id)
}

func (c *Controller) onTabClose(id string) {
	// is tab data changed?
	// if yes show prompt
	// if no close tab
	env := c.state.GetEnvironment(id)
	if env == nil {
		c.view.showError(fmt.Errorf("failed to get environment %s", id))
		return
	}

	envFromFile, err := c.state.GetPersistedEnvironment(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get environment from file, %w", err))
		return
	}

	// if data is not changed close the tab
	if domain.CompareKeyValues(env.Spec.Values, envFromFile.Spec.Values) {
		c.view.CloseTab(id)
		return
	}

	// TODO check user preference to remember the choice

	c.view.ShowPrompt(id, "Save", "Do you want to save the changes? (Tips: you can always save the changes using CMD/CTRL+s)", widgets.ModalTypeWarn,
		func(selectedOption string, remember bool) {
			if selectedOption == "Cancel" {
				c.view.HidePrompt(id)
				return
			}

			if selectedOption == "Yes" {
				c.saveEnvironment(id)
			}

			c.view.CloseTab(id)
			c.state.ReloadEnvironment(id, state.SourceController)
		}, []widgets.Option{{Text: "Yes"}, {Text: "No"}, {Text: "Cancel"}}...,
	)
}

func (c *Controller) saveEnvironment(id string) {
	env := c.state.GetEnvironment(id)
	if env == nil {
		c.view.showError(fmt.Errorf("failed to get environment, %s", id))
		return
	}

	if err := c.state.UpdateEnvironment(env, state.SourceController, false); err != nil {
		c.view.showError(fmt.Errorf("failed to update environment, %w", err))
		return
	}

	c.view.SetTabDirty(id, false)
}

func (c *Controller) onTreeViewMenuClicked(id string, action string) {
	switch action {
	case Duplicate:
		c.duplicateEnvironment(id)
	case Delete:
		c.deleteEnvironment(id)
	}
}

func (c *Controller) duplicateEnvironment(id string) {
	// read environment from file to make sure we have the latest persisted data
	envFromFile, err := c.state.GetPersistedEnvironment(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get environment from file, %w", err))
		return
	}

	newEnv := envFromFile.Clone()
	newEnv.MetaData.Name += " (copy)"

	if err := c.repo.CreateEnvironment(newEnv); err != nil {
		c.view.showError(fmt.Errorf("failed to create environment: %w", err))
		return
	}

	c.state.AddEnvironment(newEnv, state.SourceController)
	c.view.AddTreeViewNode(newEnv)
}

func (c *Controller) deleteEnvironment(id string) {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return
	}

	if err := c.state.RemoveEnvironment(env, state.SourceController, false); err != nil {
		c.view.showError(fmt.Errorf("failed to delete environment, %w", err))
		return
	}

	c.view.RemoveTreeViewNode(id)
	c.view.CloseTab(id)
}
