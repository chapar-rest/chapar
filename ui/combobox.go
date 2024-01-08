package ui

import (
	"fmt"
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type DropDown struct {
	// Theme is the material theme.
	Theme *material.Theme

	menuContextArea component.ContextArea
	menu            component.MenuState
	dropDownIcon    *widget.Icon

	isOpen              bool
	selectedOptionIndex int
	options             []*Option

	size image.Point
}

type Option struct {
	Text    string
	OnClick func()

	clickable widget.Clickable

	isDivider bool
	isDefault bool
}

func NewOption(text string, onClick func()) *Option {
	return &Option{
		Text:      text,
		OnClick:   onClick,
		isDivider: false,
	}
}

func NewDivider() *Option {
	return &Option{
		isDivider: true,
	}
}

func (o *Option) DefaultSelected() *Option {
	o.isDefault = true
	return o
}

func NewDropDown(theme *material.Theme, options ...*Option) *DropDown {
	c := &DropDown{
		Theme: theme,
		menuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
	}

	c.dropDownIcon, _ = widget.NewIcon(icons.NavigationArrowDropDown)

	minX := 0
	for i, opt := range options {
		if opt.isDefault {
			c.selectedOptionIndex = i
		}

		if opt.isDivider {
			c.menu.Options = append(c.menu.Options, component.Divider(theme).Layout)
			continue
		}

		opt.clickable = widget.Clickable{}

		if len(opt.Text) > minX {
			minX = len(opt.Text)
		}
		c.menu.Options = append(c.menu.Options, component.MenuItem(theme, &opt.clickable, opt.Text).Layout)
	}

	c.size.X = minX * 20

	c.options = options
	return c
}

func (c *DropDown) box(gtx C, text string) D {
	borderColor := c.Theme.Palette.ContrastFg //color.NRGBA{R: 0xc0, G: 0xc3, B: 0xc8, A: 0xff}
	if c.isOpen {
		borderColor = c.Theme.Palette.ContrastBg
	}

	cornerRadius := unit.Dp(4)
	border := widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: cornerRadius,
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = c.size
		return layout.Inset{
			Top:    10,
			Bottom: 5,
			Left:   10,
			Right:  5,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Label(c.Theme, unit.Sp(16), text).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return c.dropDownIcon.Layout(gtx, borderColor)
				}),
			)
		})
	})
}

func (c *DropDown) SetSize(size image.Point) {
	c.size = size
}

// Layout the DropDown.
func (c *DropDown) Layout(gtx C) D {
	c.isOpen = c.menuContextArea.Active()

	for i, opt := range c.options {
		for opt.clickable.Clicked(gtx) {
			fmt.Println(i)
			c.isOpen = false
			c.selectedOptionIndex = i
		}
	}

	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx C) D {
			return c.box(gtx, c.options[c.selectedOptionIndex].Text)
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return c.menuContextArea.Layout(gtx, func(gtx C) D {
				offset := layout.Inset{
					Top: unit.Dp(float32(70)/gtx.Metric.PxPerDp + 5),
				}
				return offset.Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min = image.Point{}
					return component.Menu(c.Theme, &c.menu).Layout(gtx)
				})
			})
		}),
	)
}
