package component

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/chapartheme"
)

func Message(gtx layout.Context, theme *chapartheme.Theme, message string) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		l := material.LabelStyle{
			Text:     message,
			Color:    theme.TextColor,
			TextSize: theme.TextSize,
			Shaper:   theme.Shaper,
		}
		l.Font.Typeface = theme.Face
		return l.Layout(gtx)
	})
}
