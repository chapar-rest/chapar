package modal

import (
	"time"

	"gioui.org/widget/material"

	"gioui.org/layout"
	"github.com/chapar-rest/chapar/ui/widgets"
)

var Controller = widgets.NewModal()

func Show(t time.Time, w layout.Widget) {
	Controller.Show(t, w)
}

func Visible() bool {
	return Controller.Visible()
}

func Hide(t time.Time) {
	Controller.Hide(t)
}

func Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return Controller.Layout(gtx, th)
}
