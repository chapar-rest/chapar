package console

import (
	"fmt"
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Console struct {
	isVisible bool
	list      *widget.List

	searchBox   *widgets.TextField
	clearButton widget.Clickable
	closeButton widget.Clickable
	copyButton  widget.Clickable
}

func New(theme *chapartheme.Theme) *Console {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(theme.SeparatorColor)
	search.SetSize(image.Point{X: 200, Y: 15})
	search.BorderColorFocused = widgets.WithAlpha(theme.BorderColorFocused, 0x60)

	c := &Console{
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		searchBox: search,
	}

	return c
}

func (c *Console) IsVisible() bool {
	return c.isVisible
}

func (c *Console) ToggleVisibility() {
	c.isVisible = !c.isVisible
}

func (c *Console) SetVisible(visible bool) {
	c.isVisible = visible
}

func (c *Console) logLayout(gtx layout.Context, theme *chapartheme.Theme, log *domain.Log) layout.Dimensions {
	textColor := theme.Palette.Fg
	switch log.Level {
	case "info":
		textColor = chapartheme.LightGreen
	case "error":
		textColor = chapartheme.LightRed
	case "warn":
		textColor = chapartheme.LightYellow
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Label(theme.Material(), theme.TextSize, fmt.Sprintf("[%s] ", log.Time.Format(time.DateTime)))
			l.Color = textColor
			return l.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme.Material(), theme.TextSize, log.Message).Layout(gtx)
		}),
	)
}

func (c *Console) actionsLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(200)
			return c.searchBox.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &c.clearButton, widgets.CleanIcon, widgets.IconPositionStart, "Clear")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(10), Right: unit.Dp(10)}
			return btn.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &c.copyButton, widgets.CopyIcon, widgets.IconPositionStart, "Copy")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(10), Right: unit.Dp(10)}
			return btn.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := widgets.Button(theme.Material(), &c.closeButton, widgets.CloseIcon, widgets.IconPositionStart, "")
			btn.TextSize = unit.Sp(12)
			btn.IconSize = unit.Sp(12)
			btn.IconInset = layout.Inset{Right: unit.Dp(3)}
			btn.Inset = layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(10), Right: unit.Dp(10)}
			return btn.Layout(gtx, theme)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
	)
}

func (c *Console) titleLayout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	width := 0
	return layout.Stack{Alignment: layout.S}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			dims := layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(12)
					return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return widgets.ConsoleIcon.Layout(gtx, theme.Palette.ContrastFg)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(5), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme.Material(), theme.TextSize, "Console").Layout(gtx)
					})
				}),
			)
			width = dims.Size.X
			return dims
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			h := gtx.Dp(unit.Dp(2))
			tabRect := image.Rect(0, 0, width, h)
			paint.FillShape(gtx.Ops, theme.TabInactiveColor, clip.Rect(tabRect).Op())
			return layout.Dimensions{
				Size: image.Point{X: width, Y: h},
			}
		}),
	)
}

func (c *Console) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if !c.isVisible {
		return layout.Dimensions{}
	}

	if c.clearButton.Clicked(gtx) {
		logger.Clear()
	}

	if c.closeButton.Clicked(gtx) {
		c.isVisible = false
	}

	return layout.Inset{
		Top:    unit.Dp(3),
		Left:   unit.Dp(10),
		Bottom: unit.Dp(5),
		Right:  unit.Dp(5),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return c.titleLayout(gtx, theme)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return c.actionsLayout(gtx, theme)
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if len(logger.GetLogs()) == 0 {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.Label(theme.Material(), theme.TextSize, "No logs available").Layout(gtx)
						})
					}

					return material.List(theme.Material(), c.list).Layout(gtx, len(logger.GetLogs()), func(gtx layout.Context, i int) layout.Dimensions {
						return c.logLayout(gtx, theme, &logger.GetLogs()[i])
					})
				})
			}),
		)
	})
}
