package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ----------------------------------------------------------------------------
// Scrollbar: tracks content vs. viewport height and converts thumb drags / wheel
// movement into a pixel scroll offset. It does not move geometry itself; it
// drives a *float32 offset that the owner (e.g. the editor, or a content
// container's ScrollOffset) reads.
// ----------------------------------------------------------------------------

// Axis selects a scrollbar's orientation.
type Axis int

const (
	Vertical Axis = iota
	Horizontal
)

type Scrollbar struct {
	host          *layout.Element // visual track: an absolute strip on one edge
	axis          Axis
	Offset        *float32 // owner-owned scroll position in pixels (along the axis)
	ContentHeight *float32 // owner-owned total content length along the axis, in pixels

	dragging bool
	hovered  bool
	grab     float32 // cursor-to-thumb offset captured at drag start (along axis)
}

// NewScrollbar creates a vertical scrollbar bound to the given offset/content
// pointers. width is the track thickness in pixels.
func NewScrollbar(offset, contentHeight *float32, width float32) *Scrollbar {
	return NewScrollbarAxis(Vertical, offset, contentHeight, width)
}

// NewScrollbarAxis creates a scrollbar for the given axis. The track pins itself
// to the right edge (vertical) or bottom edge (horizontal) of its parent;
// thickness is the cross-axis size in pixels. Offset/content are measured along
// the axis (height for vertical, width for horizontal).
func NewScrollbarAxis(axis Axis, offset, content *float32, thickness float32) *Scrollbar {
	s := &Scrollbar{axis: axis, Offset: offset, ContentHeight: content}
	if axis == Horizontal {
		// Strip across the bottom edge (leaving room for a vertical bar's corner
		// is the owner's concern).
		s.host = layout.New(layout.Box().H(thickness).AbsLeft(0).AbsRight(0).AbsBottom(0))
	} else {
		s.host = layout.New(layout.Box().W(thickness).AbsTop(0).AbsRight(0).AbsBottom(0))
	}
	s.host.Paint = s.paint
	s.host.OnMouse = s.onMouse
	return s
}

// scrollable reports whether content exceeds the track along the scroll axis.
func (s *Scrollbar) scrollable() bool {
	return *s.ContentHeight > s.trackLen()
}

// maxOffset is the maximum scroll offset in pixels.
func (s *Scrollbar) maxOffset() float32 {
	return f32max(0, *s.ContentHeight-s.trackLen())
}

// trackLen is the scrollbar's length along its scroll axis.
func (s *Scrollbar) trackLen() float32 {
	if s.axis == Horizontal {
		return s.host.Frame.W
	}
	return s.host.Frame.H
}

// thumb computes the thumb rectangle from the current track frame and offset.
func (s *Scrollbar) thumb() render.Rect {
	th := theme.Current()
	track := s.host.Frame
	ch := *s.ContentHeight
	along := s.trackLen()
	maxOff := f32max(0, ch-along)

	thumbLen := along
	if ch > along && ch > 0 {
		thumbLen = along * along / ch
	}
	thumbLen = clampf(thumbLen, th.Metrics.ScrollbarMinThumb, along)

	if s.axis == Horizontal {
		tx := track.X
		if maxOff > 0 {
			tx = track.X + (along-thumbLen)*(*s.Offset/maxOff)
		}
		return render.Rect{X: tx, Y: track.Y, W: thumbLen, H: track.H}
	}
	ty := track.Y
	if maxOff > 0 {
		ty = track.Y + (along-thumbLen)*(*s.Offset/maxOff)
	}
	return render.Rect{X: track.X, Y: ty, W: track.W, H: thumbLen}
}

