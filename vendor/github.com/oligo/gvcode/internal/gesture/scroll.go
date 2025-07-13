package scroll

import (
	"math"
	"runtime"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/oligo/gvcode/internal/gesture/fling"
)

// Scroll detects scroll gestures and reduces them to
// scroll distances. Scroll recognizes mouse wheel
// movements as well as drag and fling touch gestures.
//
// This is a modified version of the original [gesture.Scroll] in Gio.
// The most important change is that scrolling axis is detected, not
// passed by user.
type Scroll struct {
	dragging  bool
	estimator fling.Extrapolation
	flinger   fling.Animation
	pid       pointer.ID
	last      int
	// Leftover scroll.
	scroll float32
	// Position of the initial pointer.Press.
	initialPos f32.Point
	// The initial pointer press duration.
	initialPosTime time.Duration
	// The determined axis for the current drag gesture or wheel scroll.
	scrollAxis Axis
	// True if the axis for the current gesture/scrolling has been determined.
	axisLocked bool
}

type ScrollState uint8

type Axis uint8

const (
	Horizontal Axis = iota
	Vertical
)

const (
	// StateIdle is the default scroll state.
	StateIdle ScrollState = iota
	// StateDragging is reported during drag gestures.
	StateDragging
	// StateFlinging is reported when a fling is
	// in progress.
	StateFlinging
)

const touchSlop = unit.Dp(3)

// Add the handler to the operation list to receive scroll events.
// The bounds variable refers to the scrolling boundaries
// as defined in [pointer.Filter].
func (s *Scroll) Add(ops *op.Ops) {
	event.Op(ops, s)
}

// Stop any remaining fling movement.
func (s *Scroll) Stop() {
	s.flinger = fling.Animation{}
}

// Direction returns the last scrolling axis detected by Update.
func (s *Scroll) Direction() Axis {
	// if s.axisLocked || s.flinger.Active() {
	// 	return s.scrollAxis
	// }
	// slog.Info("returning default direction", "direction", Vertical)
	// return Vertical
	return s.scrollAxis
}

// Update state and report the scroll distance along axis.
func (s *Scroll) Update(cfg unit.Metric, q input.Source, t time.Time, scrollx, scrolly pointer.ScrollRange) int {
	total := 0
	f := pointer.Filter{
		Target:  s,
		Kinds:   pointer.Press | pointer.Drag | pointer.Release | pointer.Scroll | pointer.Cancel,
		ScrollX: scrollx,
		ScrollY: scrolly,
	}
	for {
		evt, ok := q.Event(f)
		if !ok {
			break
		}
		e, ok := evt.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Kind {
		case pointer.Press:
			if s.dragging {
				break
			}
			// Only scroll on touch drags, or on Android where mice
			// drags also scroll by convention.
			if e.Source != pointer.Touch && runtime.GOOS != "android" {
				break
			}

			s.Stop()
			s.initialPos = e.Position // Store initial position
			s.initialPosTime = e.Time
			s.dragging = true
			s.pid = e.PointerID
			// Reset axis lock for the new gesture
			s.axisLocked = false
			// Reset estimator
			s.estimator = fling.Extrapolation{}
		case pointer.Release:
			if s.pid != e.PointerID {
				break
			}
			fling := s.estimator.Estimate()
			if slop, d := float32(cfg.Dp(touchSlop)), fling.Distance; d < -slop || d > slop {
				s.flinger.Start(cfg, t, fling.Velocity)
			}
			fallthrough
		case pointer.Cancel:
			s.dragging = false
			//s.axisLocked = false
		case pointer.Scroll:
			if e.Modifiers.Contain(key.ModShift) {
				s.scrollAxis = Horizontal
			} else {
				s.scrollAxis = Vertical
			}
			s.axisLocked = true

			switch s.scrollAxis {
			case Horizontal:
				s.scroll += e.Scroll.X
			case Vertical:
				s.scroll += e.Scroll.Y
			}
			iscroll := int(s.scroll)
			s.scroll -= float32(iscroll)
			total += iscroll
		case pointer.Drag:
			if !s.dragging || s.pid != e.PointerID {
				continue
			}

			var scrollDelta int

			if !s.axisLocked {
				deltaX := e.Position.X - s.initialPos.X
				deltaY := e.Position.Y - s.initialPos.Y
				slopVal := float32(cfg.Dp(touchSlop))
				absDeltaX := float32(math.Abs(float64(deltaX)))
				absDeltaY := float32(math.Abs(float64(deltaY)))

				// If slop overcome
				if absDeltaX > slopVal || absDeltaY > slopVal {
					if absDeltaX > absDeltaY {
						s.scrollAxis = Horizontal
					} else {
						s.scrollAxis = Vertical
					}
					s.axisLocked = true
					q.Execute(pointer.GrabCmd{Tag: s, ID: e.PointerID})

					val := s.val(s.scrollAxis, e.Position)
					initialVal := s.val(s.scrollAxis, s.initialPos)
					s.last = int(math.Round(float64(val)))
					// Sample the initial point (or current point) for the determined axis
					// The estimator needs to be fed values along the chosen axis.
					s.estimator.Sample(s.initialPosTime, initialVal)
					// And also sample the current point on this axis.
					s.estimator.Sample(e.Time, val)

					// Calculate initial scroll amount from press to current
					scrollDelta = int(math.Round(float64(initialVal))) - s.last
				}
			} else {
				val := s.val(s.scrollAxis, e.Position)
				s.estimator.Sample(e.Time, val)

				v := int(math.Round(float64(val)))
				scrollDelta = s.last - v
				s.last = v
			}

			total += scrollDelta
		}
	}

	total += s.flinger.Tick(t)
	if s.flinger.Active() || (s.dragging && s.axisLocked) {
		q.Execute(op.InvalidateCmd{})
	}

	return total
}

func (s *Scroll) val(axis Axis, p f32.Point) float32 {
	switch axis {
	case Horizontal:
		return p.X
	case Vertical:
		return p.Y
	default:
		return 0
	}
}

func (a Axis) String() string {
	switch a {
	case Horizontal:
		return "Horizontal"
	case Vertical:
		return "Vertical"
	default:
		panic("invalid Axis")
	}
}

// State reports the scroll state.
func (s *Scroll) State() ScrollState {
	switch {
	case s.flinger.Active():
		return StateFlinging
	case s.dragging:
		return StateDragging
	default:
		return StateIdle
	}
}

func (s ScrollState) String() string {
	switch s {
	case StateIdle:
		return "StateIdle"
	case StateDragging:
		return "StateDragging"
	case StateFlinging:
		return "StateFlinging"
	default:
		panic("unreachable")
	}
}
