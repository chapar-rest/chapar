package ui

import (
	"math"
	"time"

	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const (
	drawerHandleHit   = 6
	drawerMinSize     = 80
	drawerAnimDur     = 180 * time.Millisecond
	drawerSwipeSlop   = 8
	drawerSwipeSnap   = 0.4
	drawerEdgeStrip   = 20
)

// Edge selects which viewport edge a Drawer docks to.
type Edge int

const (
	EdgeLeft Edge = iota
	EdgeRight
	EdgeTop
	EdgeBottom
)

// drawerMode is overlay (stack on top of page) or push (page shrinks).
type drawerMode int

const (
	drawerOverlay drawerMode = iota
	drawerPush
)

// drawerGeom centralizes edge-specific layout and resize math.
type drawerGeom struct {
	edge Edge
}

func (g drawerGeom) horizontal() bool {
	return g.edge == EdgeLeft || g.edge == EdgeRight
}

func (g drawerGeom) panelDirection() layout.FlexDirection {
	if g.horizontal() {
		return layout.Row
	}
	return layout.Column
}

// handleFirst reports whether the resize handle precedes the viewport in flex order.
func (g drawerGeom) handleFirst() bool {
	return g.edge == EdgeRight || g.edge == EdgeBottom
}

func (g drawerGeom) handleStyle() layout.Style {
	style := layout.Box().FlexShrink(0)
	if g.horizontal() {
		return style.W(drawerHandleHit)
	}
	return style.H(drawerHandleHit)
}

// applySizeDelta maps pointer movement to a new main-axis size. Dragging the inner
// edge toward the page shrinks the drawer; dragging away expands it.
func (g drawerGeom) applySizeDelta(startSz, delta float32) float32 {
	switch g.edge {
	case EdgeRight, EdgeBottom:
		return startSz - delta
	case EdgeLeft, EdgeTop:
		return startSz + delta
	default:
		return startSz
	}
}

func (g drawerGeom) resizeCursor() input.Cursor {
	if g.horizontal() {
		return input.CursorResizeEW
	}
	return input.CursorResizeNS
}

func (g drawerGeom) gripHorizontal() bool {
	return g.horizontal()
}

func (g drawerGeom) dockLine(f render.Rect) render.Rect {
	switch g.edge {
	case EdgeRight:
		return render.Rect{X: f.X, Y: f.Y, W: 1, H: f.H}
	case EdgeLeft:
		return render.Rect{X: f.X + f.W - 1, Y: f.Y, W: 1, H: f.H}
	case EdgeBottom:
		return render.Rect{X: f.X, Y: f.Y, W: f.W, H: 1}
	case EdgeTop:
		return render.Rect{X: f.X, Y: f.Y + f.H - 1, W: f.W, H: 1}
	default:
		return f
	}
}

func (g drawerGeom) swipeDeltaFromStart(startPtr, ptr float32) float32 {
	switch g.edge {
	case EdgeRight, EdgeBottom:
		return startPtr - ptr
	case EdgeLeft, EdgeTop:
		return ptr - startPtr
	default:
		return 0
	}
}

// drawerView slides a resizable panel from an edge over or beside wrapped page
// content. Open is controlled each frame; size and animation live in the
// widget store keyed by id.
type drawerView struct {
	id           string
	panel        View
	page         View
	open         bool
	edge         Edge
	mode         drawerMode
	size         float32
	minSize      float32
	maxSize      float32
	resizable    bool
	modal        bool
	swipe        bool
	onOpenChange func(bool)
	spec         Spec
}

type drawerState struct {
	size       float32
	progress   float32
	animFrom   float32
	animTo     float32
	animTarget float32
	animStart  time.Time
	animating  bool

	resizing      bool
	resizeStart   float32
	resizeStartSz float32
	resizeHover   bool

	swiping       bool
	swipePending  bool
	swipeStartPtr float32
	swipeStartPr  float32

	root      *layout.Element
	panelEl   *layout.Element
	pageEl    *layout.Element
	hostW     float32
	hostH     float32
	modalHost *drawerModalHost
}

type drawerModalHost struct {
	d *drawerView
}

var _ View = (*drawerView)(nil)
var _ Focusable = (*drawerModalHost)(nil)

// Drawer wraps page content with a slide-from-edge panel. panel is the drawer
// body; page is the main content the drawer overlays or pushes aside.
func Drawer(id string, panel, page View) *drawerView {
	return &drawerView{
		id:        id,
		panel:     panel,
		page:      page,
		edge:      EdgeRight,
		mode:      drawerOverlay,
		resizable: true,
	}
}

// Open sets whether the drawer is shown (controlled).
func (d *drawerView) Open(v bool) *drawerView { d.open = v; return d }

// Edge sets the dock edge (default EdgeRight).
func (d *drawerView) Edge(e Edge) *drawerView { d.edge = e; return d }

// Overlay stacks the panel over page content (default).
func (d *drawerView) Overlay() *drawerView { d.mode = drawerOverlay; return d }

// Push shrinks page content when the drawer is open.
func (d *drawerView) Push() *drawerView { d.mode = drawerPush; return d }

// Size sets the initial main-axis size in pixels (320 horizontal, 240 vertical).
func (d *drawerView) Size(v float32) *drawerView { d.size = v; return d }

// MinSize sets the minimum size when resizing (default 80).
func (d *drawerView) MinSize(v float32) *drawerView { d.minSize = v; return d }

// MaxSize caps resize; 0 means 80% of the drawer host (default).
func (d *drawerView) MaxSize(v float32) *drawerView { d.maxSize = v; return d }

// Resizable enables the inner-edge drag handle (default true).
func (d *drawerView) Resizable(v bool) *drawerView { d.resizable = v; return d }

// Modal adds a dimmed scrim and outside-click / Escape dismiss (overlay only).
func (d *drawerView) Modal(v bool) *drawerView { d.modal = v; return d }

// Swipe enables edge drag-to-open and panel drag-to-close.
func (d *drawerView) Swipe(v bool) *drawerView { d.swipe = v; return d }

// OnOpenChange is called when the user opens or closes via scrim, Escape, or swipe.
func (d *drawerView) OnOpenChange(fn func(bool)) *drawerView {
	d.onOpenChange = fn
	return d
}

// Grow sets flex grow on the drawer host.
func (d *drawerView) Grow(v float32) *drawerView {
	d.spec.grow = v
	d.spec.hasGrow = true
	return d
}

func (d *drawerView) Layout(c *Ctx) *layout.Element {
	id := d.id
	if id == "" {
		id = autoID(c, "drawer")
	}
	initSize := d.size
	if initSize <= 0 {
		initSize = defaultDrawerSize(d.edge)
	}
	st := c.Widget(id, func() any {
		return &drawerState{
			size:       initSize,
			animTarget: -1,
		}
	}).(*drawerState)
	if st.size <= 0 {
		st.size = initSize
	}
	if st.modalHost == nil {
		st.modalHost = &drawerModalHost{d: d}
	}
	st.modalHost.d = d

	d.updateAnimation(c, st)

	th := c.Theme()
	var root *layout.Element
	if d.mode == drawerPush {
		root = d.layoutPush(c, st, th)
	} else {
		root = d.layoutOverlay(c, st, th)
	}
	st.root = root
	box := applyLayoutSpec(layout.Box().FlexGrow(1), d.spec)
	if box.Grow != 0 {
		root.Style.Grow = box.Grow
	}
	if box.Shrink != 0 {
		root.Style.Shrink = box.Shrink
	}
	root.ReapplyStyle()

	if d.modal && d.mode == drawerOverlay && st.progress > 0 && c.Focus() != nil {
		c.Focus().Add(st.modalHost)
	}

	if st.animating || st.swiping {
		c.Animate(16 * time.Millisecond)
	}
	return root
}

func defaultDrawerSize(edge Edge) float32 {
	if edge == EdgeTop || edge == EdgeBottom {
		return 240
	}
	return 320
}

func (d *drawerView) geom() drawerGeom { return drawerGeom{edge: d.edge} }

func (d *drawerView) updateAnimation(c *Ctx, st *drawerState) {
	target := float32(0)
	if d.open {
		target = 1
	}
	if st.swiping {
		return
	}
	if st.animTarget != target {
		st.animating = true
		st.animFrom = st.progress
		st.animTo = target
		st.animStart = c.Now()
		st.animTarget = target
	}
	if st.animating {
		elapsed := c.Now().Sub(st.animStart)
		t := float32(elapsed) / float32(drawerAnimDur)
		if t >= 1 {
			st.progress = st.animTo
			st.animating = false
		} else {
			st.progress = st.animFrom + (st.animTo-st.animFrom)*easeOutCubic(t)
			remain := drawerAnimDur - elapsed
			if remain > 0 {
				c.Animate(remain)
			}
		}
	}
}

func easeOutCubic(t float32) float32 {
	u := 1 - t
	return 1 - u*u*u
}

func (d *drawerView) requestOpen(c *Ctx, open bool) {
	if d.onOpenChange != nil {
		d.onOpenChange(open)
	}
}

func (d *drawerView) minSz() float32 {
	if d.minSize > 0 {
		return d.minSize
	}
	return drawerMinSize
}

func (d *drawerView) maxSz(st *drawerState) float32 {
	if d.maxSize > 0 {
		return d.maxSize
	}
	g := d.geom()
	if g.horizontal() {
		return f32max(d.minSz(), st.hostW*0.8)
	}
	return f32max(d.minSz(), st.hostH*0.8)
}

func (d *drawerView) visibleSize(st *drawerState) float32 {
	return st.size * st.progress
}

func (d *drawerView) layoutPush(c *Ctx, st *drawerState, th *theme.Theme) *layout.Element {
	pageEl := layout.New(layout.Box().FlexGrow(1))
	if d.page != nil {
		pageEl = d.page.Layout(c)
		pageEl.Style = layout.Box().FlexGrow(1)
		pageEl.ReapplyStyle()
	}
	st.pageEl = pageEl

	vis := d.visibleSize(st)
	var children []*layout.Element
	appendPanel := func() {
		if vis <= 0 && !st.swiping {
			return
		}
		panelEl := d.buildPanelShell(c, st, th, vis, false)
		st.panelEl = panelEl
		switch d.edge {
		case EdgeLeft:
			children = append(children, panelEl, pageEl)
		case EdgeRight:
			children = append(children, pageEl, panelEl)
		case EdgeTop:
			children = append(children, panelEl, pageEl)
		case EdgeBottom:
			children = append(children, pageEl, panelEl)
		}
	}
	appendPanel()
	if len(children) == 0 {
		children = []*layout.Element{pageEl}
	}

	g := d.geom()
	dir := g.panelDirection()
	root := layout.New(layout.Box().Direction(dir).AlignItems(layout.AlignStretch).FlexGrow(1), children...)
	root.OnMouse = d.chainMouse(d.mouseRoot(st), d.mousePushEdgeSwipe(c, st))
	return root
}

func (d *drawerView) layoutOverlay(c *Ctx, st *drawerState, th *theme.Theme) *layout.Element {
	pageEl := layout.New(layout.Box().FlexGrow(1))
	if d.page != nil {
		pageEl = d.page.Layout(c)
	}
	pageEl.Style = layout.Box().FlexGrow(1)
	pageEl.ReapplyStyle()
	st.pageEl = pageEl

	children := []*layout.Element{pageEl}
	if d.modal && st.progress > 0 {
		scrim := layout.New(layout.Box().AbsLeft(0).AbsTop(0).AbsRight(0).AbsBottom(0))
		scrim.Paint = func(dl *render.DrawList, _ *shape.Engine) {
			col := render.RGBA8(0, 0, 0, 255)
			col.A = 0.45 * st.progress
			dl.AddRect(scrim.Frame, col)
		}
		scrim.OnMouse = func(_ *layout.Element, m *input.Mouse) {
			if m.Pressed {
				d.requestOpen(c, false)
				m.Consumed = true
			}
		}
		children = append(children, scrim)
	}

	vis := d.visibleSize(st)
	if vis > 0 || st.swiping {
		panelEl := d.buildPanelShell(c, st, th, st.size, true)
		st.panelEl = panelEl
		off := (1 - st.progress) * st.size
		d.applyOverlayPanelPos(panelEl, off, st.size)
		children = append(children, panelEl)
	}

	if d.swipe && !d.open && st.progress <= 0 && !st.swiping {
		strip := d.buildEdgeStrip(c, st)
		children = append(children, strip)
	}

	root := layout.New(
		layout.Box().Display(layout.DisplayStack).AlignItems(layout.AlignStretch).FlexGrow(1),
		children...,
	)
	root.OnMouse = d.mouseRoot(st)
	return root
}

func (d *drawerView) applyOverlayPanelPos(panel *layout.Element, offset, size float32) {
	s := panel.Style
	unset := layout.Px(float32(math.NaN()))
	s.Pos = layout.PositionAbsolute
	s.Left, s.Top, s.Right, s.Bottom = unset, unset, unset, unset
	switch d.edge {
	case EdgeRight:
		s.Top, s.Bottom, s.Right = 0, 0, -offset
		s.Width = size
		s.Height = unset
	case EdgeLeft:
		s.Top, s.Bottom, s.Left = 0, 0, -offset
		s.Width = size
		s.Height = unset
	case EdgeBottom:
		s.Left, s.Right, s.Bottom = 0, 0, -offset
		s.Height = size
		s.Width = unset
	case EdgeTop:
		s.Left, s.Right, s.Top = 0, 0, -offset
		s.Height = size
		s.Width = unset
	}
	panel.Style = s
	panel.ReapplyStyle()
}

// buildPanelShell lays out [handle|viewport] as flex siblings inside the chrome.
func (d *drawerView) buildPanelShell(c *Ctx, st *drawerState, th *theme.Theme, layoutSize float32, overlay bool) *layout.Element {
	g := d.geom()

	var viewport *layout.Element
	if d.panel != nil {
		inner := d.panel.Layout(c)
		viewport = layout.New(
			layout.Box().FlexGrow(1).FlexBasis(0).Min(0, 0),
			inner,
		)
		viewport.Clip = true
	}

	var handle *layout.Element
	if d.resizable && layoutSize > 0 {
		handle = d.buildResizeHandle(st, th, g)
	}

	var children []*layout.Element
	if g.handleFirst() {
		if handle != nil {
			children = append(children, handle)
		}
		if viewport != nil {
			children = append(children, viewport)
		}
	} else {
		if viewport != nil {
			children = append(children, viewport)
		}
		if handle != nil {
			children = append(children, handle)
		}
	}

	box := layout.Box().Direction(g.panelDirection()).AlignItems(layout.AlignStretch).FlexShrink(0)
	if g.horizontal() {
		box = box.W(layoutSize)
	} else {
		box = box.H(layoutSize)
	}
	panel := layout.New(box, children...)
	panel.Clip = true
	panel.Style.BgColor = TokenChrome.Resolve(th)
	panel.Style.Radius = th.Radius.Large
	panel.ReapplyStyle()
	panel.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		f := panel.Frame
		if overlay {
			drawElevationShadow(dl, f, th.Radius.Large, th.Elevation.ShadowLg)
		}
		dl.AddRoundedRect(f, th.Radius.Large, panel.Style.BgColor)
	}
	if d.swipe && (d.open || st.swiping) {
		panel.OnMouse = d.mousePanelSwipe(c, st)
	}
	return panel
}

