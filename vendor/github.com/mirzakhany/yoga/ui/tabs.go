package ui

import (
	"strconv"

	"github.com/mirzakhany/yoga/icons"
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
	tabs      []TabModel
	bg        render.Color
	closable  bool
	tabMenuFn func(int) []MenuItem
}

type tabsState struct {
	hoverTab, hoverClose int
	hoverOverflow        bool
	focused              bool
	el                   *layout.Element
	onActivate           func(int)
	n, active            int

	// scrollX is how far the strip is scrolled right, in pixels. Revealed
	// tracks the active index, tab count and strip width the scroll was last
	// fitted to, so the active tab is scrolled into view only when one of
	// them changes and a wheel scroll is otherwise left alone.
	scrollX                   float32
	revealedActive, revealedN int
	revealedW                 float32
	menu                      *Menu
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

// Overflow menu width bounds; the menu sizes to its longest title in between.
const (
	tabMenuMinW = 160
	tabMenuMaxW = 360
)

// Tabs is a horizontal strip of document tabs. Active index is controlled via .Selected(i).
//
// When the tabs are wider than the strip it scrolls: the wheel or trackpad
// moves it sideways, the active tab is kept in view, and a button at the right
// edge shows how many tabs are hidden and opens a list of them.
func Tabs(id string, tabs []TabModel) *Node {
	return &Node{kind: kindTabs, id: id, extra: &tabsData{tabs: tabs, closable: true}}
}

// TabBackground overrides the strip fill (e.g. workspace background).
func (n *Node) TabBackground(c render.Color) *Node {
	if d, ok := n.extra.(*tabsData); ok {
		d.bg = c
	}
	return n
}

// Closable controls whether tabs show a close control (default true).
func (n *Node) Closable(v bool) *Node {
	if d, ok := n.extra.(*tabsData); ok {
		d.closable = v
	}
	return n
}

// OnTabContextMenu sets the right-click menu of a tab: fn receives the tab
// index and returns its items, such as Close / Close Others / Close All.
// Returning none shows no menu.
func (n *Node) OnTabContextMenu(fn func(i int) []MenuItem) *Node {
	if d, ok := n.extra.(*tabsData); ok {
		d.tabMenuFn = fn
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
	st := c.Widget(id, func() any {
		return &tabsState{hoverTab: -1, hoverClose: -1, revealedActive: -1, revealedN: -1}
	}).(*tabsState)
	if st.menu == nil {
		st.menu = NewMenu(tabMenuMinW, nil)
	}
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
	tabMenuFn := d.tabMenuFn
	bg := d.bg
	closable := d.closable
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		g := st.fit(el, tabs, active, closable)
		paintTabs(dl, text, el, g, tabs, active, st, bg, closable)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		g := st.fit(e, tabs, active, closable)
		hoverTab, hoverClose, hoverOverflow := -1, -1, false
		if e.Frame.Contains(m.X, m.Y) && !m.Consumed {
			if g.overflows() && (m.ScrollX != 0 || m.ScrollY != 0) {
				st.scrollX -= (m.ScrollX + m.ScrollY) * WheelPixelsPerUnit
				m.ScrollX, m.ScrollY = 0, 0
				m.Consumed = true
				g = st.fit(e, tabs, active, closable)
				c.MarkNeedsPaint()
			}
			if g.overflows() && g.overflow.Contains(m.X, m.Y) {
				hoverOverflow = true
				if m.Pressed {
					m.Consumed = true
					st.openOverflowMenu(tabs, g, onActivate)
					c.MarkNeedsPaint()
				}
			} else if i := g.tabAt(m.X, m.Y); i >= 0 {
				hoverTab = i
				te := g.ext[i]
				switch {
				case m.RightPressed:
					if tabMenuFn != nil {
						m.Consumed = true
						if items := tabMenuFn(i); len(items) > 0 {
							st.menu.width = tabMenuMinW
							st.menu.SetItems(items)
							st.menu.OpenAt(m.X, m.Y)
						}
						c.MarkNeedsPaint()
					}
				case closable && te.close.Contains(m.X, m.Y):
					hoverClose = i
					if m.Pressed {
						m.Consumed = true
						if onClose != nil {
							onClose(i)
						}
						c.MarkNeedsPaint()
					}
				case m.Pressed:
					m.Consumed = true
					if onActivate != nil {
						onActivate(i, "")
					}
					c.MarkNeedsPaint()
				}
			}
		}
		if st.hoverTab != hoverTab || st.hoverClose != hoverClose || st.hoverOverflow != hoverOverflow {
			st.hoverTab, st.hoverClose, st.hoverOverflow = hoverTab, hoverClose, hoverOverflow
			c.MarkNeedsPaint()
		}
	}
	if st.menu.Open {
		if kb := c.Keyboard(); kb != nil {
			for _, ev := range kb.Keys {
				if ev.Key == input.KeyEscape {
					st.menu.Close()
					c.MarkNeedsPaint()
					break
				}
			}
		}
		st.menu.BindPaint(c)
		c.Overlay(st.menu.overlay())
	}
	return el
}

// openOverflowMenu lists the hidden tabs under the overflow button, in strip
// order, with a separator between those cut off on the left and those on the
// right; choosing one activates it, which scrolls it into view.
func (st *tabsState) openOverflowMenu(tabs []TabModel, g tabGeom, onActivate func(int, string)) {
	th := theme.Current()
	var items []MenuItem
	w := float32(tabMenuMinW)
	eng := frameText()
	leftGroup := false
	for i, tab := range tabs {
		left, right := g.hiddenLeft(i), g.hiddenRight(i)
		if !left && !right {
			continue
		}
		if left {
			leftGroup = true
		} else if leftGroup {
			items = append(items, MenuSeparator)
			leftGroup = false
		}
		i := i
		label := tab.Title
		if tab.Modified {
			label += " \u2022"
		}
		items = append(items, MenuItem{
			Label:    label,
			Shortcut: tab.Badge,
			OnSelect: func() {
				if onActivate != nil {
					onActivate(i, "")
				}
			},
		})
		if eng != nil {
			lw, _ := eng.MeasureAt(label+tab.Badge, th.Typography.Body.Size)
			lw += 2*th.Spacing.MNudge + th.Spacing.L
			if lw > w {
				w = lw
			}
		}
	}
	if len(items) == 0 {
		return
	}
	if w > tabMenuMaxW {
		w = tabMenuMaxW
	}
	st.menu.width = w
	st.menu.SetItems(items)
	o := g.overflow
	st.menu.OpenAt(o.X+o.W-w, o.Y+o.H)
}

// fit clamps the scroll offset to the strip and, when the active tab, the tab
// count or the strip width changed since the last fit, scrolls the active tab
// fully into view. It returns the resulting geometry.
func (st *tabsState) fit(el *layout.Element, tabs []TabModel, active int, closable bool) tabGeom {
	g := tabGeometry(el, tabs, closable, st.scrollX)
	if !g.overflows() {
		st.scrollX = 0
	} else if active != st.revealedActive || len(tabs) != st.revealedN || el.Frame.W != st.revealedW {
		if active >= 0 && active < len(g.ext) {
			left := g.ext[active].x - g.view.X + st.scrollX
			right := left + g.ext[active].w
			if left < st.scrollX {
				st.scrollX = left
			} else if right > st.scrollX+g.view.W {
				st.scrollX = right - g.view.W
			}
		}
	}
	st.revealedActive, st.revealedN, st.revealedW = active, len(tabs), el.Frame.W
	if maxScroll := g.content - g.view.W; st.scrollX > maxScroll {
		st.scrollX = maxScroll
	}
	if st.scrollX < 0 {
		st.scrollX = 0
	}
	if st.scrollX != g.scrollX {
		g = tabGeometry(el, tabs, closable, st.scrollX)
	}
	return g
}

type tabExtent struct {
	x, w  float32
	close render.Rect
}

// tabGeom is the strip layout for one scroll offset: every tab's extent in
// screen coordinates, the viewport the tabs are clipped to, and the overflow
// button, which is empty when every tab fits.
type tabGeom struct {
	ext      []tabExtent
	view     render.Rect
	overflow render.Rect
	content  float32 // total width of all tabs
	scrollX  float32
}

func (g tabGeom) overflows() bool { return g.overflow.W > 0 }

// tabAt returns the index of the tab under (x, y), or -1 outside the
// viewport or past the last tab.
func (g tabGeom) tabAt(x, y float32) int {
	if !g.view.Contains(x, y) {
		return -1
	}
	for i, te := range g.ext {
		if x >= te.x && x <= te.x+te.w {
			return i
		}
	}
	return -1
}

// hiddenLeft reports whether tab i is cut off by the left edge of the viewport.
func (g tabGeom) hiddenLeft(i int) bool { return g.ext[i].x < g.view.X-0.5 }

// hiddenRight reports whether tab i is cut off by the right edge of the viewport.
func (g tabGeom) hiddenRight(i int) bool {
	return g.ext[i].x+g.ext[i].w > g.view.X+g.view.W+0.5
}

// hidden counts the tabs that are not fully inside the viewport.
func (g tabGeom) hidden() int {
	n := 0
	for i := range g.ext {
		if g.hiddenLeft(i) || g.hiddenRight(i) {
			n++
		}
	}
	return n
}

// tabOverflowWidth is the width of the overflow button for a strip of n tabs.
// It is sized for the largest count it can show, so it does not change width
// (and move the viewport) as tabs scroll in and out of view.
func tabOverflowWidth(n int) float32 {
	th := theme.Current()
	w := th.Metrics.IconSizeSM + 2*th.Spacing.S
	if eng := frameText(); eng != nil {
		tw, _ := eng.MeasureAt("+"+strconv.Itoa(n), th.Typography.Caption.Size)
		w += tw + th.Spacing.XS
	}
	return w
}

func tabGeometry(el *layout.Element, tabs []TabModel, closable bool, scrollX float32) tabGeom {
	th := theme.Current()
	f := el.Frame
	padX := th.Spacing.M
	closeW := float32(0)
	if closable {
		closeW = th.Metrics.IconSizeMD
	}
	style := th.Typography.Body
	eng := frameText()
	g := tabGeom{ext: make([]tabExtent, len(tabs)), scrollX: scrollX}
	widths := make([]float32, len(tabs))
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
		widths[i] = tw + badgeW + 2*padX + closeW
		g.content += widths[i]
	}
	g.view = render.Rect{X: f.X + el.Style.Padding.Left, Y: f.Y, W: f.W - el.Style.Padding.Left - el.Style.Padding.Right, H: f.H}
	if g.content > g.view.W+0.5 && len(tabs) > 1 {
		ow := tabOverflowWidth(len(tabs))
		if ow > g.view.W {
			ow = g.view.W
		}
		g.view.W -= ow
		g.overflow = render.Rect{X: g.view.X + g.view.W, Y: f.Y, W: ow, H: f.H}
	} else {
		g.scrollX = 0
	}
	x := g.view.X - g.scrollX
	for i, w := range widths {
		var closeRect render.Rect
		if closable {
			closeX := x + w - closeW
			cy := f.Y + (f.H-closeW)/2
			closeRect = render.Rect{X: closeX, Y: cy, W: closeW - th.Spacing.XS, H: closeW - th.Spacing.XS}
		}
		g.ext[i] = tabExtent{x: x, w: w, close: closeRect}
		x += w
	}
	return g
}

