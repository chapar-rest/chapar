package widgets

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type DropDown struct {
	// Theme is the material theme.
	Theme *material.Theme

	menuContextArea component.ContextArea
	menu            component.MenuState

	isOpen              bool
	selectedOptionIndex int
	options             []*Option

	size image.Point

	borderColor  color.NRGBA
	borderWidth  unit.Dp
	cornerRadius unit.Dp
}

type Option struct {
	Text      string
	clickable widget.Clickable

	isDivider bool
	isDefault bool
}

func NewOption(text string) *Option {
	return &Option{
		Text:      text,
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

func (c *DropDown) GetSelected() *Option {
	return c.options[c.selectedOptionIndex]
}

func (c *DropDown) SetBorder(color color.NRGBA, width unit.Dp, cornerRadius unit.Dp) {
	c.borderColor = color
	c.borderWidth = width
	c.cornerRadius = cornerRadius
}

func (c *DropDown) box(gtx layout.Context, text string) layout.Dimensions {
	borderColor := c.borderColor
	if c.isOpen {
		borderColor = c.Theme.Palette.ContrastBg
	}

	border := widget.Border{
		Color:        borderColor,
		Width:        c.borderWidth,
		CornerRadius: c.cornerRadius,
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = c.size
		return layout.Inset{
			Top:    4,
			Bottom: 4,
			Left:   8,
			Right:  8,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(c.Theme, c.Theme.TextSize, text).Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(16)
					return ExpandIcon.Layout(gtx, c.Theme.Palette.Fg)
				}),
			)
		})
	})
}

func (c *DropDown) SetSize(size image.Point) {
	c.size = size
}

// Layout the DropDown.
func (c *DropDown) Layout(gtx layout.Context) layout.Dimensions {
	c.isOpen = c.menuContextArea.Active()

	for i, opt := range c.options {
		for opt.clickable.Clicked(gtx) {
			c.isOpen = false
			c.selectedOptionIndex = i
		}
	}

	box := c.box(gtx, c.options[c.selectedOptionIndex].Text)
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return box
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return c.menuContextArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				offset := layout.Inset{
					Top: unit.Dp(float32(box.Size.Y)/gtx.Metric.PxPerDp + 1),
				}
				return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min = image.Point{}
					return component.Menu(c.Theme, &c.menu).Layout(gtx)
				})
			})
		}),
	)
}
