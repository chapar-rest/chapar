package ui

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type ComboBox struct {
	// Theme is the material theme.
	Theme *material.Theme

	textEditor widget.Editor
	textField  material.EditorStyle
	textLabel  widget.Label

	menuContextArea component.ContextArea
	menu            component.MenuState
	dropDownIcon    *widget.Icon

	isOpen bool
	minX   int

	selectedOptionIndex int
	options             []*Option
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

func NewComboBox(theme *material.Theme, options ...*Option) *ComboBox {
	c := &ComboBox{
		Theme:      theme,
		textEditor: widget.Editor{},
		menuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
		textLabel: widget.Label{},
	}

	c.dropDownIcon, _ = widget.NewIcon(icons.NavigationArrowDropDown)

	var editorText = ""
	for i, opt := range options {

		if opt.isDefault {
			editorText = opt.Text
			c.selectedOptionIndex = i
		}

		if opt.isDivider {
			c.menu.Options = append(c.menu.Options, component.Divider(theme).Layout)
			continue
		}

		opt.clickable = widget.Clickable{}
		c.menu.Options = append(c.menu.Options, component.MenuItem(theme, &opt.clickable, opt.Text).Layout)
	}

	c.options = options

	c.textField = material.Editor(theme, &c.textEditor, "")
	c.textField.Editor.ReadOnly = true
	c.textField.Editor.SetText(editorText)

	return c
}

func (c *ComboBox) TextField(gtx C, text string) D {
	borderColor := color.NRGBA{R: 0xc0, G: 0xc3, B: 0xc8, A: 0xff}
	if c.isOpen {
		borderColor = color.NRGBA{R: 0x3f, G: 0x7e, B: 0xca, A: 0xff}
	}

	cornerRadius := unit.Dp(4)
	border := widget.Border{
		Color:        borderColor,
		Width:        unit.Dp(1),
		CornerRadius: cornerRadius,
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = 300
		gtx.Constraints.Min.Y = 60
		return layout.Inset{
			Top:    10,
			Bottom: 0,
			Left:   10,
			Right:  5,
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					c.textField.Editor.SetText(text)
					return c.textField.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return c.dropDownIcon.Layout(gtx, borderColor)
				}),
			)
		})
	})
}

// Layout the ComboBox.
func (c *ComboBox) Layout(gtx C) D {
	// set min width to 250
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
			return c.TextField(gtx, c.options[c.selectedOptionIndex].Text)
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
