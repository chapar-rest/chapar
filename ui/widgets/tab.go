package widgets

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const (
	IndicatorPositionTop = iota
	IndicatorPositionBottom
)

type Tab struct {
	clickable *widget.Clickable

	Closable       bool
	CloseClickable *widget.Clickable

	IsSelected bool

	*TabStyle

	Text string
}

type TabStyle struct {
	BackgroundColor   color.NRGBA
	IndicatorColor    color.NRGBA
	IndicatorPosition int

	BorderWidth unit.Dp
	BorderColor color.NRGBA
}

func NewTab(text string, clickable *widget.Clickable, style *TabStyle) *Tab {
	return &Tab{
		Text:      text,
		clickable: clickable,
		TabStyle:  style,
	}
}

func (t *Tab) SetStyle(style *TabStyle) *Tab {
	t.TabStyle = style
	return t
}

func (t *Tab) Layout(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	border := widget.Border{
		Color:        Gray300,
		Width:        t.BorderWidth,
		CornerRadius: 0,
	}

	if t.BackgroundColor == (color.NRGBA{}) {
		t.BackgroundColor = theme.Palette.Bg
	}

	if t.IndicatorColor == (color.NRGBA{}) {
		t.IndicatorColor = theme.Palette.ContrastBg
	}

	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if t.Closable {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, unit.Sp(13), t.Text).Layout(gtx)
					})
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					ib := &IconButton{
						Icon:            CloseIcon,
						Color:           theme.ContrastFg,
						BackgroundColor: t.BackgroundColor,
						Size:            unit.Dp(16),
						Clickable:       t.CloseClickable,
					}
					return ib.Layout(theme, gtx)
				}),
			)
		}

		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme, unit.Sp(13), t.Text).Layout(gtx)
		})
	})
	cc := macro.Stop()

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return t.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					background := t.BackgroundColor
					switch {
					case gtx.Queue == nil:
						background = Disabled(t.BackgroundColor)
					case t.clickable.Hovered() || t.clickable.Focused():
						if !t.IsSelected {
							background = Hovered(t.BackgroundColor)
						}
					}
					paint.Fill(gtx.Ops, background)
					return layout.Dimensions{Size: gtx.Constraints.Min}
				})
			},
			func(gtx layout.Context) layout.Dimensions {
				indicator := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !t.IsSelected {
						return layout.Dimensions{Size: image.Pt(dims.Size.X, 5)}
					}
					return Rect{
						Color: t.IndicatorColor,
						Size:  f32.Point{X: float32(dims.Size.X), Y: 5},
						Radii: 0,
					}.Layout(gtx)
				})

				if t.IndicatorPosition == IndicatorPositionTop {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Baseline}.Layout(gtx,
						indicator,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							cc.Add(gtx.Ops)
							return dims
						}),
					)
				}
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Baseline}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						cc.Add(gtx.Ops)
						return dims
					}),
					indicator,
				)
			},
		)
	})
}
