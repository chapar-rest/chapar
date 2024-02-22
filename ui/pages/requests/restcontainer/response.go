package restcontainer

import (
	"fmt"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mirzakhany/chapar/ui/widgets"
)

func (r *RestContainer) responseLayout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	if r.result == "" {
		return r.messageLayout(gtx, theme, "No response available yet ;)")
	}

	if r.copyResponseButton.Clickable.Clicked(gtx) {
		r.copyResponseToClipboard(gtx)
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return r.responseTabs.Layout(gtx, theme)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.LabelStyle{
							Text:     r.resultStatus,
							Color:    widgets.LightGreen,
							TextSize: theme.TextSize,
							Shaper:   theme.Shaper,
						}
						l.Font.Typeface = theme.Face
						return l.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return r.copyResponseButton.Layout(gtx, theme)
				}),
			)
		}),
		widgets.DrawLineFlex(widgets.Gray300, unit.Dp(1), unit.Dp(gtx.Constraints.Max.Y)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			switch r.responseTabs.Selected() {
			case 1:
				return r.responseKeyValue(gtx, theme, r.responseHeadersList, "headers", r.responseHeaders)
			case 2:
				return r.responseKeyValue(gtx, theme, r.responseCookiesList, "cookies", r.responseCookies)
			default:
				return layout.Inset{Left: unit.Dp(5), Right: unit.Dp(5), Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return r.jsonViewer.Layout(gtx, theme)
				})
			}
		}),
	)
}

func (r *RestContainer) responseKeyValue(gtx layout.Context, theme *material.Theme, state *widget.List, itemType string, items []keyValue) layout.Dimensions {
	if len(items) == 0 {
		return r.messageLayout(gtx, theme, fmt.Sprintf("No %s available", itemType))
	}

	return material.List(theme, state).Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(theme, theme.TextSize, items[i].Key+":")
					l.Font.Weight = font.Bold
					l.State = items[i].keySelectable
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(theme, theme.TextSize, items[i].Value)
					l.State = items[i].valueSelectable
					return l.Layout(gtx)
				})
			}),
		)
	})
}
