package pages

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/unit"

	"gioui.org/widget"

	"gioui.org/layout"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type RestContainer struct {
	methodDropDown *widgets.DropDown
	textEditor     widget.Editor

	sendButton widget.Clickable

	split widgets.SplitView

	responseEditor widget.Editor

	requestTabs *widgets.TabsV2
}

func NewRestContainer(theme *material.Theme) *RestContainer {
	r := &RestContainer{
		split: widgets.SplitView{
			Ratio:         0,
			BarWidth:      unit.Dp(2),
			BarColor:      color.NRGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff},
			BarColorHover: theme.Palette.ContrastBg,
		},
	}

	tabV2Items := []widgets.TabV2{
		{Title: "Params"},
		{Title: "Body"},
		{Title: "Headers"},
		{Title: "Pre-req"},
		{Title: "Post-req"},
	}

	onTabsChange := func(index int) {
		fmt.Println("selected tab", index)
	}

	r.requestTabs = widgets.NewTabsV2(tabV2Items, onTabsChange)

	r.methodDropDown = widgets.NewDropDown(theme,
		widgets.NewOption("GET", func() { fmt.Println("none") }),
		widgets.NewOption("POST", func() { fmt.Println("none") }),
		widgets.NewOption("PUT", func() { fmt.Println("none") }),
		widgets.NewOption("PATCH", func() { fmt.Println("none") }),
		widgets.NewOption("DELETE", func() { fmt.Println("none") }),
		widgets.NewOption("HEAD", func() { fmt.Println("none") }),
		widgets.NewOption("OPTION", func() { fmt.Println("none") }),
	)

	r.methodDropDown.SetSize(image.Point{X: 150})
	r.textEditor.SingleLine = true

	r.responseEditor.SingleLine = false
	r.responseEditor.ReadOnly = false

	return r
}

func (r *RestContainer) requestBar(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	border := widget.Border{
		Color:        widgets.Gray400,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
			Spacing:   layout.SpaceEnd,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.methodDropDown.Layout(gtx)
			}),
			widgets.VerticalLine(40.0),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Editor(theme, &r.textEditor, "https://example.com").Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(80)
					return material.Button(theme, &r.sendButton, "Send").Layout(gtx)
				})
			}),
		)
	})
}

func (r *RestContainer) requestLayout(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.requestTabs.Layout(theme, gtx)
		}),
	)
}

func (r *RestContainer) Layout(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Left:   unit.Dp(10),
				Top:    unit.Dp(10),
				Bottom: unit.Dp(5),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme, theme.TextSize, "Create user").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.requestBar(theme, gtx)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return r.split.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return r.requestLayout(theme, gtx)
				},
				func(gtx layout.Context) layout.Dimensions {
					return material.Editor(theme, &r.responseEditor, "").Layout(gtx)
				},
			)
		}),
	)
}
