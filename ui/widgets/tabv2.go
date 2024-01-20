package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Tabs struct {
	list     layout.List
	tabs     []Tab
	selected int

	onSelectedChange func(int)
}

type Tab struct {
	btn   widget.Clickable
	Title string

	Closable       bool
	CloseClickable *widget.Clickable
}

func NewTabs(items []Tab, onSelectedChange func(int)) *Tabs {
	t := &Tabs{
		tabs:             items,
		selected:         0,
		onSelectedChange: onSelectedChange,
	}
	return t
}

func (tabs *Tabs) Selected() int {
	return tabs.selected
}

func (tabs *Tabs) Layout(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return tabs.list.Layout(gtx, len(tabs.tabs), func(gtx layout.Context, tabIdx int) layout.Dimensions {
				if tabIdx >= len(tabs.tabs) {
					tabIdx = len(tabs.tabs) - 1
				}

				t := &tabs.tabs[tabIdx]
				if t.Closable && t.CloseClickable.Clicked(gtx) {
					tabs.tabs = append(tabs.tabs[:tabIdx], tabs.tabs[tabIdx+1:]...)
				}

				if t.btn.Clicked(gtx) {
					tabs.selected = tabIdx
					if tabs.onSelectedChange != nil {
						go tabs.onSelectedChange(tabIdx)
					}
				}

				if t.btn.Hovered() || t.btn.Focused() {
					paint.FillShape(gtx.Ops, theme.Palette.ContrastBg, clip.Rect{Max: gtx.Constraints.Min}.Op())
				}

				var tabWidth int
				return layout.Stack{Alignment: layout.S}.Layout(gtx,
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						var dims layout.Dimensions
						if t.Closable {
							dims = Clickable(gtx, &t.btn, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(12)).Layout(gtx,
											material.Label(theme, unit.Sp(13), t.Title).Layout,
										)
									}),
									layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										bkColor := color.NRGBA{}
										hoveredColor := Hovered(bkColor)
										if t.btn.Hovered() {
											bkColor = hoveredColor
										}
										ib := &IconButton{
											Icon:                 CloseIcon,
											Color:                theme.ContrastFg,
											BackgroundColor:      bkColor,
											BackgroundColorHover: hoveredColor,
											Size:                 unit.Dp(16),
											Clickable:            t.CloseClickable,
										}
										return layout.UniformInset(unit.Dp(4)).Layout(gtx,
											func(gtx layout.Context) layout.Dimensions {
												return ib.Layout(theme, gtx)
											},
										)
									}),
								)
							})
						} else {
							dims = Clickable(gtx, &t.btn, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(12)).Layout(gtx,
									material.Label(theme, unit.Sp(13), t.Title).Layout,
								)
							})
						}

						tabWidth = dims.Size.X
						return dims
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						if tabs.selected != tabIdx {
							return layout.Dimensions{}
						}
						tabHeight := gtx.Dp(unit.Dp(2))
						tabRect := image.Rect(0, 0, tabWidth, tabHeight)
						paint.FillShape(gtx.Ops, theme.Palette.ContrastBg, clip.Rect(tabRect).Op())
						return layout.Dimensions{
							Size: image.Point{X: tabWidth, Y: tabHeight},
						}
					}),
				)
			})
		}),
		HorizontalFullLine(),
	)
}
