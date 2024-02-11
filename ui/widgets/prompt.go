package widgets

import (
	"image"
	"time"

	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type Prompt struct {
	Title   string
	Content string

	options []string
	result  string

	modal *component.ModalState
}

func NewPrompt(title, content string, options ...string) *Prompt {
	return &Prompt{
		Title:   title,
		Content: content,
		options: options,
		modal: &component.ModalState{
			ScrimState: component.ScrimState{
				VisibilityAnimation: component.VisibilityAnimation{
					Duration: time.Millisecond * 1,
					State:    component.Invisible,
				},
			},
		},
	}
}

func (p *Prompt) Show(theme *material.Theme) {
	p.modal.Show(time.Now(), func(gtx layout.Context) layout.Dimensions {
		return p.modalLayout(gtx, theme)
	})
}

func (p *Prompt) modalLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        Gray300,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(5),
	}

	// Create a full-screen rectangle for the background
	bgRect := image.Rectangle{Max: gtx.Constraints.Max}
	paint.FillShape(gtx.Ops, theme.Palette.Bg, clip.Rect(bgRect).Op())

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, material.H6(theme, p.Title).Layout)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, material.Body1(theme, p.Content).Layout)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Right: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, material.Button(theme, new(widget.Clickable), p.options[0]).Layout)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, material.Button(theme, new(widget.Clickable), p.options[1]).Layout)
								})
							}),
						)
					})
				})
			}),
		)
	})
}

func (p *Prompt) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return component.Modal(theme, p.modal).Layout(gtx)
}