func (d *drawerView) buildResizeHandle(st *drawerState, th *theme.Theme, g drawerGeom) *layout.Element {
	handle := layout.New(g.handleStyle())
	handle.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		d.paintResizeGrip(dl, handle.Frame, st, th, g)
	}
	handle.OnMouse = d.mouseHandlePress(st)
	return handle
}

func (d *drawerView) paintResizeGrip(dl *render.DrawList, f render.Rect, st *drawerState, th *theme.Theme, g drawerGeom) {
	active := st.resizeHover || st.resizing
	col := th.Border
	if active {
		col = th.Accent
	}
	dl.AddRect(g.dockLine(f), col)

	gripCol := th.ForegroundMuted
	if active {
		gripCol = th.Accent
	}
	const (
		tickTh  float32 = 2
		tickGap float32 = 3
	)
	tickLen := float32(4)
	if g.horizontal() {
		if tickLen > f.W-2 {
			tickLen = f.W - 2
		}
	} else if tickLen > f.H-2 {
		tickLen = f.H - 2
	}
	cx := f.X + f.W/2
	cy := f.Y + f.H/2
	if g.gripHorizontal() {
		total := 3*tickLen + 2*tickGap
		y0 := cy - total/2
		for i := range 3 {
			y := y0 + float32(i)*(tickLen+tickGap)
			dl.AddRect(render.Rect{X: cx - tickLen/2, Y: y, W: tickLen, H: tickTh}, gripCol)
		}
	} else {
		total := 3*tickLen + 2*tickGap
		x0 := cx - total/2
		for i := range 3 {
			x := x0 + float32(i)*(tickLen+tickGap)
			dl.AddRect(render.Rect{X: x, Y: cy - tickLen/2, W: tickTh, H: tickLen}, gripCol)
		}
	}
}