// thumbVisual returns the thumb rectangle inset inside the track for drawing and
// hit-testing.
func (s *Scrollbar) thumbVisual() render.Rect {
	th := theme.Current()
	t := s.thumb()
	inset := th.Metrics.ScrollbarThumbInset
	if s.axis == Horizontal {
		return render.Rect{X: t.X, Y: t.Y + inset, W: t.W, H: t.H - 2*inset}
	}
	return render.Rect{X: t.X + inset, Y: t.Y, W: t.W - 2*inset, H: t.H}
}

// setOffsetFromPointer maps a pointer position along the track to a scroll offset.
func (s *Scrollbar) setOffsetFromPointer(px, py float32) {
	track := s.host.Frame
	t := s.thumb()
	along := s.trackLen()
	maxOff := s.maxOffset()
	if maxOff <= 0 {
		return
	}
	if s.axis == Horizontal {
		travel := along - t.W
		if travel <= 0 {
			return
		}
		*s.Offset = (px - s.grab - track.X) / travel * maxOff
	} else {
		travel := along - t.H
		if travel <= 0 {
			return
		}
		*s.Offset = (py - s.grab - track.Y) / travel * maxOff
	}
	*s.Offset = clampf(*s.Offset, 0, maxOff)
}

func (s *Scrollbar) onMouse(el *layout.Element, m *input.Mouse) {
	s.hovered = false
	if !s.scrollable() || !el.Frame.Contains(m.X, m.Y) {
		return
	}
	tv := s.thumbVisual()
	s.hovered = tv.Contains(m.X, m.Y)

	if m.Pressed {
		m.Consumed = true
		if tv.Contains(m.X, m.Y) {
			s.dragging = true
			if s.axis == Horizontal {
				s.grab = m.X - tv.X
			} else {
				s.grab = m.Y - tv.Y
			}
		} else {
			// Click on track: jump so the thumb center moves toward the click.
			raw := s.thumb()
			if s.axis == Horizontal {
				s.grab = raw.W / 2
				s.setOffsetFromPointer(m.X, m.Y)
			} else {
				s.grab = raw.H / 2
				s.setOffsetFromPointer(m.X, m.Y)
			}
		}
	}
	if !m.Down {
		s.dragging = false
	}
	if s.dragging && m.Down {
		s.setOffsetFromPointer(m.X, m.Y)
		m.Consumed = true
	}
}

// ApplyWheel consumes wheel deltas when the pointer is over area. Call from
// OnMouse (dispatch), not Layout: overlays are hit-tested front-to-back only
// during dispatch, so applying the wheel in Layout lets page scrollers behind
// a dialog steal the event.
func (s *Scrollbar) ApplyWheel(m *input.Mouse, area render.Rect) {
	if m == nil || !s.scrollable() || !area.Contains(m.X, m.Y) {
		return
	}
	if s.axis == Horizontal {
		if m.ScrollX != 0 {
			*s.Offset -= m.ScrollX * 3 * 14
			m.ScrollX = 0
			m.Consumed = true
		}
		return
	}
	if m.ScrollY != 0 {
		*s.Offset -= m.ScrollY * 3 * 14 // ~3 lines per wheel notch
		m.ScrollY = 0
		m.Consumed = true
	}
}

// Update processes thumb drag. Wheel input is handled by ApplyWheel during
// pointer dispatch so overlay widgets receive it before the page behind them.
func (s *Scrollbar) Update(m *input.Mouse, _ render.Rect) {
	if m == nil {
		return
	}
	if s.dragging && m.Down {
		s.setOffsetFromPointer(m.X, m.Y)
	}
	if !m.Down {
		s.dragging = false
	}
	*s.Offset = clampf(*s.Offset, 0, s.maxOffset())
}

func (s *Scrollbar) paint(dl *render.DrawList, _ *shape.Engine) {
	if !s.scrollable() {
		return
	}
	th := theme.Current()
	dl.AddRect(s.host.Frame, th.ScrollTrack)
	col := th.ScrollThumb
	if s.dragging || s.hovered {
		col = th.ScrollThumbHover
	}
	dl.AddRect(s.thumbVisual(), col)
}
