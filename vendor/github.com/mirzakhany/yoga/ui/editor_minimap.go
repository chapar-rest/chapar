package ui

import (
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// The minimap is a strip at the editor's right edge that draws every line
// as thin bars in its syntax colors, with the lines on screen shaded.
// Clicking or dragging it scrolls. It takes its width from the text area,
// so soft wrap wraps short of it.

const (
	minimapW     = 100 // strip width
	minimapLineH = 3   // pixels per line
	minimapBarH  = 2   // a line's bar height, leaving a gap below it
	minimapPad   = 6   // left inset of the bars
	minimapMinW  = 480 // narrower editors hide it, leaving the text the room
)

// SetMinimap shows or hides this editor's minimap, whatever
// EditorDefaults.Minimap says. Editors built WithoutGutter never show one.
func (e *Editor) SetMinimap(on bool) { e.minimapSet, e.minimapOn = true, on }

// Minimap reports whether this editor asks for a minimap; it shows when
// the editor is wide enough.
func (e *Editor) Minimap() bool {
	if e.noGutter {
		return false
	}
	if e.minimapSet {
		return e.minimapOn
	}
	return EditorDefaults.Minimap
}

// minimapWidth is the strip's width this frame, 0 when hidden.
func (e *Editor) minimapWidth() float32 {
	if !e.Minimap() || e.viewport == nil || e.viewport.Frame.W < minimapMinW {
		return 0
	}
	return minimapW
}

// minimapGeom is the strip's rect and how far its lines are scrolled, in
// minimap pixels. A document taller than the strip scrolls it in step with
// the editor, so the first line shows at the top and the last at the bottom.
func (e *Editor) minimapGeom() (r render.Rect, top float32) {
	cw, ch, _, _ := e.scrollMetrics()
	f := e.viewport.Frame
	r = render.Rect{X: f.X + cw, Y: f.Y, W: e.minimapWidth(), H: ch}
	if over := float32(e.pt.LineCount())*minimapLineH - r.H; over > 0 {
		if maxScroll := e.ContentHeight - ch; maxScroll > 0 {
			top = over * clampf(e.ScrollPx/maxScroll, 0, 1)
		}
	}
	return r, top
}

func (e *Editor) paintMinimap(dl *render.DrawList, th *theme.Theme) {
	if e.minimapWidth() == 0 {
		return
	}
	r, top := e.minimapGeom()
	dl.PushClip(r)
	defer dl.PopClip()
	dl.AddRect(r, th.Background)
	dl.AddRect(render.Rect{X: r.X, Y: r.Y, W: 1, H: r.H}, th.Border)

	// The lines on screen.
	_, ch, _, _ := e.scrollMetrics()
	firstRow := int(e.ScrollPx / e.lineH)
	lastRow := int((e.ScrollPx + ch - 1) / e.lineH)
	first := e.rowLine(firstRow)
	last := e.rowLine(min(lastRow, e.lineRowCount()-1))
	shade := th.Foreground
	shade.A *= 0.08
	dl.AddRect(render.Rect{
		X: r.X + 1, Y: r.Y + float32(first)*minimapLineH - top,
		W: r.W - 1, H: max(float32(last-first+1)*minimapLineH, minimapLineH),
	}, shade)

	tabW := e.TabWidth()
	cols := int(r.W - minimapPad) // a column per pixel
	lc := e.pt.LineCount()
	ln := int(top / minimapLineH)
	hint := -1
	for y := r.Y + float32(ln)*minimapLineH - top; ln < lc && y < r.Y+r.H; ln, y = ln+1, y+minimapLineH {
		ls := e.pt.LineStart(ln)
		// Read no more of a line than fits: a minified file's one line can
		// be megabytes.
		line := e.pt.LineSlice(ln, 0, cols*4)
		if hint < 0 {
			hint = tokenIndexAt(e.tokens, ls)
		}
		col, runStart, runCol := 0, -1, 0
		var runClass highlight.ColorClass
		flush := func(end int) {
			if runStart < 0 {
				return
			}
			c := th.SyntaxColor(runClass)
			c.A *= 0.75
			dl.AddRect(render.Rect{X: r.X + minimapPad + float32(runCol), Y: y, W: float32(col - runCol), H: minimapBarH}, c)
			runStart = -1
		}
		for i := 0; i < len(line) && col < cols; {
			b := line[i]
			switch b {
			case ' ':
				flush(i)
				col++
				i++
				continue
			case '\t':
				flush(i)
				col += tabW - col%tabW
				i++
				continue
			}
			cls, h := e.colorAtHint(ls+i, hint)
			hint = h
			if runStart >= 0 && cls != runClass {
				flush(i)
			}
			if runStart < 0 {
				runStart, runCol, runClass = i, col, cls
			}
			// One column per rune: skip a multi-byte rune's tail.
			i++
			for i < len(line) && line[i]&0xC0 == 0x80 {
				i++
			}
			col++
		}
		flush(len(line))
	}
}

// minimapMouse scrolls from a press or drag on the strip and reports
// whether it took the event. A document that fits shows each line where it
// is, so a click centers that line; a taller one maps the strip's height to
// the whole scroll range, which holds still under a drag.
func (e *Editor) minimapMouse(m *input.Mouse) bool {
	if e.minimapDrag && !m.Down {
		e.minimapDrag = false
		return true
	}
	if e.minimapWidth() == 0 {
		e.minimapDrag = false
		return false
	}
	r, _ := e.minimapGeom()
	if m.Pressed && r.Contains(m.X, m.Y) && !m.Consumed {
		e.minimapDrag = true
		m.Consumed = true
	}
	if !e.minimapDrag {
		return false
	}
	_, ch, _, _ := e.scrollMetrics()
	maxScroll := e.ContentHeight - ch
	if float32(e.pt.LineCount())*minimapLineH <= r.H {
		ln := clampInt(int((m.Y-r.Y)/minimapLineH), 0, e.pt.LineCount()-1)
		row := e.rowOfByte(e.pt.LineStart(ln))
		e.ScrollPx = float32(row)*e.lineH - ch/2 + e.lineH/2
	} else {
		e.ScrollPx = clampf((m.Y-r.Y)/r.H, 0, 1) * maxScroll
	}
	e.clampScroll()
	return true
}