func (d *drawerView) buildEdgeStrip(c *Ctx, st *drawerState) *layout.Element {
	strip := layout.New(layout.Box())
	d.applyEdgeStripPos(strip)
	strip.OnMouse = d.mouseEdgeStrip(c, st)
	return strip
}

func (d *drawerView) applyEdgeStripPos(strip *layout.Element) {
	w := float32(drawerEdgeStrip)
	switch d.edge {
	case EdgeRight:
		strip.Style = layout.Box().AbsTop(0).AbsBottom(0).AbsRight(0).W(w)
	case EdgeLeft:
		strip.Style = layout.Box().AbsTop(0).AbsBottom(0).AbsLeft(0).W(w)
	case EdgeBottom:
		strip.Style = layout.Box().AbsLeft(0).AbsRight(0).AbsBottom(0).H(w)
	case EdgeTop:
		strip.Style = layout.Box().AbsLeft(0).AbsRight(0).AbsTop(0).H(w)
	}
	strip.ReapplyStyle()
}

func (d *drawerView) mouseRoot(st *drawerState) layout.MouseFunc {
	return func(_ *layout.Element, m *input.Mouse) {
		if st.root != nil {
			st.hostW = st.root.Frame.W
			st.hostH = st.root.Frame.H
		}
		if st.resizing {
			g := d.geom()
			m.SetCursor(g.resizeCursor())
			if m.Down {
				delta := d.pointerAlong(m) - st.resizeStart
				newSz := g.applySizeDelta(st.resizeStartSz, delta)
				st.size = clampf(newSz, d.minSz(), d.maxSz(st))
				m.Consumed = true
			} else {
				st.resizing = false
				st.resizeHover = false
			}
		}
	}
}

