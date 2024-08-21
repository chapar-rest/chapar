package environments

import (
	"errors"
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

	repo repository.Repository

	explorer *explorer.Explorer

	activeTabID string
}

func NewController(view *View, repo repository.Repository, envState *state.Environments, explorer *explorer.Explorer) *Controller {
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

func (c *Controller) onNewEnvironment() error {
	env := domain.NewEnvironment("New Environment")

	filePath, err := c.repo.GetNewEnvironmentFilePath(env.MetaData.Name)
	if err != nil {
		return fmt.Errorf("failed to get new environment file path, %w", err)
	}

	env.FilePath = filePath.Path
	env.MetaData.Name = filePath.NewName

	c.state.AddEnvironment(env, state.SourceController)
	c.view.AddTreeViewNode(env)
	return c.saveEnvironmentToDisc(env.MetaData.ID)
}

func (c *Controller) onImportEnvironment() error {
	err := c.explorer.ChoseFile(func(result explorer.Result) error {
		if result.Error != nil {
			return fmt.Errorf("failed to get file, %w", result.Error)
		}

		if err := importer.ImportPostmanEnvironment(result.Data); err != nil {
			return fmt.Errorf("failed to import postman environment, %w", err)
		}

		if err := c.LoadData(); err != nil {
			return fmt.Errorf("failed to load environments, %w", err)
		}

		return nil
	}, "json")

	return <-err
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

func (c *Controller) onTitleChanged(id string, title string) error {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return fmt.Errorf("failed to get environment, %s", id)
	}

	env.MetaData.Name = title
	c.view.UpdateTreeNodeTitle(env.MetaData.ID, env.MetaData.Name)
	c.view.UpdateTabTitle(env.MetaData.ID, env.MetaData.Name)

	if err := c.state.UpdateEnvironment(env, state.SourceController, false); err != nil {
		return fmt.Errorf("failed to update environment, %w", err)
	}

	c.view.SetTabDirty(id, false)
	return nil
}

func (c *Controller) onTreeViewNodeDoubleClicked(id string) error {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return fmt.Errorf("failed to get environment, %s", id)
	}

	if c.view.IsTabOpen(id) {
		c.view.SwitchToTab(env.MetaData.ID)
		c.view.OpenContainer(env)
		return nil
	}

	c.view.OpenTab(env)
	c.view.OpenContainer(env)

	return nil
}

func (c *Controller) LoadData() error {
	data, err := c.state.LoadEnvironmentsFromDisk()
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

func (c *Controller) onItemsChanged(id string, items []domain.KeyValue) error {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return fmt.Errorf("failed to get environment, %s", id)
	}

	// is data changed?
	if domain.CompareKeyValues(env.Spec.Values, items) {
		return nil
	}

	env.Spec.Values = items
	if err := c.state.UpdateEnvironment(env, state.SourceController, true); err != nil {
		return fmt.Errorf("failed to update environment, %w", err)
	}

	// set tab dirty if the in memory data is different from the file
	envFromFile, err := c.state.GetEnvironmentFromDisc(id)
	if err != nil {
		return fmt.Errorf("failed to get environment from file, %w", err)
	}

	c.view.SetTabDirty(id, !domain.CompareKeyValues(env.Spec.Values, envFromFile.Spec.Values))
	return nil
}

func (c *Controller) onSave(id string) error {
	return c.saveEnvironmentToDisc(id)
}

func (c *Controller) onTabClose(id string) error {
	// is tab data changed?
	// if yes show prompt
	// if no close tab
	env := c.state.GetEnvironment(id)
	if env == nil {
		return fmt.Errorf("failed to get environment, %s", id)
	}

	envFromFile, err := c.state.GetEnvironmentFromDisc(id)
	if err != nil {
		return fmt.Errorf("failed to get environment from file, %w", err)
	}

	// if data is not changed close the tab
	if domain.CompareKeyValues(env.Spec.Values, envFromFile.Spec.Values) {
		c.view.CloseTab(id)
		return nil
	}

	// TODO check user preference to remember the choice

	c.view.ShowPrompt(id, "Save", "Do you want to save the changes? (Tips: you can always save the changes using CMD/CTRL+s)", widgets.ModalTypeWarn,
		func(selectedOption string, remember bool) error {
			if selectedOption == "Cancel" {
				c.view.HidePrompt(id)
				return nil
			}

			if selectedOption == "Yes" {
				if err := c.saveEnvironmentToDisc(id); err != nil {
					return err
				}
			}

			c.view.CloseTab(id)
			c.state.ReloadEnvironmentFromDisc(id, state.SourceController)
			return nil
		}, []widgets.Option{{Text: "Yes"}, {Text: "No"}, {Text: "Cancel"}}...,
	)

	return nil
}

func (c *Controller) saveEnvironmentToDisc(id string) error {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return fmt.Errorf("failed to get environment, %s", id)
	}

	if err := c.state.UpdateEnvironment(env, state.SourceController, false); err != nil {
		return fmt.Errorf("failed to update environment, %w", err)
	}

	c.view.SetTabDirty(id, false)
	return nil
}

func (c *Controller) onTreeViewMenuClicked(id string, action string) error {
	switch action {
	case Duplicate:
		return c.duplicateEnvironment(id)
	case Delete:
		return c.deleteEnvironment(id)
	}

	return errors.New("onTreeViewMenuClicked - unknown action")
}

func (c *Controller) duplicateEnvironment(id string) error {
	// read environment from file to make sure we have the latest persisted data
	envFromFile, err := c.state.GetEnvironmentFromDisc(id)
	if err != nil {
		return fmt.Errorf("failed to get environment from file, %w", err)
	}

	newEnv := envFromFile.Clone()
	newEnv.MetaData.Name += " (copy)"
	newEnv.FilePath = repository.AddSuffixBeforeExt(newEnv.FilePath, "-copy")
	c.state.AddEnvironment(newEnv, state.SourceController)
	c.view.AddTreeViewNode(newEnv)

	return c.saveEnvironmentToDisc(newEnv.MetaData.ID)
}

func (c *Controller) deleteEnvironment(id string) error {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return fmt.Errorf("failed to get environment, %s", id)
	}

	if err := c.state.RemoveEnvironment(env, state.SourceController, false); err != nil {
		return fmt.Errorf("failed to delete environment, %w", err)
	}

	c.view.RemoveTreeViewNode(id)
	c.view.CloseTab(id)
	return nil
}
