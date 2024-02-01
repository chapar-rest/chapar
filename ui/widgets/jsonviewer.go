package widgets

import (
	"fmt"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type JsonViewer struct {
	data string

	lines []string

	list *widget.List
}

func NewJsonViewer() *JsonViewer {
	return &JsonViewer{
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (j *JsonViewer) SetData(data string) {
	j.data = data
	j.lines = strings.Split(data, "\n")
}

func (j *JsonViewer) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        Gray400,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(0),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.List(theme, j.list).Layout(gtx, len(j.lines), func(gtx layout.Context, i int) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(theme, theme.TextSize, fmt.Sprintf("%d", i+1))
						l.Font.Weight = font.Medium
						l.Color = Gray800
						l.Alignment = text.End
						return l.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Label(theme, theme.TextSize, j.lines[i]).Layout(gtx)
					})
				}),
			)
		})
	})
}