func (d *drawerView) mouseHandlePress(st *drawerState) layout.MouseFunc {
	return func(e *layout.Element, m *input.Mouse) {
		if !d.resizable || st.resizing {
			return
		}
		inside := e.Frame.Contains(m.X, m.Y)
		st.resizeHover = inside
		g := d.geom()
		if inside {
			m.SetCursor(g.resizeCursor())
		}
		if m.Pressed && inside {
			st.resizing = true
			st.resizeStart = d.pointerAlong(m)
			st.resizeStartSz = st.size
			m.Consumed = true
		}
	}
}

func (d *drawerView) chainMouse(a, b layout.MouseFunc) layout.MouseFunc {
	return func(e *layout.Element, m *input.Mouse) {
		if a != nil {
			a(e, m)
		}
		if !m.Consumed && b != nil {
			b(e, m)
		}
	}
}

func (d *drawerView) mouseEdgeStrip(c *Ctx, st *drawerState) layout.MouseFunc {
	return func(_ *layout.Element, m *input.Mouse) {
		if !d.swipe || d.open || st.resizing {
			return
		}
		d.handleSwipe(c, st, m, true)
	}
}

func (d *drawerView) mousePushEdgeSwipe(c *Ctx, st *drawerState) layout.MouseFunc {
	return func(e *layout.Element, m *input.Mouse) {
		if st.resizing {
			return
		}
		if !d.swipe || d.open || st.swiping {
			if m.Released && st.swipePending {
				d.finishSwipe(c, st, m)
			}
			return
		}
		if !d.inEdgeStrip(e, m) {
			if m.Released && st.swipePending {
				d.finishSwipe(c, st, m)
			}
			return
		}
		d.handleSwipe(c, st, m, true)
	}
}

