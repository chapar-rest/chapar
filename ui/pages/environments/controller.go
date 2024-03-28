package environments

import (
	"fmt"

	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/repository"
	"github.com/mirzakhany/chapar/internal/state"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Controller struct {
	view  *View
	state *state.Environments

	repo repository.Repository

	activeTabID string
}

func NewController(view *View, repo repository.Repository, state *state.Environments) *Controller {
	c := &Controller{
		view:  view,
		state: state,
		repo:  repo,
	}

	view.SetOnNewEnv(c.onNewEnvironment)
	view.SetOnTitleChanged(c.onTitleChanged)
	view.SetOnTreeViewNodeClicked(c.onTreeViewNodeDoubleClicked)
	view.SetOnTabSelected(c.onTabSelected)
	view.SetOnItemsChanged(c.onItemsChanged)
	view.SetOnSave(c.onSave)
	view.SetOnTabClose(c.onTabClose)
	view.SetOnTreeViewMenuClicked(c.onTreeViewMenuClicked)
	return c
}

func (c *Controller) onNewEnvironment() {
	env := domain.NewEnvironment("New Environment")

	filePath, err := c.repo.GetNewEnvironmentFilePath(env.MetaData.Name)
	if err != nil {
		fmt.Println("failed to get new environment file path", err)
		return
	}

	fmt.Println("filePath", filePath)

	env.FilePath = filePath.Path
	env.MetaData.Name = filePath.NewName

	c.state.AddEnvironment(env)
	c.view.AddTreeViewNode(env)
	c.saveEnvironmentToDisc(env.MetaData.ID)
}

func (c *Controller) onTitleChanged(id string, title string) {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return
	}

	env.MetaData.Name = title
	c.view.UpdateTreeNodeTitle(env.MetaData.ID, env.MetaData.Name)
	c.view.UpdateTabTitle(env.MetaData.ID, env.MetaData.Name)

	if err := c.state.UpdateEnvironment(env, false); err != nil {
		fmt.Println("failed to update environment", err)
		return
	}

	c.view.SetTabDirty(id, false)
}

func (c *Controller) onTreeViewNodeDoubleClicked(id string) {
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
	if err := c.state.UpdateEnvironment(env, true); err != nil {
		fmt.Println("failed to update environment", err)
		return
	}

	// set tab dirty if the in memory data is different from the file
	envFromFile, err := c.state.GetEnvironmentFromDisc(id)
	if err != nil {
		fmt.Println("failed to get environment from file", err)
		return
	}

	c.view.SetTabDirty(id, !domain.CompareKeyValues(env.Spec.Values, envFromFile.Spec.Values))
}

func (c *Controller) onSave(id string) {
	c.saveEnvironmentToDisc(id)
}

func (c *Controller) onTabClose(id string) {
	// is tab data changed?
	// if yes show prompt
	// if no close tab
	env := c.state.GetEnvironment(id)
	if env == nil {
		fmt.Println("failed to get environment", id)
		return
	}

	envFromFile, err := c.state.GetEnvironmentFromDisc(id)
	if err != nil {
		fmt.Println("failed to get environment from file", err)
		return
	}

	// if data is not changed close the tab
	if domain.CompareKeyValues(env.Spec.Values, envFromFile.Spec.Values) {
		c.view.CloseTab(id)
		return
	}

	// TODO check user preference to remember the choice

	c.view.ShowPrompt(id, "Save", "Do you want to save the changes?", widgets.ModalTypeWarn,
		func(selectedOption string, remember bool) {
			if selectedOption == "Cancel" {
				c.view.HidePrompt(id)
				return
			}

			if selectedOption == "Yes" {
				c.saveEnvironmentToDisc(id)
			}

			c.view.CloseTab(id)
			c.state.ReloadEnvironmentFromDisc(id)
		}, "Yes", "No", "Cancel",
	)
}

func (c *Controller) saveEnvironmentToDisc(id string) {
	env := c.state.GetEnvironment(id)
	if env == nil {
		fmt.Println("failed to get environment", id)
		return
	}

	if err := c.state.UpdateEnvironment(env, false); err != nil {
		fmt.Println("failed to update environment", err)
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
	envFromFile, err := c.state.GetEnvironmentFromDisc(id)
	if err != nil {
		fmt.Println("failed to get environment from file", err)
		return
	}

	newEnv := envFromFile.Clone()
	newEnv.MetaData.Name = newEnv.MetaData.Name + " (copy)"
	newEnv.FilePath = repository.AddSuffixBeforeExt(newEnv.FilePath, "-copy")
	c.state.AddEnvironment(newEnv)
	c.view.AddTreeViewNode(newEnv)
	c.saveEnvironmentToDisc(newEnv.MetaData.ID)
}

func (c *Controller) deleteEnvironment(id string) {
	env := c.state.GetEnvironment(id)
	if env == nil {
		return
	}

	if err := c.state.RemoveEnvironment(env, false); err != nil {
		fmt.Println("failed to delete environment", err)
		return
	}

	c.view.RemoveTreeViewNode(id)
	c.view.CloseTab(id)
}
