package settings

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
)

type Controller struct {
	view *View

	oldViewChanged bool

	state domain.GlobalConfig
}

func NewController(view *View) *Controller {
	c := &Controller{
		view: view,
	}

	view.SetOnChange(c.onChange)
	view.SetOnSave(c.onSave)
	view.SetOnCancel(c.onCancel)
	view.SetOnLoadDefaults(c.onLoadDefaults)
	return c
}

func (c *Controller) onSave() {
	if !c.view.IsDataChanged {
		return
	}

	if err := prefs.UpdateGlobalConfig(c.state); err != nil {
		c.view.ShowError(err)
		return
	}

	c.view.IsDataChanged = false
	c.oldViewChanged = false
	c.view.Refresh()
}

func (c *Controller) onCancel() {
	if !c.view.IsDataChanged {
		return
	}

	c.view.Load(prefs.GetGlobalConfig())
	c.view.IsDataChanged = false
	c.oldViewChanged = false
	c.view.Refresh()
}

func (c *Controller) onLoadDefaults() {
	c.view.Load(*domain.GetDefaultGlobalConfig())
	c.view.IsDataChanged = false
	c.oldViewChanged = false
	c.view.Refresh()
}

func (c *Controller) onChange(values map[string]any) {
	// load data from the settings
	globalSettings := prefs.GetGlobalConfig()
	// input values
	inputSettings := domain.GlobalConfigFromValues(globalSettings, values)
	if globalSettings.Changed(&inputSettings) {
		c.view.IsDataChanged = true
		c.oldViewChanged = true
		c.state = inputSettings
		c.view.Refresh()
	} else {
		c.view.IsDataChanged = false
		if c.oldViewChanged {
			c.view.Refresh()
		}
		c.oldViewChanged = false
	}
}