func (d *drawerView) inEdgeStrip(e *layout.Element, m *input.Mouse) bool {
	f := e.Frame
	if f.W <= 0 || f.H <= 0 {
		return false
	}
	w := float32(drawerEdgeStrip)
	switch d.edge {
	case EdgeRight:
		return m.X >= f.X+f.W-w
	case EdgeLeft:
		return m.X <= f.X+w
	case EdgeBottom:
		return m.Y >= f.Y+f.H-w
	case EdgeTop:
		return m.Y <= f.Y+w
	}
	return false
}

func (d *drawerView) mousePanelSwipe(c *Ctx, st *drawerState) layout.MouseFunc {
	return func(e *layout.Element, m *input.Mouse) {
		if !d.swipe || st.resizing {
			return
		}
		if !e.Frame.Contains(m.X, m.Y) {
			if m.Released && st.swipePending {
				d.finishSwipe(c, st, m)
			}
			return
		}
		if d.resizable && d.isOnHandle(st, m) {
			return
		}
		if !d.open && !st.swiping {
			return
		}
		d.handleSwipe(c, st, m, true)
	}
}

func (d *drawerView) isOnHandle(st *drawerState, m *input.Mouse) bool {
	if st.panelEl == nil {
		return false
	}
	for _, ch := range st.panelEl.Children {
		f := ch.Frame
		if ch.OnMouse == nil {
			continue
		}
		isHandle := (f.W > 0 && f.W <= drawerHandleHit+1) || (f.H > 0 && f.H <= drawerHandleHit+1)
		if isHandle && f.Contains(m.X, m.Y) {
			return true
		}
	}
	return false
}

