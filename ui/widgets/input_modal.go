package widgets

import (
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type InputModal struct {
	textField *TextField
	addBtn    widget.Clickable
	closeBtn  widget.Clickable

	Title string

	onClose func()
	onAdd   func(text string)
}

func NewInputModal(title, placeholder string) *InputModal {
	ed := NewTextField("", placeholder)
	ed.SetIcon(FileFolderIcon, IconPositionStart)
	return &InputModal{
		textField: ed,
		Title:     title,
	}
}

func (i *InputModal) SetOnClose(f func()) {
	i.onClose = f
}

func (i *InputModal) SetOnAdd(f func(text string)) {
	i.onAdd = f
}

func (i *InputModal) SetText(text string) {
	i.textField.SetText(text)
}

func (i *InputModal) layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if i.onClose != nil && i.closeBtn.Clicked(gtx) {
		i.onClose()
	}

	if i.onAdd != nil && i.addBtn.Clicked(gtx) {
		i.onAdd(i.textField.GetText())
	}

	border := widget.Border{
		Color:        theme.TableBorderColor,
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
								return material.Label(theme.Material(), unit.Sp(14), i.Title).Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return i.textField.Layout(gtx, theme)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceStart}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										closeBtn := Button(theme.Material(), &i.closeBtn, CloseIcon, IconPositionStart, "Close")
										closeBtn.Color = theme.ButtonTextColor
										return closeBtn.Layout(gtx, theme)
									}),
									layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										addBtn := Button(theme.Material(), &i.addBtn, PlusIcon, IconPositionStart, "Add")
										addBtn.Color = theme.ButtonTextColor
										addBtn.Background = theme.SendButtonBgColor
										return addBtn.Layout(gtx, theme)
									}),
								)
							}),
						)
					})
				})
			})
		})
	})
}

func (i *InputModal) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	ops := op.Record(gtx.Ops)
	dims := i.layout(gtx, theme)
	defer op.Defer(gtx.Ops, ops.Stop())

	return dims
}
