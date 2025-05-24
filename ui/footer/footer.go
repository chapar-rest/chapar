package footer

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Footer struct {
	NotificationsClickable widget.Clickable
	ConsoleClickable       widget.Clickable
}

func NewFooter() *Footer {
	return &Footer{}
}

func (f *Footer) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart, Alignment: layout.End}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &f.ConsoleClickable, widgets.ConsoleIcon, widgets.IconPositionStart, "Console")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(10), Right: unit.Dp(10)}
			return btn.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &f.NotificationsClickable, widgets.Notifications, widgets.IconPositionStart, "Notifications")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(10), Right: unit.Dp(10)}
			return btn.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
	)
}
