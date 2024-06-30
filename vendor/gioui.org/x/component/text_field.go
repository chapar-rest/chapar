package component

import (
	"image"
	"image/color"
	"strconv"
	"time"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// TextField implements the Material Design Text Field
// described here: https://material.io/components/text-fields
type TextField struct {
	// Editor contains the edit buffer.
	widget.Editor
	// click detects when the mouse pointer clicks or hovers
	// within the textfield.
	click gesture.Click

	// Helper text to give additional context to a field.
	Helper string
	// CharLimit specifies the maximum number of characters the text input
	// will allow. Zero means "no limit".
	CharLimit uint
	// Prefix appears before the content of the text input.
	Prefix layout.Widget
	// Suffix appears after the content of the text input.
	Suffix layout.Widget

	// Animation state.
	state
	label  label
	border border
	helper helper
	anim   *Progress

	// errored tracks whether the input is in an errored state.
	// This is orthogonal to the other states: the input can be both errored
	// and inactive for example.
	errored bool
}

// Validator validates text and returns a string describing the error.
// Error is displayed as helper text.
type Validator = func(string) string

type label struct {
	TextSize unit.Sp
	Inset    layout.Inset
	Smallest layout.Dimensions
}

type border struct {
	Thickness unit.Dp
	Color     color.NRGBA
}

type helper struct {
	Color color.NRGBA
	Text  string
}

type state int

const (
	inactive state = iota
	hovered
	activated
	focused
)

// IsActive if input is in an active state (Active, Focused or Errored).
func (in TextField) IsActive() bool {
	return in.state >= activated
}

// IsErrored if input is in an errored state.
// Typically this is when the validator has returned an error message.
func (in *TextField) IsErrored() bool {
	return in.errored
}

// SetError puts the input into an errored state with the specified error text.
func (in *TextField) SetError(err string) {
	in.errored = true
	in.helper.Text = err
}

// ClearError clears any errored status.
func (in *TextField) ClearError() {
	in.errored = false
	in.helper.Text = in.Helper
}

// Clear the input text and reset any error status.
func (in *TextField) Clear() {
	in.Editor.SetText("")
	in.ClearError()
}

// TextTooLong returns whether the current editor text exceeds the set character
// limit.
func (in *TextField) TextTooLong() bool {
	return !(in.CharLimit == 0 || uint(len(in.Editor.Text())) < in.CharLimit)
}

func (in *TextField) Update(gtx C, th *material.Theme, hint string) {
	disabled := gtx.Source == (input.Source{})
	for {
		ev, ok := in.click.Update(gtx.Source)
		if !ok {
			break
		}
		switch ev.Kind {
		case gesture.KindPress:
			gtx.Execute(key.FocusCmd{Tag: &in.Editor})
		}
	}
	in.state = inactive
	if in.click.Hovered() && !disabled {
		in.state = hovered
	}
	hasContents := in.Editor.Len() > 0
	if hasContents {
		in.state = activated
	}
	if gtx.Source.Focused(&in.Editor) && !disabled {
		in.state = focused
	}
	const (
		duration = time.Millisecond * 100
	)
	if in.anim == nil {
		in.anim = &Progress{}
	}
	if in.state == activated || hasContents {
		in.anim.Start(gtx.Now, Forward, 0)
	}
	if in.state == focused && !hasContents && !in.anim.Started() {
		in.anim.Start(gtx.Now, Forward, duration)
	}
	if in.state == inactive && !hasContents && in.anim.Finished() {
		in.anim.Start(gtx.Now, Reverse, duration)
	}
	if in.anim.Started() {
		gtx.Execute(op.InvalidateCmd{})
	}
	in.anim.Update(gtx.Now)
	var (
		// Text size transitions.
		textNormal = th.TextSize
		textSmall  = th.TextSize * 0.8
		// Border color transitions.
		borderColor        = WithAlpha(th.Palette.Fg, 128)
		borderColorHovered = WithAlpha(th.Palette.Fg, 221)
		borderColorActive  = th.Palette.ContrastBg
		// TODO: derive from Theme.Error or Theme.Danger
		dangerColor = color.NRGBA{R: 200, A: 255}
		// Border thickness transitions.
		borderThickness       = unit.Dp(0.5)
		borderThicknessActive = unit.Dp(2.0)
	)
	in.label.TextSize = unit.Sp(lerp(float32(textSmall), float32(textNormal), 1.0-in.anim.Progress()))
	switch in.state {
	case inactive:
		in.border.Thickness = borderThickness
		in.border.Color = borderColor
		in.helper.Color = borderColor
	case hovered, activated:
		in.border.Thickness = borderThickness
		in.border.Color = borderColorHovered
		in.helper.Color = borderColorHovered
	case focused:
		in.border.Thickness = borderThicknessActive
		in.border.Color = borderColorActive
		in.helper.Color = borderColorHovered
	}
	if in.IsErrored() {
		in.border.Color = dangerColor
		in.helper.Color = dangerColor
	}
	// Calculate the dimensions of the smallest label size and store the
	// result for use in clipping.
	// Hack: Reset min constraint to 0 to avoid min == max.
	gtx.Constraints.Min.X = 0
	macro := op.Record(gtx.Ops)
	var spacing unit.Dp
	if len(hint) > 0 {
		spacing = 4
	}
	in.label.Smallest = layout.Inset{
		Left:  spacing,
		Right: spacing,
	}.Layout(gtx, func(gtx C) D {
		return material.Label(th, textSmall, hint).Layout(gtx)
	})
	macro.Stop()
	labelTopInsetNormal := float32(in.label.Smallest.Size.Y) - float32(in.label.Smallest.Size.Y/4)
	topInsetDP := unit.Dp(labelTopInsetNormal / gtx.Metric.PxPerDp)
	topInsetActiveDP := (topInsetDP / 2 * -1) - unit.Dp(in.border.Thickness)
	in.label.Inset = layout.Inset{
		Top:  unit.Dp(lerp(float32(topInsetDP), float32(topInsetActiveDP), in.anim.Progress())),
		Left: unit.Dp(10),
	}
}

func (in *TextField) Layout(gtx C, th *material.Theme, hint string) D {
	in.Update(gtx, th, hint)
	// Offset accounts for label height, which sticks above the border dimensions.
	defer op.Offset(image.Pt(0, in.label.Smallest.Size.Y/2)).Push(gtx.Ops).Pop()
	in.label.Inset.Layout(
		gtx,
		func(gtx C) D {
			return layout.Inset{
				Left:  unit.Dp(4),
				Right: unit.Dp(4),
			}.Layout(gtx, func(gtx C) D {
				label := material.Label(th, unit.Sp(in.label.TextSize), hint)
				label.Color = in.border.Color
				return label.Layout(gtx)
			})
		})

	dims := layout.Flex{
		Axis: layout.Vertical,
	}.Layout(
		gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Stack{}.Layout(
				gtx,
				layout.Expanded(func(gtx C) D {
					cornerRadius := unit.Dp(4)
					dimsFunc := func(gtx C) D {
						return D{Size: image.Point{
							X: gtx.Constraints.Max.X,
							Y: gtx.Constraints.Min.Y,
						}}
					}
					border := widget.Border{
						Color:        in.border.Color,
						Width:        unit.Dp(in.border.Thickness),
						CornerRadius: cornerRadius,
					}
					if gtx.Source.Focused(&in.Editor) || in.Editor.Len() > 0 {
						visibleBorder := clip.Path{}
						visibleBorder.Begin(gtx.Ops)
						// Move from the origin to the beginning of the
						visibleBorder.LineTo(f32.Point{
							Y: float32(gtx.Constraints.Min.Y),
						})
						visibleBorder.LineTo(f32.Point{
							X: float32(gtx.Constraints.Max.X),
							Y: float32(gtx.Constraints.Min.Y),
						})
						visibleBorder.LineTo(f32.Point{
							X: float32(gtx.Constraints.Max.X),
						})
						labelStartX := float32(gtx.Dp(in.label.Inset.Left))
						labelEndX := labelStartX + float32(in.label.Smallest.Size.X)
						labelEndY := float32(in.label.Smallest.Size.Y)
						visibleBorder.LineTo(f32.Point{
							X: labelEndX,
						})
						visibleBorder.LineTo(f32.Point{
							X: labelEndX,
							Y: labelEndY,
						})
						visibleBorder.LineTo(f32.Point{
							X: labelStartX,
							Y: labelEndY,
						})
						visibleBorder.LineTo(f32.Point{
							X: labelStartX,
						})
						visibleBorder.LineTo(f32.Point{})
						visibleBorder.Close()
						defer clip.Outline{
							Path: visibleBorder.End(),
						}.Op().Push(gtx.Ops).Pop()
					}
					return border.Layout(gtx, dimsFunc)
				}),
				layout.Stacked(func(gtx C) D {
					return layout.UniformInset(unit.Dp(12)).Layout(
						gtx,
						func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Flex{
								Axis:      layout.Horizontal,
								Alignment: layout.Middle,
							}.Layout(
								gtx,
								layout.Rigid(func(gtx C) D {
									if in.IsActive() && in.Prefix != nil {
										return in.Prefix(gtx)
									}
									return D{}
								}),
								layout.Flexed(1, func(gtx C) D {
									return material.Editor(th, &in.Editor, "").Layout(gtx)
								}),
								layout.Rigid(func(gtx C) D {
									if in.IsActive() && in.Suffix != nil {
										return in.Suffix(gtx)
									}
									return D{}
								}),
							)
						},
					)
				}),
				layout.Expanded(func(gtx C) D {
					defer pointer.PassOp{}.Push(gtx.Ops).Pop()
					defer clip.Rect(image.Rectangle{
						Max: gtx.Constraints.Min,
					}).Push(gtx.Ops).Pop()
					in.click.Add(gtx.Ops)
					return D{}
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
				Spacing:   layout.SpaceBetween,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx C) D {
					if in.helper.Text == "" {
						return D{}
					}
					return layout.Inset{
						Top:  unit.Dp(4),
						Left: unit.Dp(10),
					}.Layout(
						gtx,
						func(gtx C) D {
							helper := material.Label(th, unit.Sp(12), in.helper.Text)
							helper.Color = in.helper.Color
							return helper.Layout(gtx)
						},
					)
				}),
				layout.Rigid(func(gtx C) D {
					if in.CharLimit == 0 {
						return D{}
					}
					return layout.Inset{
						Top:   unit.Dp(4),
						Right: unit.Dp(10),
					}.Layout(
						gtx,
						func(gtx C) D {
							count := material.Label(
								th,
								unit.Sp(12),
								strconv.Itoa(in.Editor.Len())+"/"+strconv.Itoa(int(in.CharLimit)),
							)
							count.Color = in.helper.Color
							return count.Layout(gtx)
						},
					)
				}),
			)
		}),
	)
	return D{
		Size: image.Point{
			X: dims.Size.X,
			Y: dims.Size.Y + in.label.Smallest.Size.Y/2,
		},
		Baseline: dims.Baseline,
	}
}

// interpolate linearly between two values based on progress.
//
// Progress is expected to be [0, 1]. Values greater than 1 will therefore be
// become a coeficient.
//
// For example, 2.5 is 250% progress.
func lerp(start, end, progress float32) float32 {
	return start + (end-start)*progress
}
