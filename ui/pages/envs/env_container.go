package envs

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type envContainer struct {
	items *widgets.KeyValue
	title *widgets.EditableLabel

	env *domain.Environment

	searchBox *widgets.TextField

	onEnvChanged func(*domain.Environment)
}

func newEnvContainer() *envContainer {
	search := widgets.NewTextField("", "Search items")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(widgets.Gray600)

	c := &envContainer{
		items: widgets.NewKeyValue(
			widgets.NewKeyValueItem("", "", false),
		),
		title:     widgets.NewEditableLabel(""),
		searchBox: search,
	}

	c.title.SetOnChanged(func(text string) {
		if c.env == nil {
			return
		}

		if c.onEnvChanged != nil {
			c.env.Meta.Name = text
			c.onEnvChanged(c.env)
		}
	})

	c.items.SetOnChanged(func(items []*widgets.KeyValueItem) {
		if c.env == nil {
			return
		}

		if c.onEnvChanged != nil {
			c.env.Values = []domain.EnvValue{}
			for _, vv := range items {
				c.env.Values = append(c.env.Values, domain.EnvValue{
					Key:    vv.Key,
					Value:  vv.Value,
					Enable: vv.Active,
				})
			}
			c.onEnvChanged(c.env)
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

func (r *envContainer) SetOnEnvChanged(f func(*domain.Environment)) {
	r.onEnvChanged = f
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
				return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(15)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
				return r.items.WithAddLayout(gtx, "", "* Disabled items have no effect on your requests", theme)
			}),
		)
	})
}
