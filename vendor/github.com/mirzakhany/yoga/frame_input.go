package yoga

import "github.com/mirzakhany/yoga/input"

// inputImpliesPaint reports whether this frame's input changed what is on
// screen. It covers the coarse cases (click, scroll, typing, drag); hover
// transitions mark themselves via trackHover in the widgets.
//
// scrolled, typed and dragged must be sampled *before* dispatch. The widget
// that handles a wheel delta or a key zeroes it as it consumes it — see
// ui.Scrollbar.ApplyWheel — so reading them from the mouse afterwards reports
// no input on exactly the frames that scrolled or typed, and the frame is
// never repainted.
func inputImpliesPaint(m *input.Mouse, scrolled, typed, dragged bool) bool {
	return m.Pressed || m.Released ||
		m.RightPressed || m.RightReleased ||
		scrolled || typed || dragged
}
