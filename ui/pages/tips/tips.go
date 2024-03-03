package tips

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type Tips struct {
	messages []string
}

func New() *Tips {
	return &Tips{
		messages: []string{
			"Welcome to Chapar",
			"Double Click on any item to open it.",
			"Use Cmd/Ctrl+s to save the changes",
			"Import your data from other apps using import functionality",
			"Using the environment dropdown you can switch between different environments",
			"Use the sidebar to navigate between different sections",
		},
	}
}

func (t *Tips) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	items := make([]layout.FlexChild, 0, len(t.messages))
	for i, m := range t.messages {
		m := m
		i := i
		items = append(items, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if i == 0 {
					return material.H6(theme, m).Layout(gtx)
				}
				return material.Body1(theme, m).Layout(gtx)
			})
		}))
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start, Spacing: layout.SpaceBetween}.Layout(gtx,
			items...,
		)
	})
}
