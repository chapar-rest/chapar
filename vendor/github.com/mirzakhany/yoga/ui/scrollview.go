package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ScrollView is a clipped vertical scroll container for tall content.
//
// Structure: the host holds two children — an inner viewport element that carries
// ScrollOffset (so only the content moves) and the scrollbar as a sibling that
// stays pinned to the right edge. The viewport has Clip set, so the engine
// scissors scrolled-out content during Paint and skips it during Dispatch.
type ScrollView struct {
	host     *layout.Element
	Content  *layout.Element
	viewEl   *layout.Element // scrolling viewport: carries ScrollOffset + Clip
	vbar     *Scrollbar
	scrollY  float32
	contentH float32
}

// NewScrollView wraps content in a vertically scrollable viewport.
func NewScrollView(content *layout.Element) *ScrollView {
	th := theme.Current()
	sv := &ScrollView{}
	bar := th.Metrics.ScrollbarSize
	sv.vbar = NewScrollbarAxis(Vertical, &sv.scrollY, &sv.contentH, bar)

	// Basis 0 + min 0: scroll panes must not publish content height as their
	// flex basis, or a tall page forces ancestors to overflow and shrink chrome
	// (top bars, sidebars). Same idea as CSS flex min-height:0 on overflow:auto.
	sv.viewEl = layout.New(layout.Box().FlexGrow(1).FlexBasis(0).Min(0, 0))
	sv.viewEl.Clip = true
	sv.setContent(content)

	sv.host = layout.New(layout.Box().FlexGrow(1).FlexBasis(0).Min(0, 0), sv.viewEl, sv.vbar.host)
	sv.host.Paint = sv.paint
	sv.host.OnMouse = sv.onMouse
	// Re-clamp after measure so a shorter content tree (e.g. page switch) does
	// not paint with the previous tall contentH / scrollY until the next click.
	sv.host.AfterLayout = sv.afterLayout
	return sv
}

func (sv *ScrollView) setContent(content *layout.Element) {
	if sv.Content != nil {
		if _, h := sv.Content.LayoutSize(); h > 0 {
			sv.contentH = h
		} else if sv.Content.Frame.H > 0 {
			sv.contentH = sv.Content.Frame.H
		}
	}
	sv.Content = content
	if content != nil {
		content.Style.Shrink = 0
		sv.viewEl.Children = []*layout.Element{content}
	} else {
		sv.viewEl.Children = nil
	}
}

func (sv *ScrollView) contentHeight() float32 {
	if sv.Content != nil {
		if _, h := sv.Content.LayoutSize(); h > 0 {
			sv.contentH = h
		} else if sv.Content.Frame.H > 0 {
			sv.contentH = sv.Content.Frame.H
		}
	}
	return sv.contentH
}

// hostHeight prefers solver size (valid after layoutNode / during AfterLayout)
// and falls back to Frame after Flatten.
func (sv *ScrollView) hostHeight() float32 {
	if _, h := sv.host.LayoutSize(); h > 0 {
		return h
	}
	return sv.host.Frame.H
}

// viewport is the content area: the element frame minus the scrollbar strip
// when the bar is visible.
func (sv *ScrollView) viewport() render.Rect {
	th := theme.Current()
	f := sv.host.Frame
	bar := th.Metrics.ScrollbarSize
	w := f.W
	if sv.contentHeight() > f.H {
		w = f.W - bar
	}
	return render.Rect{X: f.X, Y: f.Y, W: w, H: f.H}
}

// syncScroll clamps offset and updates scrollbar gutter. Returns true when the
// viewport right margin changed (caller may need another layout pass).
func (sv *ScrollView) syncScroll() bool {
	th := theme.Current()
	ch := sv.contentHeight()
	if sv.vbar.ContentHeight != nil {
		*sv.vbar.ContentHeight = ch
	}
	hostH := sv.hostHeight()
	bar := th.Metrics.ScrollbarSize
	vShow := ch > hostH
	prevMargin := sv.viewEl.Style.Margin.Right
	if vShow {
		sv.vbar.host.Style = layout.Box().W(bar).AbsTop(0).AbsRight(0).AbsBottom(0)
		sv.vbar.host.ReapplyStyle()
		sv.viewEl.Style.Margin.Right = bar
	} else {
		sv.viewEl.Style.Margin.Right = 0
	}
	maxOff := f32max(0, ch-hostH)
	sv.scrollY = clampf(sv.scrollY, 0, maxOff)
	sv.viewEl.ScrollOffset = sv.scrollY
	return sv.viewEl.Style.Margin.Right != prevMargin
}

func (sv *ScrollView) afterLayout(_ *layout.Element) bool {
	return sv.syncScroll()
}

func (sv *ScrollView) paint(dl *render.DrawList, _ *shape.Engine) {
	th := theme.Current()
	dl.AddRect(sv.host.Frame, th.Surface)
}

func (sv *ScrollView) onMouse(e *layout.Element, m *input.Mouse) {
	if m == nil || e == nil {
		return
	}
	if !e.Frame.Contains(m.X, m.Y) {
		return
	}
	sv.syncScroll()
	if m.ScrollY != 0 || m.ScrollX != 0 {
		sv.vbar.ApplyWheel(m, e.Frame)
		sv.syncScroll()
	}
}

// Update drives scroll wheel and thumb drag. Call after layout each frame.
func (sv *ScrollView) Update(m *input.Mouse) {
	sv.syncScroll()
	if m != nil {
		sv.vbar.Update(m, sv.viewport())
		sv.syncScroll()
	}
}

func (sv *ScrollView) Layout(c *Ctx) *layout.Element {
	if c != nil {
		sv.Update(c.Mouse())
	}
	return sv.host
}

func (n *Node) layoutScroll(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "scroll")
	}
	sv := c.Widget(id, func() any { return NewScrollView(nil) }).(*ScrollView)
	var content *layout.Element
	if n.child != nil {
		content = n.child.Layout(c)
	}
	sv.setContent(content)
	// Re-apply each frame; keep basis/min so Grow(1) fills leftover space only.
	sv.host.Style = applyLayoutSpec(layout.Box().FlexGrow(1).FlexBasis(0).Min(0, 0), n.spec)
	sv.host.AfterLayout = sv.afterLayout
	sv.Update(c.Mouse())
	return sv.host
}
