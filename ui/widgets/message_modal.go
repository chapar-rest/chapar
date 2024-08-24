package widgets

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

const (
	MessageModalTypeInfo = "info"
	MessageModalTypeWarn = "warn"
	MessageModalTypeErr  = "err"
)

type MessageModal struct {
	Title string
	Body  string
	Type  string

	Visible bool

	options  []ModalOption
	onSubmit OnModalSubmit
}

type ModalOption struct {
	Text   string
	Button widget.Clickable
	Icon   *widget.Icon
}

type OnModalSubmit func(selectedOption string)

func NewMessageModal(title, body, modalType string, onSubmit OnModalSubmit, options ...ModalOption) *MessageModal {
	return &MessageModal{
		Title:    title,
		Body:     body,
		Type:     modalType,
		onSubmit: onSubmit,

		options: options,
	}
}

func (modal *MessageModal) Show() {
	modal.Visible = true
}

func (modal *MessageModal) Hide() {
	modal.Visible = false
}

func (modal *MessageModal) layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	borderColor := theme.TableBorderColor
	switch modal.Type {
	case MessageModalTypeErr:
		borderColor = color.NRGBA{R: 0xD1, G: 0x1E, B: 0x35, A: 0xFF}
	case MessageModalTypeInfo:
		borderColor = color.NRGBA{R: 0x1D, G: 0xBF, B: 0xEC, A: 0xFF}
	case MessageModalTypeWarn:
		borderColor = color.NRGBA{R: 0xFD, G: 0xB5, B: 0x0E, A: 0xFF}
	}

	border := widget.Border{
		Color:        borderColor,
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(1),
	}

	return layout.N.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(80)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(500)
				gtx.Constraints.Max.Y = gtx.Dp(180)

				return component.NewModalSheet(component.NewModal()).Layout(gtx, theme.Material(), &component.VisibilityAnimation{}, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(15)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Label(theme.Material(), unit.Sp(14), modal.Title).Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Body1(theme.Material(), modal.Body).Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								count := len(modal.options)
								items := make([]layout.FlexChild, 0, count)
								for i := range modal.options {
									i := i

									if modal.onSubmit != nil {
										if modal.options[i].Button.Clicked(gtx) {
											modal.onSubmit(modal.options[i].Text)
										}
									}

									items = append(
										items,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											btn := Button(theme.Material(), &modal.options[i].Button, nil, IconPositionStart, modal.options[i].Text)
											btn.Background = chapartheme.White
											btn.Color = chapartheme.Black
											return btn.Layout(gtx, theme)
										}),
										layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
									)
								}
								return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{
										Axis:      layout.Horizontal,
										Alignment: layout.Middle,
										Spacing:   layout.SpaceStart,
									}.Layout(gtx,
										items...,
									)
								})
							}),
						)
					})
				})
			})
		})
	})
}

func (modal *MessageModal) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if modal == nil || !modal.Visible {
		return layout.Dimensions{}
	}

	ops := op.Record(gtx.Ops)
	dims := modal.layout(gtx, theme)
	defer op.Defer(gtx.Ops, ops.Stop())

	return dims
}
