package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// BreadcrumbSegment is one clickable crumb.
type BreadcrumbSegment struct {
	Label    string
	OnSelect func()
}

type breadcrumbData struct {
	segments []BreadcrumbSegment
}

type breadcrumbState struct {
	hover int
}

// Breadcrumb is a horizontal navigation trail.
func Breadcrumb(id string, segments ...BreadcrumbSegment) *Node {
	return &Node{kind: kindBreadcrumb, id: id, extra: &breadcrumbData{segments: segments}}
}

func (n *Node) layoutBreadcrumb(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "crumb")
	}
	st := c.Widget(id, func() any { return &breadcrumbState{hover: -1} }).(*breadcrumbState)
	d, _ := n.extra.(*breadcrumbData)
	segs := []BreadcrumbSegment{}
	if d != nil {
		segs = d.segments
	}
	th := c.Theme()
	el := layout.New(applyLayoutSpec(layout.Box().H(th.Metrics.ControlHeight), n.spec))
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		paintBreadcrumb(dl, text, el.Frame, segs, st.hover)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		st.hover = -1
		if !e.Frame.Contains(m.X, m.Y) {
			return
		}
		locs := breadcrumbLayout(e.Frame, segs)
		for i, loc := range locs {
			if m.X >= loc.x && m.X <= loc.x+loc.w {
				st.hover = i
				m.SetCursor(CursorPointer)
				if m.Released && i < len(segs)-1 && segs[i].OnSelect != nil {
					segs[i].OnSelect()
				}
				return
			}
		}
	}
	return el
}

func breadcrumbLayout(f render.Rect, segs []BreadcrumbSegment) []struct{ x, w float32 } {
	th := theme.Current()
	style := th.Typography.Body
	chevW := th.Metrics.TreeChevronSize
	gap := th.Spacing.S
	out := make([]struct{ x, w float32 }, len(segs))
	x := f.X + th.Spacing.S
	eng := frameText()
	for i, seg := range segs {
		var tw float32
		if eng != nil {
			tw, _ = eng.MeasureAt(seg.Label, style.Size)
		}
		w := tw
		if i < len(segs)-1 {
			w += gap + chevW
		}
		out[i] = struct{ x, w float32 }{x: x, w: w}
		x += w + gap
	}
	return out
}

func paintBreadcrumb(dl *render.DrawList, text *shape.Engine, f render.Rect, segs []BreadcrumbSegment, hover int) {
	th := theme.Current()
	locs := breadcrumbLayout(f, segs)
	style := th.Typography.Body
	chevSz := th.Metrics.TreeChevronSize
	dl.PushClip(f)
	defer dl.PopClip()
	for i, seg := range segs {
		loc := locs[i]
		col := th.ForegroundMuted
		if i == len(segs)-1 {
			col = th.Foreground
		} else if i == hover {
			col = th.Accent
		}
		_, lh := text.MeasureAt(seg.Label, style.Size)
		text.DrawStringTopAt(dl, seg.Label, loc.x, f.Y+(f.H-lh)/2, col, style.Size)
		if i < len(segs)-1 {
			tw, _ := text.MeasureAt(seg.Label, style.Size)
			cx := loc.x + tw + th.Spacing.S
			cr := render.Rect{X: cx, Y: f.Y + (f.H-chevSz)/2, W: chevSz, H: chevSz}
			if sheet := frameIcons(); sheet != nil {
				sheet.Draw(dl, "chevron_right", cr, th.ForegroundSubtle)
			}
		}
	}
}
