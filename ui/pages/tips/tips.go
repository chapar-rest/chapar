package tips

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/assets"
	"github.com/mirzakhany/chapar/ui/theme"
)

type Tips struct {
	messages []string

	chaparImage image.Image
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

	data, err := assets.LoadImage("chapar.png")
	if err != nil {
		panic(err)
	}

	tips.chaparImage = data

	return tips
}

func (t *Tips) Layout(gtx layout.Context, theme *theme.Theme) layout.Dimensions {
	items := make([]layout.FlexChild, 0, len(t.messages))

	items = append(items,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return widget.Image{
				Src:      paint.NewImageOp(t.chaparImage),
				Fit:      widget.Unscaled,
				Position: layout.Center,
				Scale:    1.0,
			}.Layout(gtx)
		}),
	)

	for i, m := range t.messages {
		m := m
		i := i
		items = append(items, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				if i == 0 {
					return material.H6(theme.Material(), m).Layout(gtx)
				}
				return material.Body1(theme.Material(), m).Layout(gtx)
			})
		}))
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
			items...,
		)
	})
}
