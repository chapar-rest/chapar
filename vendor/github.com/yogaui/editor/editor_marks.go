package editor

import (
	"time"

	"github.com/yogaui/yoga/input"
	"github.com/yogaui/yoga/render"
)

// markHoverDelay is how long the pointer rests on a change bar before
// OnLineMark fires.
const markHoverDelay = 350 * time.Millisecond

// markSlop widens a change bar's hit area, which is only a few pixels wide.
const markSlop = 4

type markRow struct {
	line int
	y    float32
}

type markState struct {
	rows  []markRow // first rows painted last frame, top to bottom
	line  int       // the change bar under the pointer, or -1
	since time.Time
	fired bool
}

// markX is the left edge of the gutter's change bar column.
func (e *Editor) markX() float32 {
	foldW := float32(0)
	if !e.fold.off {
		foldW = foldGutterW
	}
	return e.viewport.Frame.X + e.gutterW - foldW - lineMarkW - 1
}

// lineMarkAt returns the line whose change bar is at the point, or -1.
func (e *Editor) lineMarkAt(x, y float32) int {
	if e.gutterW <= 0 || len(e.lineMarks) == 0 || !e.viewport.Frame.Contains(x, y) {
		return -1
	}
	if mx := e.markX(); x < mx-markSlop || x > mx+lineMarkW+markSlop {
		return -1
	}
	for _, r := range e.marks.rows {
		if y >= r.y && y < r.y+e.lineH {
			if _, ok := e.lineMarks[r.line]; ok {
				return r.line
			}
			return -1
		}
	}
	return -1
}

// LineRect is the screen rect of line's first row across the editor, from
// the last frame painted; false when the line is not on screen.
func (e *Editor) LineRect(line int) (render.Rect, bool) {
	f := e.viewport.Frame
	for _, r := range e.marks.rows {
		if r.line == line {
			return render.Rect{X: f.X, Y: r.y, W: f.W, H: e.lineH}, true
		}
	}
	return render.Rect{}, false
}

// lineMarkMouse runs OnLineMark for a click on a change bar.
func (e *Editor) lineMarkMouse(m *input.Mouse) bool {
	if e.OnLineMark == nil || !m.Pressed || m.Consumed {
		return false
	}
	line := e.lineMarkAt(m.X, m.Y)
	if line < 0 {
		return false
	}
	m.Consumed = true
	e.marks.fired = true
	e.OnLineMark(line)
	return true
}

// trackMarkHover runs OnLineMark once the pointer rests on a change bar.
func (e *Editor) trackMarkHover(m *input.Mouse) {
	if e.OnLineMark == nil || m == nil {
		return
	}
	line := e.lineMarkAt(m.X, m.Y)
	if line != e.marks.line {
		e.marks.line, e.marks.since, e.marks.fired = line, time.Now(), false
		return
	}
	if line >= 0 && !e.marks.fired && time.Since(e.marks.since) >= markHoverDelay {
		e.marks.fired = true
		e.OnLineMark(line)
	}
}

// markWake is the time left before a resting pointer fires OnLineMark.
func (e *Editor) markWake() (time.Duration, bool) {
	if e.OnLineMark == nil || e.marks.line < 0 || e.marks.fired {
		return 0, false
	}
	return max(0, markHoverDelay-time.Since(e.marks.since)), true
}
