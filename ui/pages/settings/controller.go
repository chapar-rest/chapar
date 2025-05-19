package settings

import "fmt"

type Controller struct {
	view *View
}

func NewController(view *View) *Controller {
	c := &Controller{
		view: view,
	}

	view.SetOnChange(c.OnChange)
	return c
}

func (c *Controller) OnChange(values map[string]any) {
	c.view.IsDataChanged = true
	fmt.Println("Settings changed:", values)

}