// tabExtents returns the tab extents with the strip scrolled to the start.
func tabExtents(el *layout.Element, tabs []TabModel, closable bool) []tabExtent {
	return tabGeometry(el, tabs, closable, 0).ext
}

func paintTabs(dl *render.DrawList, text *shape.Engine, el *layout.Element, g tabGeom, tabs []TabModel, active int, st *tabsState, bgOverride render.Color, closable bool) {
	th := theme.Current()
	f := el.Frame
	padX := th.Spacing.M
	style := th.Typography.Body
	bg := th.Chrome
	if bgOverride.A > 0 {
		bg = bgOverride
	}
	hoverTab, hoverClose := st.hoverTab, st.hoverClose
	dl.AddRect(f, bg)
	dl.PushClip(g.view)
	for i, tab := range tabs {
		e := g.ext[i]
		if e.x+e.w < g.view.X || e.x > g.view.X+g.view.W {
			continue
		}
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
		if st.focused && i == active {
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
		if closable {
			c := e.close
			if i == hoverClose {
				dl.AddRect(c, th.ListHover)
				if sheet := frameIcons(); sheet != nil {
					sheet.Draw(dl, icons.X, c, th.Foreground)
				}
			} else if tab.Modified {
				if sheet := frameIcons(); sheet != nil {
					sheet.Draw(dl, icons.Circle, shrinkRect(c, 0.5), th.ForegroundMuted)
				}
			} else if i == hoverTab || i == active {
				if sheet := frameIcons(); sheet != nil {
					sheet.Draw(dl, icons.X, c, th.ForegroundMuted)
				}
			}
		}
	}
	dl.PopClip()
	if g.overflows() {
		paintTabOverflow(dl, text, g, st.hoverOverflow || st.menu != nil && st.menu.Open)
	}
}

// paintTabOverflow draws the overflow button: a divider, the hidden-tab count
// and a chevron.
func paintTabOverflow(dl *render.DrawList, text *shape.Engine, g tabGeom, hot bool) {
	th := theme.Current()
	o := g.overflow
	if hot {
		dl.AddRect(o, th.ListHover)
	}
	dl.AddRect(render.Rect{X: o.X, Y: o.Y + th.Spacing.XS, W: th.Stroke.Thin, H: o.H - 2*th.Spacing.XS}, th.Border)
	col := th.ForegroundMuted
	if hot {
		col = th.Foreground
	}
	icon := th.Metrics.IconSizeSM
	x := o.X + th.Spacing.S
	if n := g.hidden(); n > 0 {
		label := "+" + strconv.Itoa(n)
		sz := th.Typography.Caption.Size
		_, lh := text.MeasureAt(label, sz)
		text.DrawStringTopAt(dl, label, x, o.Y+(o.H-lh)/2, col, sz)
	}
	if sheet := frameIcons(); sheet != nil {
		ix := o.X + o.W - th.Spacing.S - icon
		sheet.Draw(dl, icons.ChevronDown, render.Rect{X: ix, Y: o.Y + (o.H-icon)/2, W: icon, H: icon}, col)
	}
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
