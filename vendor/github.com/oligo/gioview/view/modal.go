package view

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	cmp "gioui.org/x/component"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var (
	closeIcon, _    = widget.NewIcon(icons.NavigationClose)
	defaultMaxWidth = unit.Dp(600)
	defaultPadding  = layout.Inset{
		Top:    unit.Dp(15),
		Bottom: unit.Dp(25),
		Left:   unit.Dp(20),
		Right:  unit.Dp(20),
	}
)

type ModalView struct {
	View
	// padding between the modal border and inner widget. If it is not set
	// A default value will be used.
	Padding layout.Inset
	// The background color of the modal area.
	Background color.NRGBA
	// Maximum width of the modal area.
	MaxWidth unit.Dp
	// Maximum height of the modal in ratio relateive to the window.
	// The max value is restricted to 0.9 to prevent it overflow.
	MaxHeight float32
	Radius    unit.Dp
	// Make the ModalView halted. Halted modal view will not receive key events.
	Halted bool
	//position  f32.Point
	dims     layout.Dimensions
	closed   bool
	closeBtn widget.Clickable
	anim     *cmp.VisibilityAnimation
}

func (m *ModalView) IsClosed(gtx layout.Context) bool {
	return m.update(gtx)
}

func (m *ModalView) ShowUp(gtx layout.Context) {
	if m.anim == nil {
		m.anim = &cmp.VisibilityAnimation{
			State:    cmp.Invisible,
			Duration: time.Millisecond * 250,
		}
	}

	m.anim.Appear(gtx.Now)
}

func (m *ModalView) update(gtx layout.Context) bool {
	if m.closeBtn.Clicked(gtx) {
		m.closed = true
	}

	if m.View.Finished() {
		m.closed = true
	}

	if m.anim != nil && m.anim.Visible() && !m.Halted {
		for {
			// Use a global event filter to catch quit events.
			// Applications have to be very careful as they may also use global escape
			// elsewhere, which causes conflicts.
			event, ok := gtx.Event(key.Filter{Name: key.NameEscape})
			if !ok {
				break
			}

			ev, ok := event.(key.Event)
			if !ok {
				continue
			}

			if ev.Name == key.NameEscape && ev.State == key.Release {
				m.closed = true
			}
		}
	}

	return m.closed
}

func (m *ModalView) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	m.update(gtx)
	if !m.anim.Visible() {
		return layout.Dimensions{}
	}

	dims := layout.Dimensions{Size: gtx.Constraints.Min}

	contentOps := func() op.CallOp {
		macro := op.Record(gtx.Ops)
		m.dims = m.layoutView(gtx, th)
		return macro.Stop()
	}()

	max := gtx.Constraints.Max
	offset := image.Point{
		X: int(math.Round(float64((max.X - m.dims.Size.X) / 2))),
		Y: int(math.Round(float64((max.Y - m.dims.Size.Y) / 4))),
	}

	if m.anim.Animating() {
		offset.Y = int(float32(offset.Y) * m.anim.Revealed(gtx))
	}

	// Lay out a transparent scrim to block input to things beneath the
	// modal widget.
	suppressionScrim := func() op.CallOp {
		macro2 := op.Record(gtx.Ops)
		pr := clip.Rect(image.Rectangle{Min: image.Point{-1e6, -1e6}, Max: image.Point{1e6, 1e6}})
		stack := pr.Push(gtx.Ops)
		paint.ColorOp{Color: color.NRGBA{R: 86, G: 86, B: 86, A: 100}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		event.Op(gtx.Ops, m)
		stack.Pop()
		return macro2.Stop()
	}()
	op.Defer(gtx.Ops, suppressionScrim)

	modalAreaOps := func() op.CallOp {
		macro := op.Record(gtx.Ops)
		//draw at offset
		op.Offset(offset).Add(gtx.Ops)
		// draw the background
		modalArea := clip.UniformRRect(image.Rectangle{Max: image.Point{m.dims.Size.X, m.dims.Size.Y}}, gtx.Dp(m.Radius))
		stack := modalArea.Push(gtx.Ops)

		if m.Background != (color.NRGBA{}) {
			paint.ColorOp{Color: m.Background}.Add(gtx.Ops)
		} else {
			paint.ColorOp{Color: th.Bg}.Add(gtx.Ops)
		}
		paint.PaintOp{}.Add(gtx.Ops)

		contentOps.Add(gtx.Ops)
		stack.Pop()
		// offsetOp.Pop()
		return macro.Stop()
	}()
	op.Defer(gtx.Ops, modalAreaOps)

	return dims
}

func (m *ModalView) layoutView(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	if m.MaxWidth == 0 {
		m.MaxWidth = defaultMaxWidth
	}

	if m.Padding == (layout.Inset{}) {
		m.Padding = defaultPadding
	}

	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(m.MaxWidth))
	gtx.Constraints.Max.Y = int(float32(gtx.Constraints.Max.Y) * min(0.9, m.MaxHeight))
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	gtx.Constraints.Min.Y = 0

	return m.Padding.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return m.layoutHeader(gtx, th)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{
					Left:  unit.Dp(5),
					Right: unit.Dp(5),
				}.Layout(gtx, func(gtx C) D {
					return m.View.Layout(gtx, th)
				})
			}),
		)
	})
}

func (m *ModalView) layoutHeader(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			label := material.H6(th.Theme, m.View.Title())
			label.Color = cmp.WithAlpha(th.Fg, 0xb6)
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			return material.Clickable(gtx, &m.closeBtn, func(gtx C) D {
				return misc.Icon{Icon: closeIcon}.Layout(gtx, th)
			})
		}),
	)
}
