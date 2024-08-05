package widgets

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type LabeledInput struct {
	Label          string
	SpaceBetween   int
	MinEditorWidth unit.Dp
	MinLabelWidth  unit.Dp
	Editor         *PatternEditor
	Hint           string
}

func (l *LabeledInput) SetText(text string) {
	l.Editor.SetText(text)
}

func (l *LabeledInput) Text() string {
	return l.Editor.Text()
}

func (l *LabeledInput) SetHint(hint string) {
	l.Hint = hint
}

func (l *LabeledInput) SetLabel(label string) {
	l.Label = label
}

func (l *LabeledInput) SetOnChanged(f func(text string)) {
	l.Editor.SetOnChanged(f)
}

func (l *LabeledInput) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(l.SpaceBetween)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(l.MinLabelWidth)
				return material.Label(theme.Material(), theme.TextSize, l.Label).Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(l.MinEditorWidth)
			return widget.Border{
				Color:        theme.BorderColor,
				Width:        unit.Dp(1),
				CornerRadius: unit.Dp(4),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return l.Editor.Layout(gtx, theme, l.Hint)
				})
			})
		}),
	)
}
