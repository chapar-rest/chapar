package console

import (
	"fmt"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/ui/widgets"
)

type Console struct {
	logs []domain.Log

	selectables []*widget.Selectable
	list        *widget.List

	clearButton *widget.Clickable
}

func New() *Console {
	c := &Console{
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		clearButton: &widget.Clickable{},
	}

	// bus.Subscribe(state.LogSubmitted, c.handleIncomingLog)
	return c
}

func (c *Console) handleIncomingLog(log any) {
	if log == nil {
		return
	}

	if l, ok := log.(domain.Log); ok {
		c.logs = append(c.logs, l)
	}
}

func (c *Console) logLayout(gtx layout.Context, theme *material.Theme, log *domain.Log) layout.Dimensions {
	textColor := theme.Palette.Fg
	switch log.Level {
	case "info":
		textColor = widgets.LightGreen
	case "error":
		textColor = widgets.LightRed
	case "warn":
		textColor = widgets.LightYellow
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			l := material.Label(theme, theme.TextSize, fmt.Sprintf("[%s] ", log.Time.Format(time.DateTime)))
			l.Color = textColor
			return l.Layout(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.Label(theme, theme.TextSize, log.Message).Layout(gtx)
		}),
	)
}

func (c *Console) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return layout.Inset{
		Top:    unit.Dp(15),
		Left:   unit.Dp(5),
		Bottom: unit.Dp(5),
		Right:  unit.Dp(5),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceStart}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if c.clearButton.Clicked(gtx) {
							c.logs = make([]domain.Log, 0)
						}
						return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.Button(theme, c.clearButton, "Clear").Layout(gtx)
						})
					}),
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return widget.Border{
					Color:        widgets.Gray400,
					Width:        unit.Dp(1),
					CornerRadius: unit.Dp(4),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.List(theme, c.list).Layout(gtx, len(c.logs), func(gtx layout.Context, i int) layout.Dimensions {
							return c.logLayout(gtx, theme, &c.logs[i])
						})
					})
				})
			}),
		)
	})
}
