package component

import (
	"gioui.org/layout"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

const (
	MessageTypeInfo    = "info"
	MessageTypeError   = "error"
	MessageTypeWarning = "warning"
)

func Message(gtx layout.Context, messageType string, theme *chapartheme.Theme, message string) layout.Dimensions {
	textColor := theme.TextColor
	switch messageType {
	case MessageTypeError:
		textColor = theme.ErrorColor
	case MessageTypeWarning:
		textColor = theme.WarningColor
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		l := material.LabelStyle{
			Text:     message,
			Color:    textColor,
			TextSize: theme.TextSize,
			Shaper:   theme.Shaper,
		}
		l.Font.Typeface = theme.Face
		return l.Layout(gtx)
	})
}
