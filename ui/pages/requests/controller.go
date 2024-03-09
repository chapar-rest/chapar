package requests

type Controller struct {
	model *Model
	view  *View

	activeTabID string
}

func NewController(view *View, model *Model) *Controller {
	c := &Controller{
		view:  view,
		model: model,
	}

	return c
}
