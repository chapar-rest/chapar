package card

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/widgets"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type Float int

const (
	FloatLeft Float = iota
	FloatRight
)

// Card implements "https://material.io/components/cards".
type Card struct {
	// Media    image.Image
	Title    string
	Subtitle string
	Body     layout.Widget
	Actions  []Action
}

type Action struct {
	*widget.Clickable
	Label string
	Fg    color.NRGBA
	Bg    color.NRGBA
	Float Float
}

func (c Card) Layout(gtx C, th *material.Theme) D {
	return layout.Stack{}.Layout(
		gtx,
		layout.Expanded(func(gtx C) D {
			return widgets.Rect{
				Color: th.ContrastBg,
				Size:  layout.FPt(gtx.Constraints.Min),
				Radii: 8,
			}.Layout(gtx)
		}),
		layout.Stacked(func(gtx C) D {
			return layout.Inset{
				Bottom: unit.Dp(20),
				Left:   unit.Dp(15),
				Right:  unit.Dp(15),
			}.Layout(gtx, func(gtx C) D {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(
					gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Inset{
							Top:    unit.Dp(20),
							Bottom: unit.Dp(20),
						}.Layout(gtx, func(gtx C) D {
							return layout.Flex{
								Axis: layout.Vertical,
							}.Layout(
								gtx,
								layout.Rigid(func(gtx C) D {
									if c.Title == "" {
										return D{}
									}
									return material.H5(th, c.Title).Layout(gtx)
								}),
								layout.Rigid(func(gtx C) D {
									if c.Subtitle == "" {
										return D{}
									}
									return D{Size: image.Point{Y: gtx.Dp(unit.Dp(10))}}
								}),
								layout.Rigid(func(gtx C) D {
									if c.Subtitle == "" {
										return D{}
									}
									return material.Body1(th, c.Subtitle).Layout(gtx)
								}),
							)
						})
					}),
					layout.Rigid(func(gtx C) D {
						if c.Body == nil {
							return D{}
						}
						return c.Body(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						return D{Size: image.Point{Y: gtx.Dp(unit.Dp(20))}}
					}),
					layout.Rigid(func(gtx C) D {
						if len(c.Actions) < 1 {
							return D{}
						}
						return layout.Flex{
							Axis: layout.Horizontal,
						}.Layout(
							gtx,
							func() (flex []layout.FlexChild) {
								var (
									floatRight []Action
									floatLeft  []Action
								)
								for ii := range c.Actions {
									if c.Actions[ii].Float == FloatRight {
										floatRight = append(floatRight, c.Actions[ii])
									} else {
										floatLeft = append(floatLeft, c.Actions[ii])
									}
								}
								for ii := range floatLeft {
									action := &floatLeft[ii]
									flex = append(flex, layout.Rigid(func(gtx C) D {
										btn := material.Button(th, action.Clickable, action.Label)
										btn.Color = action.Fg
										btn.Background = action.Bg
										return btn.Layout(gtx)
									}))
									flex = append(flex, layout.Rigid(func(gtx C) D {
										return layout.Spacer{Width: unit.Dp(5)}.Layout(gtx)
									}))
								}
								if len(floatRight) > 0 {
									flex = append(flex, layout.Flexed(1, func(gtx C) D {
										return D{Size: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Min.Y}}
									}))
								}
								for ii := range floatRight {
									action := &floatRight[ii]
									flex = append(flex, layout.Rigid(func(gtx C) D {
										btn := material.Button(th, action.Clickable, action.Label)
										btn.Color = action.Fg
										btn.Background = action.Bg
										return btn.Layout(gtx)
									}))
									flex = append(flex, layout.Rigid(func(gtx C) D {
										return layout.Spacer{Width: unit.Dp(5)}.Layout(gtx)
									}))
								}
								return flex
							}()...,
						)
					}),
				)
			})
		}),
	)
}
