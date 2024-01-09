package pages

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Actions struct {
	theme *material.Theme

	addRequestButton widget.Clickable
	importButton     widget.Clickable
}

func NewActions(theme *material.Theme) *Actions {
	return &Actions{
		theme: theme,
	}
}

func (a *Actions) Layout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(a.theme, &a.addRequestButton, "Add").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: 2}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(a.theme, &a.importButton, "Import").Layout(gtx)
		}),
	)
}