func (d *drawerView) handleSwipe(c *Ctx, st *drawerState, m *input.Mouse, active bool) {
	if !d.swipe || st.resizing {
		return
	}
	if m.Released {
		d.finishSwipe(c, st, m)
		return
	}
	if !active {
		return
	}
	if m.Pressed {
		st.swipePending = true
		st.swipeStartPtr = d.pointerAlong(m)
		st.swipeStartPr = st.progress
		return
	}
	if st.swipePending && m.Down {
		g := d.geom()
		delta := g.swipeDeltaFromStart(st.swipeStartPtr, d.pointerAlong(m))
		if !st.swiping {
			if absf(delta) < drawerSwipeSlop {
				return
			}
			st.swiping = true
			st.animating = false
		}
		st.progress = clampf(st.swipeStartPr+delta/st.size, 0, 1)
		m.Consumed = true
	}
}

func (d *drawerView) finishSwipe(c *Ctx, st *drawerState, m *input.Mouse) {
	if !st.swipePending {
		return
	}
	st.swipePending = false
	if st.swiping {
		open := st.progress >= drawerSwipeSnap
		d.requestOpen(c, open)
		st.swiping = false
		st.animTarget = -1
		m.Consumed = true
	}
}

func (d *drawerView) pointerAlong(m *input.Mouse) float32 {
	if d.geom().horizontal() {
		return m.X
	}
	return m.Y
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (h *drawerModalHost) Focus()                   {}
func (h *drawerModalHost) Blur()                    {}
func (h *drawerModalHost) Focused() bool            { return h.d.open && h.d.modal }
func (h *drawerModalHost) HandleText([]rune)        {}
func (h *drawerModalHost) CapturesTab() bool        { return false }
func (h *drawerModalHost) FocusOnClick() bool       { return false }
func (h *drawerModalHost) FocusEl() *layout.Element { return nil }

func (h *drawerModalHost) HandleKeys(keys []input.KeyEvent) {
	if !h.d.open || !h.d.modal || h.d.mode != drawerOverlay {
		return
	}
	for _, ev := range keys {
		if ev.Key == input.KeyEscape {
			if h.d.onOpenChange != nil {
				h.d.onOpenChange(false)
			}
			return
		}
	}
}
