package footer

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Footer struct {
	NotificationsClickable widget.Clickable
	ConsoleClickable       widget.Clickable

	AppVersion string
}

func New(appVersion string) *Footer {
	return &Footer{
		AppVersion:             appVersion,
		NotificationsClickable: widget.Clickable{},
		ConsoleClickable:       widget.Clickable{},
	}
}

func (f *Footer) leftLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Label(theme.Material(), unit.Sp(12), f.AppVersion).Layout(gtx)
	})
}

func (f *Footer) rightLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
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

func (f *Footer) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(24)) // Set a maximum height for the footer
	return component.Sheet{}.Layout(gtx, theme.Material(), &component.VisibilityAnimation{}, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Start}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.leftLayout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.rightLayout(gtx, theme)
			}),
		)
	})
}
