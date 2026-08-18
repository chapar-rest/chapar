package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// TabModel is the display state of a single tab.
type TabModel struct {
	Title    string
	Modified bool
	Badge    string
}

type tabsData struct {
	tabs []TabModel
	bg   render.Color
}

type tabsState struct {
	hoverTab, hoverClose int
	focused              bool
	el                   *layout.Element
	onActivate           func(int)
	n, active            int
}

func (t *tabsState) Focus()                   { t.focused = true }
func (t *tabsState) Blur()                    { t.focused = false }
func (t *tabsState) Focused() bool            { return t.focused }
func (t *tabsState) HandleText(_ []rune)      {}
func (t *tabsState) CapturesTab() bool        { return false }
func (t *tabsState) FocusOnClick() bool       { return false }
func (t *tabsState) FocusEl() *layout.Element { return t.el }

func (t *tabsState) HandleKeys(keys []input.KeyEvent) {
	if !t.focused || t.n == 0 {
		return
	}
	for _, ev := range keys {
		if ev.Mods != 0 {
			continue
		}
		switch ev.Key {
		case input.KeyLeft:
			if t.active > 0 && t.onActivate != nil {
				t.onActivate(t.active - 1)
			}
		case input.KeyRight:
			if t.active < t.n-1 && t.onActivate != nil {
				t.onActivate(t.active + 1)
			}
		case input.KeyEnter:
			if t.onActivate != nil {
				t.onActivate(t.active)
			}
		}
	}
}

const tabMaxText = 22

// Tabs is a horizontal strip of document tabs. Active index is controlled via .Selected(i).
func Tabs(id string, tabs []TabModel) *Node {
	return &Node{kind: kindTabs, id: id, extra: &tabsData{tabs: tabs}}
}

// TabBackground overrides the strip fill (e.g. workspace background).
func (n *Node) TabBackground(c render.Color) *Node {
	if d, ok := n.extra.(*tabsData); ok {
		d.bg = c
	}
	return n
}

func tabBadgeWidth(badge string) float32 {
	if badge == "" {
		return 0
	}
	eng := frameText()
	if eng == nil {
		return 0
	}
	bw, _ := eng.MeasureAt(badge, theme.Current().Typography.Caption.Size)
	return bw + 10
}

func (n *Node) layoutTabs(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "tabs")
	}
	st := c.Widget(id, func() any { return &tabsState{hoverTab: -1, hoverClose: -1} }).(*tabsState)
	d, _ := n.extra.(*tabsData)
	if d == nil {
		d = &tabsData{}
	}
	th := c.Theme()
	el := layout.New(applyLayoutSpec(layout.Box().H(th.Metrics.ControlHeight), n.spec))
	st.el = el
	st.n = len(d.tabs)
	st.active = n.selected
	st.onActivate = func(i int) {
		if n.onSelectIdx != nil {
			n.onSelectIdx(i, "")
		}
	}
	if c.Focus() != nil {
		c.Focus().Add(st)
	}
	tabs := d.tabs
	active := n.selected
	onActivate := n.onSelectIdx
	onClose := n.onCloseIdx
	bg := d.bg
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		paintTabs(dl, text, el, tabs, active, st.hoverTab, st.hoverClose, st.focused, bg)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		st.hoverTab, st.hoverClose = -1, -1
		if !e.Frame.Contains(m.X, m.Y) {
			return
		}
		ext := tabExtents(e, tabs)
		for i, te := range ext {
			if m.X < te.x || m.X > te.x+te.w {
				continue
			}
			st.hoverTab = i
			if te.close.Contains(m.X, m.Y) {
				st.hoverClose = i
				if m.Pressed {
					m.Consumed = true
					if onClose != nil {
						onClose(i)
					}
				}
				return
			}
			if m.Pressed {
				m.Consumed = true
				if onActivate != nil {
					onActivate(i, "")
				}
			}
			return
		}
	}
	return el
}

type tabExtent struct {
	x, w  float32
	close render.Rect
}

