package tips

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/assets"
	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type Tips struct {
	messages []string
	items    []layout.FlexChild
}

func New() *Tips {
	tips := &Tips{
		messages: []string{
			"Welcome to Chapar",
			"Double Click on any item to open it.",
			"Use Cmd/Ctrl+s to save the changes",
			"Import your data from other apps using import functionality",
			"Using the environment dropdown you can switch between different environments",
			"Use the sidebar to navigate between different sections",
		},
	}

	tips.items = make([]layout.FlexChild, 0)

	return tips
}

func (t *Tips) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if len(t.items) == 0 {
		t.items = append(t.items,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return widget.Image{
					Src:      assets.ChaparImage,
					Fit:      widget.Unscaled,
					Position: layout.Center,
					Scale:    1.0,
				}.Layout(gtx)
			}),
		)

		for i, m := range t.messages {
			m := m
			i := i
			t.items = append(t.items, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if i == 0 {
						return material.H6(theme.Material(), m).Layout(gtx)
					}
					return material.Body1(theme.Material(), m).Layout(gtx)
				})
			}))
		}
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
			t.items...,
		)
	})
}
