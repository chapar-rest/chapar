package scroll

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The duration for detecting a pointer as a hover.
const (
	hoverDuration = 200 * time.Millisecond
	hoverSlop     = unit.Dp(8)
)

// Hover is a gesture to detect if a pointer keeps still
// for some time over the area.
type Hover struct {
	entered    bool
	enteredAt  time.Duration
	startPos   f32.Point
	isHovering bool
	pid        pointer.ID
}

type HoverKind uint8

const (
	KindHovered HoverKind = iota + 1
	KindCancelled
)

type HoverEvent struct {
	Kind     HoverKind
	Position image.Point
}

// Add the gesture to detect hovering over the current pointer area.
func (h *Hover) Add(ops *op.Ops) {
	event.Op(ops, h)
}

func (h *Hover) Hovering() bool {
	return h.isHovering
}

// Update state and report whether a pointer is hovering over the area.
// The return value indicates if the hover state just started or canceled
// in this update cycle. Use Hovering() for the continuous state.
func (h *Hover) Update(gtx layout.Context) (HoverEvent, bool) {
	var hoverEvent HoverEvent

	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: h,
			Kinds:  pointer.Enter | pointer.Move | pointer.Press | pointer.Scroll | pointer.Leave | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Kind {
		case pointer.Leave, pointer.Cancel:
			if h.entered && h.pid == e.PointerID {
				h.entered = false
				h.enteredAt = 0
				h.isHovering = false
				h.startPos = f32.Point{}
			}
		case pointer.Enter:
			if !h.entered {
				h.pid = e.PointerID
			}
			if h.pid == e.PointerID {
				h.entered = true
				h.enteredAt = e.Time
				h.isHovering = false
				h.startPos = e.Position
			}
		case pointer.Move:
			if !h.entered || h.pid != e.PointerID {
				break
			}

			diff := e.Position.Sub(h.startPos)
			slop := gtx.Dp(hoverSlop)
			moved := diff.X*diff.X+diff.Y*diff.Y > float32(slop*slop)

			// If hover is already active, this Move event doesn't re-trigger
			// the "just started" signal.
			if h.isHovering {
				if moved {
					h.isHovering = false
					// Reset timer and start position so hover can re-trigger
					// if the pointer becomes still again from this new position.
					h.enteredAt = e.Time
					h.startPos = e.Position
					hoverEvent = HoverEvent{Kind: KindCancelled}
				}

				// Whether it was cancelled or continued hovering,
				// an already active hover doesn't re-trigger the "just started" signal.
				break
			}

			if moved {
				h.enteredAt = e.Time
				h.startPos = e.Position
				break
			}

			// If still within slop, check duration for hover activation
			if e.Time-h.enteredAt > hoverDuration {
				h.isHovering = true
				// Re-anchor startPos to the current position upon activation. Future slop
				// checks for an active hover are relative to this activation point.
				h.startPos = e.Position
				hoverEvent = HoverEvent{Kind: KindHovered, Position: e.Position.Round()}
				return hoverEvent, true
			}
		case pointer.Press, pointer.Scroll:
			if !h.entered || h.pid != e.PointerID {
				break
			}

			if h.isHovering {
				h.isHovering = false
				// Reset timer and start position so hover can re-trigger
				// if the pointer becomes still again from this new position.
				h.enteredAt = e.Time
				h.startPos = e.Position
				hoverEvent = HoverEvent{Kind: KindCancelled}
			}

		}
	}

	activated := hoverEvent != (HoverEvent{})
	return hoverEvent, activated
}
