package envs

import "github.com/mirzakhany/chapar/internal/domain"

type Controller struct {
	view  *View
	model *Model
}

func NewController(view *View, model *Model) *Controller {
	c := &Controller{
		view:  view,
		model: model,
	}

	model.AddEnvChangeListener(c.onEnvChange)
	view.SetOnNewEnv(c.onNewEnv)
	return c
}

func (c *Controller) onEnvChange(env *domain.Environment) {
	c.view.UpdateTreeNodeTitle(env.MetaData.ID, env.MetaData.Name)
	c.view.UpdateTabTitle(env.MetaData.ID, env.MetaData.Name)
}

func (c *Controller) onNewEnv() {
	env := domain.NewEnvironment("New Environment")
	c.model.AddEnvironment(env)
}
