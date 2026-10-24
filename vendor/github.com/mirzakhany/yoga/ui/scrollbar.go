package ui

import (
	"time"

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

// WheelPixelsPerUnit is how far one wheel unit scrolls: a mouse notch reports
// ±1 and moves ~3 lines. Runtimes whose platform reports pixel-precise deltas
// (trackpads) divide those pixels by this to reach the same unit.
const WheelPixelsPerUnit = 3 * 14

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

	// Mode override; when modeSet is false the bar follows the theme.
	mode    theme.ScrollbarMode
	modeSet bool

	// Overlay fade state, advanced by tick once per frame.
	alpha     float32   // 0 hidden … 1 fully shown
	activeAt  time.Time // last scroll or pointer move over the scroll area
	lastOff   float32
	lastMX    float32
	lastMY    float32
	seen      bool // lastOff / lastMX / lastMY hold a previous frame's values
	markPaint func()
}

// Overlay timing: the thumb stays up for overlayHold after the last activity,
// then fades out over overlayFade.
const (
	overlayHold = 800 * time.Millisecond
	overlayFade = 200 * time.Millisecond
)

// overlayPad is the gap between an overlay thumb and the edge it hugs.
const overlayPad = 2

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

// SetMode overrides the theme's ScrollbarMode for this bar.
func (s *Scrollbar) SetMode(m theme.ScrollbarMode) { s.mode, s.modeSet = m, true }

// FollowTheme drops a SetMode override.
func (s *Scrollbar) FollowTheme() { s.modeSet = false }

// overlay reports whether the bar draws over the content (ScrollOverlay).
func (s *Scrollbar) overlay() bool {
	if s.modeSet {
		return s.mode == theme.ScrollOverlay
	}
	return theme.Current().Metrics.ScrollbarMode == theme.ScrollOverlay
}

// overlaySize is the resting thumb thickness of an overlay bar.
func overlaySize() float32 {
	if n := theme.Current().Metrics.ScrollbarOverlaySize; n > 0 {
		return n
	}
	return 6
}

// gutter is the space the owner reserves for the bar beside its content:
// classic in ScrollAlways mode, nothing for an overlay bar.
func (s *Scrollbar) gutter(classic float32) float32 {
	if s.overlay() {
		return 0
	}
	return classic
}

// strip is the bar's cross-axis size: classic, or for an overlay bar room for
// the widened thumb plus its edge gap.
func (s *Scrollbar) strip(classic float32) float32 {
	if s.overlay() {
		return overlaySize() + 4 + 2*overlayPad
	}
	return classic
}

// hit reports whether (x, y) is on the bar as it is drawn now. An overlay bar
// that has faded out lets the pointer through to the content below.
func (s *Scrollbar) hit(x, y float32) bool {
	if s.overlay() && s.alpha <= 0 && !s.dragging {
		return false
	}
	return s.host.Frame.Contains(x, y)
}

// tick advances the overlay fade. Owners call it once per frame with the
// area their content scrolls in; a scroll or pointer move there shows the
// thumb, and it hides again after overlayHold of quiet.
func (s *Scrollbar) tick(c *Ctx, area render.Rect) {
	if c == nil {
		return
	}
	s.markPaint = c.MarkNeedsPaint
	if !s.overlay() {
		s.alpha = 1
		return
	}
	now := c.Now()
	m := c.Mouse()
	if s.seen {
		if *s.Offset != s.lastOff {
			s.activeAt = now
		}
		if m != nil && (m.X != s.lastMX || m.Y != s.lastMY) && area.Contains(m.X, m.Y) {
			s.activeAt = now
		}
	}
	s.lastOff = *s.Offset
	if m != nil {
		s.lastMX, s.lastMY = m.X, m.Y
	}
	s.seen = true

	prev := s.alpha
	held := s.dragging || s.hovered
	idle := now.Sub(s.activeAt)
	switch {
	case !s.scrollable():
		s.alpha = 0
	case held || idle < overlayHold:
		s.alpha = 1
	case idle < overlayHold+overlayFade:
		s.alpha = 1 - float32(idle-overlayHold)/float32(overlayFade)
	default:
		s.alpha = 0
	}
	if s.alpha != prev {
		c.MarkNeedsPaint()
	}
	if s.alpha > 0 && !held {
		if idle < overlayHold {
			c.Animate(overlayHold - idle)
		} else {
			c.Animate(16 * time.Millisecond)
		}
	}
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
	wasHovered := s.hovered
	s.hovered = false
	defer func() {
		if s.hovered != wasHovered && s.overlay() && s.markPaint != nil {
			s.markPaint()
		}
	}()
	if !s.scrollable() || !s.hit(m.X, m.Y) {
		if !m.Down {
			s.dragging = false
		}
		return
	}
	tv := s.thumbVisual()
	if s.overlay() {
		// Widen when the pointer is anywhere on the strip, and grab the
		// thumb along its full cross-axis span so a thin thumb is easy to hit.
		s.hovered = true
		tv = s.thumb()
	} else {
		s.hovered = tv.Contains(m.X, m.Y)
	}

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
			*s.Offset -= m.ScrollX * WheelPixelsPerUnit
			m.ScrollX = 0
			m.Consumed = true
		}
		return
	}
	if m.ScrollY != 0 {
		*s.Offset -= m.ScrollY * WheelPixelsPerUnit
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
	if s.overlay() {
		s.paintOverlay(dl, th)
		return
	}
	dl.AddRect(s.host.Frame, th.ScrollTrack)
	col := th.ScrollThumb
	if s.dragging || s.hovered {
		col = th.ScrollThumbHover
	}
	dl.AddRect(s.thumbVisual(), col)
}

// paintOverlay draws a track-less rounded thumb hugging the far edge, faded by
// alpha and widened while hovered or dragged.
func (s *Scrollbar) paintOverlay(dl *render.DrawList, th *theme.Theme) {
	if s.alpha <= 0 {
		return
	}
	w := overlaySize()
	if s.hovered || s.dragging {
		w += 4
	}
	t := s.thumb()
	r := render.Rect{X: t.X + t.W - overlayPad - w, Y: t.Y, W: w, H: t.H}
	if s.axis == Horizontal {
		r = render.Rect{X: t.X, Y: t.Y + t.H - overlayPad - w, W: t.W, H: w}
	}
	col := th.ScrollThumb
	col.A *= s.alpha
	dl.AddRoundedRect(r, w/2, col)
}