func tabExtents(el *layout.Element, tabs []TabModel) []tabExtent {
	th := theme.Current()
	f := el.Frame
	out := make([]tabExtent, len(tabs))
	x := f.X + el.Style.Padding.Left
	padX := th.Spacing.M
	closeW := th.Metrics.IconSizeMD
	style := th.Typography.Body
	eng := frameText()
	for i, tab := range tabs {
		title := truncate(tab.Title, tabMaxText)
		var tw float32
		if eng != nil {
			tw, _ = eng.MeasureAt(title, style.Size)
		}
		badgeW := tabBadgeWidth(tab.Badge)
		if badgeW > 0 {
			badgeW += th.Spacing.SNudge
		}
		w := tw + badgeW + 2*padX + closeW
		closeX := x + w - closeW
		cy := f.Y + (f.H-closeW)/2
		out[i] = tabExtent{
			x: x, w: w,
			close: render.Rect{X: closeX, Y: cy, W: closeW - th.Spacing.XS, H: closeW - th.Spacing.XS},
		}
		x += w
	}
	return out
}

func paintTabs(dl *render.DrawList, text *shape.Engine, el *layout.Element, tabs []TabModel, active, hoverTab, hoverClose int, focused bool, bgOverride render.Color) {
	th := theme.Current()
	f := el.Frame
	padX := th.Spacing.M
	style := th.Typography.Body
	bg := th.Chrome
	if bgOverride.A > 0 {
		bg = bgOverride
	}
	dl.AddRect(f, bg)
	dl.PushClip(f)
	ext := tabExtents(el, tabs)
	for i, tab := range tabs {
		e := ext[i]
		rect := render.Rect{X: e.x, Y: f.Y, W: e.w, H: f.H}
		switch {
		case i == active:
			dl.AddRect(rect, th.ListActive)
		case i == hoverTab:
			dl.AddRect(rect, th.ListHover)
		}
		if i == active {
			dl.AddRect(render.Rect{X: e.x, Y: f.Y + f.H - th.Stroke.Thick, W: e.w, H: th.Stroke.Thick}, th.Accent)
		}
		if focused && i == active {
			drawFocusRing(dl, rect, th.ListActive, th)
		}
		title := truncate(tab.Title, tabMaxText)
		tw, lh := text.MeasureAt(title, style.Size)
		ty := f.Y + (f.H-lh)/2
		text.DrawStringTopAt(dl, title, e.x+padX, ty, th.Foreground, style.Size)
		if tab.Badge != "" {
			bsz := th.Typography.Caption.Size
			bw, bh := text.MeasureAt(tab.Badge, bsz)
			pillW := bw + 10
			pillH := bh + 2
			px := e.x + padX + tw + th.Spacing.SNudge
			py := f.Y + (f.H-pillH)/2
			dl.AddRoundedRect(render.Rect{X: px, Y: py, W: pillW, H: pillH}, th.Radius.Circular, th.ChromeMuted)
			text.DrawStringTopAt(dl, tab.Badge, px+5, py+(pillH-bh)/2, th.ForegroundMuted, bsz)
		}
		c := e.close
		if i == hoverClose {
			dl.AddRect(c, th.ListHover)
			if sheet := frameIcons(); sheet != nil {
				sheet.Draw(dl, "close", c, th.Foreground)
			}
		} else if tab.Modified {
			if sheet := frameIcons(); sheet != nil {
				sheet.Draw(dl, "circle", shrinkRect(c, 0.5), th.ForegroundMuted)
			}
		} else if i == hoverTab || i == active {
			if sheet := frameIcons(); sheet != nil {
				sheet.Draw(dl, "close", c, th.ForegroundMuted)
			}
		}
	}
	dl.PopClip()
}

func shrinkRect(r render.Rect, factor float32) render.Rect {
	w, h := r.W*factor, r.H*factor
	return render.Rect{X: r.X + (r.W-w)/2, Y: r.Y + (r.H-h)/2, W: w, H: h}
}

func truncate(s string, max int) string {
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	if max <= 2 {
		return string(rs[:max])
	}
	return string(rs[:max-2]) + ".."
}
