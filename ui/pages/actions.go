package pages

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Actions struct {
	theme *material.Theme

	addRequestButton widget.Clickable
	importButton     widget.Clickable

	onClick func(target string)
}

func NewActions(theme *material.Theme) *Actions {
	return &Actions{
		theme: theme,
	}
}

func (a *Actions) OnClick(f func(target string)) {
	a.onClick = f
}

func (a *Actions) Layout(gtx layout.Context) layout.Dimensions {
	for a.addRequestButton.Clicked(gtx) {
		a.onClick("Add")
	}

	for a.importButton.Clicked(gtx) {
		a.onClick("Import")
	}

	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(a.theme, &a.addRequestButton, "Add").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(a.theme, &a.importButton, "Import").Layout(gtx)
		}),
	)
}
