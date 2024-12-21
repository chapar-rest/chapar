package component

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Actions struct {
	SaveButton widget.Clickable
	CodeButton widget.Clickable

	IsDataChanged bool
}

func NewActions() *Actions {
	return &Actions{
		SaveButton: widget.Clickable{},
		CodeButton: widget.Clickable{},
	}
}

func (r *Actions) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        theme.Palette.ContrastFg,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return layout.Inset{Bottom: unit.Dp(15), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				bt := widgets.Button(theme.Material(), &r.SaveButton, widgets.SaveIcon, widgets.IconPositionStart, "Save")
				if r.IsDataChanged {
					bt.Color = theme.Palette.ContrastFg
					border.Width = unit.Dp(1)
					border.Color = theme.Palette.ContrastFg
				} else {
					bt.Color = widgets.Disabled(theme.Palette.ContrastFg)
					border.Color = widgets.Disabled(theme.Palette.ContrastFg)
					border.Width = 0
				}

				return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return bt.Layout(gtx, theme)
				})
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return widgets.Button(theme.Material(), &r.CodeButton, widgets.CodeIcon, widgets.IconPositionStart, "Code").Layout(gtx, theme)
			}),
		)
	})
}
