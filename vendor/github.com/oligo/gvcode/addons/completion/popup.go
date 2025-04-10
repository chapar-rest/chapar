package completion

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gvcode"
)

// CompletionPopup is the builtin implementation of a completion popup.
type CompletionPopup struct {
	editor  *gvcode.Editor
	cmp     gvcode.Completion
	list    widget.List
	focused int
	labels  []*itemLabel

	// Size configures the max popup dimensions. If no value
	// is provided, a reasonable value is set.
	Size image.Point
	// TextSize configures the size the text displayed in the popup. If no value
	// is provided, a reasonable value is set.
	TextSize unit.Sp
}

func NewCompletionPopup(editor *gvcode.Editor, cmp gvcode.Completion) *CompletionPopup {
	return &CompletionPopup{
		editor: editor,
		cmp:    cmp,
	}
}

func (pop *CompletionPopup) Layout(gtx layout.Context, th *material.Theme, items []gvcode.CompletionCandidate) layout.Dimensions {
	pop.update(gtx)

	if !pop.cmp.IsActive() {
		pop.reset()
		return layout.Dimensions{}
	}

	border := widget.Border{
		Color:        adjustAlpha(th.Fg, 0xb0),
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(2),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max = pop.Size
		gtx.Constraints.Min = image.Point{
			X: gtx.Constraints.Max.X,
			Y: 0,
		}

		macro := op.Record(gtx.Ops)
		dims := pop.layout(gtx, th, items)
		callOp := macro.Stop()

		defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(unit.Dp(2))).Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, th.Bg)
		callOp.Add(gtx.Ops)
		return dims
	})
}

func (pop *CompletionPopup) updateSelection(direction int) {
	pop.list.ScrollBy(float32(direction))

	pop.labels[pop.focused].selected = false
	if direction < 0 {
		pop.focused = max(pop.focused+direction, 0)
	} else {
		pop.focused = min(pop.focused+direction, len(pop.labels)-1)
	}

	pop.labels[pop.focused].selected = true
}

func (pop *CompletionPopup) reset() {
	pop.cmp.Cancel()
	pop.focused = 0
	pop.labels = pop.labels[:0]
	pop.list.ScrollTo(0)
	pop.editor.RemoveCommands(pop)
}

func (pop *CompletionPopup) update(gtx layout.Context) {
	if pop.TextSize <= 0 {
		pop.TextSize = unit.Sp(12)
	}
	if pop.Size == (image.Point{}) {
		pop.Size = image.Point{
			X: gtx.Dp(unit.Dp(400)),
			Y: gtx.Dp(unit.Dp(200)),
		}
	}

	pop.editor.RegisterCommand(pop, key.Filter{Name: key.NameUpArrow, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
			pop.updateSelection(-1)
			return nil
		},
	)
	pop.editor.RegisterCommand(pop, key.Filter{Name: key.NameDownArrow, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
			pop.updateSelection(1)
			return nil
		},
	)
	pop.editor.RegisterCommand(pop, key.Filter{Name: key.NameEnter, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
			if pop.focused >= 0 {
				// simulate a click
				pop.labels[pop.focused].Click()
			}
			return nil
		},
	)

	pop.editor.RegisterCommand(pop, key.Filter{Name: key.NameReturn, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
			if pop.focused >= 0 {
				// simulate a click
				pop.labels[pop.focused].Click()
			}
			return nil
		},
	)

	pop.editor.RegisterCommand(pop, key.Filter{Name: key.NameEscape},
		func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
			pop.reset()
			return nil
		},
	)

	if pop.focused < len(pop.labels) {
		pop.labels[pop.focused].selected = true
	}

}

func (pop *CompletionPopup) layout(gtx layout.Context, th *material.Theme, items []gvcode.CompletionCandidate) layout.Dimensions {
	pop.list.Axis = layout.Vertical

	li := material.List(th, &pop.list)
	li.AnchorStrategy = material.Overlay
	li.ScrollbarStyle.Indicator.HoverColor = adjustAlpha(th.ContrastBg, 0xb0)
	li.ScrollbarStyle.Indicator.Color = adjustAlpha(th.ContrastBg, 0x30)
	return li.Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions {
		c := items[index]
		if len(pop.labels) <= index {
			pop.labels = append(pop.labels, &itemLabel{onClicked: func() {
				pop.cmp.OnConfirm(index)
				gtx.Execute(op.InvalidateCmd{})
			}})
		}

		return pop.labels[index].Layout(gtx, th, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:    unit.Dp(2),
				Bottom: unit.Dp(2),
				Left:   unit.Dp(6),
				Right:  unit.Dp(6),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
					Spacing:   layout.SpaceBetween,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Middle,
						}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Label(th, pop.TextSize*0.75, c.Kind).Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Label(th, pop.TextSize, c.Label).Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lb := material.Label(th, pop.TextSize*0.75, c.Description)
						lb.Color = adjustAlpha(th.Fg, 200)
						lb.MaxLines = 1
						return lb.Layout(gtx)
					}),
				)
			})
		})
	})
}

type itemLabel struct {
	state     widget.Clickable
	hovering  bool
	selected  bool
	onClicked func()
}

func (l *itemLabel) update(gtx layout.Context) bool {
	for {
		event, ok := gtx.Event(
			pointer.Filter{Target: l, Kinds: pointer.Enter | pointer.Leave},
		)
		if !ok {
			break
		}

		switch event := event.(type) {
		case pointer.Event:
			switch event.Kind {
			case pointer.Enter:
				l.hovering = true
			case pointer.Leave:
				l.hovering = false
			case pointer.Cancel:
				l.hovering = false
			}
		}
	}

	if l.state.Clicked(gtx) && l.onClicked != nil {
		l.onClicked()
		return true
	}

	return false
}

func (l *itemLabel) Click() {
	l.state.Click()
}

func (l *itemLabel) Layout(gtx layout.Context, th *material.Theme, w layout.Widget) layout.Dimensions {
	l.update(gtx)

	macro := op.Record(gtx.Ops)
	dims := layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			if !l.selected && !l.hovering {
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}

			var fill color.NRGBA
			if l.selected {
				fill = adjustAlpha(th.Palette.ContrastBg, 0xb6)
			} else if l.hovering {
				fill = adjustAlpha(th.Palette.ContrastBg, 0x30)
			}

			rect := clip.Rect{
				Max: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Min.Y},
			}
			paint.FillShape(gtx.Ops, fill, rect.Op())
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return l.state.Layout(gtx, w)
		},
	)
	call := macro.Stop()

	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, l)
	call.Add(gtx.Ops)

	return dims
}

func adjustAlpha(c color.NRGBA, alpha uint8) color.NRGBA {
	return color.NRGBA{
		R: c.R,
		G: c.G,
		B: c.B,
		A: alpha,
	}
}
