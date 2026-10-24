package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

type rangeData struct {
	start, end, total float32
	style             badgeData // Tone / Hue
}

// RangeBar draws the span [start, end] of total as a filled segment on a
// track: a timing waterfall's phase, a quota's used part, a diff's range.
// Color the segment with Tone (status colors) or Hue (categorical); the
// default is the accent. Give it a Tooltip to explain the span.
//
//	ui.RangeBar("ttfb", 65, 166, 179).Tone(ui.BadgeWarning).Tooltip("Waiting (TTFB) 101 ms")
func RangeBar(id string, start, end, total float32) *Node {
	return &Node{kind: kindRangeBar, id: id, extra: &rangeData{start: start, end: end, total: total}}
}

// fraction maps the span to [0, 1] fractions of the track, keeping
// them in order and inside it.
func (d *rangeData) fraction() (lo, hi float32) {
	if d.total <= 0 {
		return 0, 0
	}
	lo = clampf(d.start/d.total, 0, 1)
	hi = clampf(d.end/d.total, 0, 1)
	if hi < lo {
		lo, hi = hi, lo
	}
	return lo, hi
}

// rangeFill is the segment color for a RangeBar's tone or hue.
func rangeFill(th *theme.Theme, d *badgeData) render.Color {
	if d.hasHue {
		return th.Categorical[d.hue]
	}
	switch d.tone {
	case BadgeSuccess:
		return th.Success
	case BadgeWarning:
		return th.Warning
	case BadgeError:
		return th.Error
	}
	return th.Accent // BadgeAccent, and no tone at all
}

func (n *Node) layoutRangeBar(c *Ctx) *layout.Element {
	th := c.Theme()
	d, _ := n.extra.(*rangeData)
	if d == nil {
		d = &rangeData{}
	}
	style := &d.style
	w, h := float32(200), float32(6)
	if n.spec.hasW {
		w = n.spec.width
	}
	if n.spec.hasH {
		h = n.spec.height
	}
	// Unlike a progress bar it gives up width in a crowded row, so the
	// labels beside it in a waterfall stay visible.
	el := layout.New(applyLayoutSpec(layout.Box().W(w).H(h).FlexShrink(1).Min(0, h), n.spec))
	lo, hi := d.fraction()
	fill := rangeFill(th, style)
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		f := el.Frame
		r := f.H / 2
		dl.AddRoundedRect(f, r, th.ChromeMuted)
		if d.total <= 0 {
			return
		}
		// A span too short to see still shows as a sliver.
		seg := render.Rect{X: f.X + f.W*lo, Y: f.Y, W: f32max(f.W*(hi-lo), f32min(2, f.W)), H: f.H}
		if seg.X+seg.W > f.X+f.W {
			seg.X = f.X + f.W - seg.W
		}
		dl.AddRoundedRect(seg, f32min(r, seg.W/2), fill)
	}
	return el
}
