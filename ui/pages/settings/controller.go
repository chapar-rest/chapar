package settings

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
)

type Controller struct {
	view *View

	oldViewChanged bool
}

func NewController(view *View) *Controller {
	c := &Controller{
		view: view,
	}

	view.SetOnChange(c.OnChange)
	return c
}

func (c *Controller) OnChange(values map[string]any) {
	// load data from the settings
	globalSettings := prefs.GetGlobalConfig()
	// input values
	inputSettings := domain.GlobalConfigFromValues(globalSettings, values)
	if globalSettings.Changed(&inputSettings) {
		c.view.IsDataChanged = true
		c.oldViewChanged = true
		c.view.Refresh()
	} else {
		c.view.IsDataChanged = false
		if c.oldViewChanged {
			c.view.Refresh()
		}
		c.oldViewChanged = false
	}
}
