package yoga

// Zoom limits for SetZoom.
const (
	MinZoom float32 = 0.5
	MaxZoom float32 = 3
)

// zoom is the window's UI zoom, and a counter bumped on each change that the
// frame loop compares against to apply it.
var zoom = struct {
	factor float32
	gen    uint64
	wake   func()
}{factor: 1}

// SetZoom scales the whole window, text, icons, spacing and hit areas alike,
// by f (1 is unzoomed), clamped to [MinZoom, MaxZoom]. Layout sees a window
// 1/f the size, so the app needs no changes; text is re-baked at the new
// scale and stays sharp. It takes effect on the next frame. Call it on the UI
// goroutine.
func SetZoom(f float32) {
	if f != f { // NaN
		return
	}
	if f < MinZoom {
		f = MinZoom
	} else if f > MaxZoom {
		f = MaxZoom
	}
	if f == zoom.factor {
		return
	}
	zoom.factor = f
	zoom.gen++
	if zoom.wake != nil {
		zoom.wake()
	}
}

// Zoom returns the factor set by SetZoom.
func Zoom() float32 { return zoom.factor }
