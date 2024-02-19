package widgets

import (
	"image"
	"image/color"

	"gioui.org/op"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type DropDown struct {
	menuContextArea component.ContextArea
	menu            component.MenuState
	list            *widget.List

	isOpen              bool
	selectedOptionIndex int
	options             []*DropDownOption

	size image.Point

	borderColor  color.NRGBA
	borderWidth  unit.Dp
	cornerRadius unit.Dp
}

type DropDownOption struct {
	Text      string
	clickable widget.Clickable

	isDivider bool
	isDefault bool
}

func NewDropDownOption(text string) *DropDownOption {
	return &DropDownOption{
		Text:      text,
		isDivider: false,
	}
}

func NewDropDownDivider() *DropDownOption {
	return &DropDownOption{
		isDivider: true,
	}
}

func (o *DropDownOption) DefaultSelected() *DropDownOption {
	o.isDefault = true
	return o
}

func NewDropDown(options ...*DropDownOption) *DropDown {
	c := &DropDown{
		menuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		options: options,

		borderColor:  Gray600,
		borderWidth:  unit.Dp(1),
		cornerRadius: unit.Dp(4),
	}

	return c
}

func NewDropDownWithoutBorder(options ...*DropDownOption) *DropDown {
	c := &DropDown{
		menuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		options: options,
	}

	return c
}

func (c *DropDown) SelectedIndex() int {
	return c.selectedOptionIndex
}

func (c *DropDown) SetOptions(options ...*DropDownOption) {
	c.options = options
}

func (c *DropDown) GetSelected() *DropDownOption {
	return c.options[c.selectedOptionIndex]
}

func (c *DropDown) SetBorder(color color.NRGBA, width unit.Dp, cornerRadius unit.Dp) {
	c.borderColor = color
	c.borderWidth = width
	c.cornerRadius = cornerRadius
}

func (c *DropDown) box(gtx layout.Context, theme *material.Theme, text string, minWidth int) layout.Dimensions {
	borderColor := c.borderColor
	if c.isOpen {
		borderColor = theme.Palette.ContrastBg
	}

	border := widget.Border{
		Color:        borderColor,
		Width:        c.borderWidth,
		CornerRadius: c.cornerRadius,
	}
	c.size.X = minWidth
	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// calculate the minimum width of the box, considering icon and padding
		gtx.Constraints.Min.X = minWidth - gtx.Dp(8)
		return layout.Inset{
			Top:    4,
			Bottom: 4,
			Left:   8,
			Right:  4,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, theme.TextSize, text).Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(16)
					return ExpandIcon.Layout(gtx, theme.Palette.Fg)
				}),
			)
		})
	})
}

func (c *DropDown) SetSize(size image.Point) {
	c.size = size
}

// Layout the DropDown.
func (c *DropDown) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	c.isOpen = c.menuContextArea.Active()

	for i, opt := range c.options {
		if opt.isDefault {
			c.selectedOptionIndex = i
		}

		for opt.clickable.Clicked(gtx) {
			c.isOpen = false
			c.selectedOptionIndex = i
		}
	}

	minWidth := 0
	menuMicro := op.Record(gtx.Ops)
	c.menu.Options = c.menu.Options[:0]
	for _, opt := range c.options {
		opt := opt
		c.menu.Options = append(c.menu.Options, func(gtx layout.Context) layout.Dimensions {
			if opt.isDivider {
				dim := component.Divider(theme).Layout(gtx)
				if dim.Size.X > minWidth {
					minWidth = dim.Size.X
				}
				return dim
			}

			dim := component.MenuItem(theme, &opt.clickable, opt.Text).Layout(gtx)
			if dim.Size.X > minWidth {
				minWidth = dim.Size.X
			}

			return dim
		})
	}

	menuDim := component.Menu(theme, &c.menu).Layout(gtx)
	menuMacroCall := menuMicro.Stop()

	box := c.box(gtx, theme, c.options[c.selectedOptionIndex].Text, minWidth)
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return box
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return c.menuContextArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				offset := layout.Inset{
					Top:  unit.Dp(float32(box.Size.Y)/gtx.Metric.PxPerDp + 1),
					Left: unit.Dp(4),
				}
				return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = minWidth
					menuMacroCall.Add(gtx.Ops)
					return menuDim
				})
			})
		}),
	)
}
