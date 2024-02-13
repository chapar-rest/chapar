package envs

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/loader"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type envContainer struct {
	items *widgets.KeyValue
	title *widgets.EditableLabel

	env *domain.Environment

	searchBox *widgets.TextField

	deleteButton *widget.Clickable

	prompt *widgets.Prompt

	dataChanged bool

	onTitleChanged func(id, title string)
	onDataChanged  func(id string, values []domain.EnvValue)
}

func newEnvContainer(env *domain.Environment) *envContainer {
	search := widgets.NewTextField("", "Search items")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	c := &envContainer{
		items: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),
		title:     widgets.NewEditableLabel(""),
		searchBox: search,

		deleteButton: new(widget.Clickable),

		prompt: widgets.NewPrompt("Save", "This environment value is changed, do you wanna save it before closing it?\nHint: you always can save the changes with ctrl+s", widgets.ModalTypeWarn, "Yes", "No"),
	}
	c.prompt.WithRememberBool()
	c.Load(env)

	c.title.SetOnChanged(func(text string) {
		if c.env == nil {
			return
		}

		if c.env.Meta.Name == text {
			return
		}

		// save changes to the environment
		c.env.Meta.Name = text
		if err := loader.SaveEnvironment(c.env); err != nil {
			c.prompt.Type = widgets.ModalTypeErr
			c.prompt.Content = fmt.Sprintf("Error on saving the environment: %s", err.Error())
			c.prompt.SetOptions("I see")
			c.prompt.WithoutRememberBool()
			c.prompt.Show()
			return
		}

		if c.onTitleChanged != nil {
			c.onTitleChanged(c.env.Meta.ID, text)
		}
	})

	c.items.SetOnChanged(func(items []*widgets.KeyValueItem) {
		if c.env == nil {
			return
		}

		if c.onDataChanged != nil {
			c.env.Values = []domain.EnvValue{}
			for _, vv := range items {
				c.env.Values = append(c.env.Values, domain.EnvValue{
					Key:    vv.Key,
					Value:  vv.Value,
					Enable: vv.Active,
				})
			}
			c.onDataChanged(c.env.Meta.ID, c.env.Values)
		}
	})

	c.searchBox.SetOnTextChange(func(text string) {
		if c.items == nil {
			return
		}

		c.items.Filter(text)
	})

	return c
}

func (r *envContainer) SetOnTitleChanged(f func(string, string)) {
	r.onTitleChanged = f
}

func (r *envContainer) Load(e *domain.Environment) {
	r.env = e
	r.title.SetText(e.Meta.Name)
	r.items.Items = []*widgets.KeyValueItem{}
	for _, vv := range e.Values {
		r.items.AddItem(widgets.NewKeyValueItem(vv.Key, vv.Value, vv.Enable))
	}
}

func (r *envContainer) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.prompt.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top:    unit.Dp(5),
					Bottom: unit.Dp(15),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return r.title.Layout(gtx, theme)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Max.X = gtx.Dp(200)
							return r.searchBox.Layout(gtx, theme)
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return r.items.WithAddLayout(gtx, "", "Disabled items have no effect on your requests", theme)
			}),
		)
	})
}
