package ui

import (
	"strings"
	"time"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// DropPos describes where a dragged node lands relative to the target.
type DropPos int

const (
	DropBefore DropPos = iota // insert before the target in its parent
	DropInside                // insert as a child of the target (branches only)
	DropAfter                 // insert after the target in its parent
)

// DropEvent is passed to Tree.OnDrop when the user completes a drag.
type DropEvent struct {
	Source *TreeNode
	Target *TreeNode
	Pos    DropPos
}

// TreeNode is one item in a Tree. Applications build a node hierarchy (or supply
// a Loader for lazy children) and attach arbitrary data via Data. Icons are
// customizable per node, or globally via Tree.IconFor.
type TreeNode struct {
	Label string

	// Icon is the leaf glyph (default File). OpenIcon/ClosedIcon are the branch
	// glyphs (defaults FolderOpen/Folder). Any may be overridden globally by
	// Tree.IconFor.
	Icon       icons.Icon
	OpenIcon   icons.Icon
	ClosedIcon icons.Icon

	// Leaf forces a node to be non-expandable even if it has (or could load)
	// children.
	Leaf bool

	// Data is an opaque payload the application can read back in callbacks.
	Data any

	// Children holds this node's child nodes. Populate directly for static trees,
	// or leave empty and use Tree.Loader for lazy loading on first expand.
	Children []*TreeNode

	// internal flattening/expansion state
	expanded bool
	loaded   bool
	depth    int
	parent   *TreeNode
}

// Tree is a scrollable, expandable tree view.
type Tree struct {
	host *layout.Element

	root    *TreeNode
	visible []*TreeNode // flattened, rebuilt on every structural change

	vbar, hbar *Scrollbar
	menu       *Menu

	scrollX, scrollY   float32
	contentW, contentH float32

	// ChevronOpen/ChevronClosed are the expand indicator glyphs for branches.
	ChevronOpen   icons.Icon
	ChevronClosed icons.Icon

	// Background overrides the panel fill color. When nil the tree is
	// transparent and the parent view's background shows through. A pointer
	// lets it track live theme switches.
	Background *render.Color

	// Loader returns a node's children the first time it is expanded.
	Loader func(n *TreeNode) []*TreeNode

	// IconFor optionally overrides the icon (and its color) for a node.
	IconFor func(n *TreeNode, expanded bool) (icon icons.Icon, color render.Color)

	// ContextMenu builds the right-click menu items for a node.
	ContextMenu func(n *TreeNode) []MenuItem

	// OnActivate fires when a leaf is clicked. OnToggle fires after a branch expands/collapses.
	OnActivate func(n *TreeNode)
	OnToggle   func(n *TreeNode)
	// OnDrop fires when the user drops a dragged node. Set this to enable
	// drag-and-drop. The handler is responsible for mutating the data model
	// and calling Rebuild as needed.
	OnDrop func(ev DropEvent)

	hover    int
	selected int
	focused  bool
	rowH     float32
	markPaint func()

	// drag-and-drop state
	dragging      bool
	dragNode      *TreeNode // the node being dragged; pointer survives t.visible rebuilds
	dragStartX    float32
	dragStartY    float32
	dragX, dragY  float32 // current cursor position during drag (for ghost rendering)
	dragTargetIdx int     // prospective drop-target index in t.visible (-1 = none)
	dragPos       DropPos // placement within dragTargetIdx

	// auto-expand: open a collapsed folder when the drag cursor lingers over it
	dragHoverNode  *TreeNode
	dragHoverStart time.Time

	// filter restricts visible rows to subtrees whose labels match.
	filter string
}

const treeMenuW = 180
const dragThreshold = float32(4)
const dragExpandDelay = 600 * time.Millisecond

// NewTree builds a tree rooted at root (root itself is not drawn; its children
// are the top-level rows).
func NewTree(root *TreeNode) *Tree {
	th := theme.Current()
	t := &Tree{
		root:          root,
		hover:         -1,
		selected:      -1,
		rowH:          th.Typography.Body.LineHeight + th.Spacing.S,
		ChevronOpen:   icons.ChevronDown,
		ChevronClosed: icons.ChevronRight,
	}
	if t.root == nil {
		t.root = &TreeNode{}
	}
	t.root.expanded = true
	t.dragTargetIdx = -1

	barSize := th.Metrics.ScrollbarSize
	t.vbar = NewScrollbarAxis(Vertical, &t.scrollY, &t.contentH, barSize)
	t.hbar = NewScrollbarAxis(Horizontal, &t.scrollX, &t.contentW, barSize)
	t.menu = NewMenu(treeMenuW, nil)

	t.host = layout.New(layout.Box().FlexGrow(1), t.vbar.host, t.hbar.host)
	t.host.Clip = true
	t.host.Paint = t.paint
	t.host.OnMouse = t.onMouse

	t.ensureLoaded(t.root)
	t.rebuild()
	return t
}

// SetRoot replaces the root node and rebuilds the visible list.
func (t *Tree) SetRoot(root *TreeNode) {
	t.root = root
	if t.root == nil {
		t.root = &TreeNode{}
	}
	t.root.expanded = true
	t.root.loaded = false
	t.ensureLoaded(t.root)
	t.rebuild()
}

// Layout is the new ui.View entry point: it registers the tree with the frame's
// focus scope and, while open, self-registers its context menu as an overlay.
func (t *Tree) Layout(c *Ctx) *layout.Element {
	c.Focus().Add(t)
	t.markPaint = c.MarkNeedsPaint
	t.Update(c.Mouse())
	if t.menu.Open {
		t.menu.BindPaint(c)
		c.Overlay(t.menu.overlay())
	}
	return t.host
}

// ContentHeight is the total pixel height of all visible rows.
func (t *Tree) ContentHeight() float32 { return t.contentH }

// Root returns the (undrawn) root node.
func (t *Tree) Root() *TreeNode { return t.root }

// Parent returns the node's parent, or nil when detached.
func (n *TreeNode) Parent() *TreeNode { return n.parent }

// AddChild appends child to n, links parent pointers, and marks n as a branch.
func (n *TreeNode) AddChild(child *TreeNode) {
	if child == nil {
		return
	}
	n.Children = append(n.Children, child)
	child.parent = n
	child.depth = n.depth + 1
	n.loaded = true
	n.Leaf = false
}

// InsertChild inserts child at index i in n's children.
func (n *TreeNode) InsertChild(i int, child *TreeNode) {
	if child == nil {
		return
	}
	if i < 0 {
		i = 0
	}
	if i > len(n.Children) {
		i = len(n.Children)
	}
	n.Children = append(n.Children[:i], append([]*TreeNode{child}, n.Children[i:]...)...)
	child.parent = n
	child.depth = n.depth + 1
	n.loaded = true
	n.Leaf = false
}

// RemoveChild detaches child from n. Returns whether child was found.
func (n *TreeNode) RemoveChild(child *TreeNode) bool {
	for i, c := range n.Children {
		if c == child {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			child.parent = nil
			return true
		}
	}
	return false
}

// AddChild appends child under parent (nil parent means the tree root) and
// rebuilds the visible rows. The parent is expanded so the new row is visible.
func (t *Tree) AddChild(parent, child *TreeNode) {
	if child == nil {
		return
	}
	if parent == nil {
		parent = t.root
	}
	parent.AddChild(child)
	if !parent.Leaf {
		parent.expanded = true
	}
	t.rebuild()
}

// Remove detaches n from its parent and rebuilds the visible rows.
func (t *Tree) Remove(n *TreeNode) bool {
	if n == nil || n.parent == nil {
		return false
	}
	if !n.parent.RemoveChild(n) {
		return false
	}
	if t.selected >= 0 && t.selected < len(t.visible) && t.visible[t.selected] == n {
		t.selected = -1
	}
	t.rebuild()
	return true
}

// Rebuild refreshes the flattened visible row list after structural changes.
func (t *Tree) Rebuild() { t.rebuild() }

// ensureLoaded populates a branch's children once, via Loader if provided.
func (t *Tree) ensureLoaded(n *TreeNode) {
	if n.loaded || n.Leaf {
		return
	}
	n.loaded = true
	if len(n.Children) == 0 && t.Loader != nil {
		n.Children = t.Loader(n)
	}
	for _, c := range n.Children {
		c.parent = n
		c.depth = n.depth + 1
	}
}

// SetFilter restricts visible rows to nodes whose labels match query.
func (t *Tree) SetFilter(query string) {
	t.filter = query
	t.rebuild()
}

func (t *Tree) labelMatches(n *TreeNode) bool {
	if t.filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(n.Label), strings.ToLower(t.filter))
}

func (t *Tree) subtreeMatches(n *TreeNode) bool {
	if t.labelMatches(n) {
		return true
	}
	if n.Leaf {
		return false
	}
	t.ensureLoaded(n)
	for _, c := range n.Children {
		if t.subtreeMatches(c) {
			return true
		}
	}
	return false
}

// rebuild flattens expanded branches into the visible slice (pre-order).
func (t *Tree) rebuild() {
	t.visible = t.visible[:0]
	if t.filter == "" {
		var walk func(n *TreeNode)
		walk = func(n *TreeNode) {
			for _, c := range n.Children {
				c.parent = n
				c.depth = n.depth + 1
				t.visible = append(t.visible, c)
				if !c.Leaf && c.expanded {
					walk(c)
				}
			}
		}
		walk(t.root)
	} else {
		var walk func(n *TreeNode)
		walk = func(n *TreeNode) {
			t.ensureLoaded(n)
			for _, c := range n.Children {
				if !t.subtreeMatches(c) {
					continue
				}
				c.parent = n
				c.depth = n.depth + 1
				if !c.Leaf {
					c.expanded = true
				}
				t.visible = append(t.visible, c)
				if !c.Leaf {
					walk(c)
				}
			}
		}
		walk(t.root)
	}
	t.computeContentSize()
}

func (t *Tree) barSize() float32  { return theme.Current().Metrics.ScrollbarSize }
func (t *Tree) indent() float32   { return theme.Current().Metrics.TreeIndent }
func (t *Tree) padX() float32     { return theme.Current().Spacing.S }
func (t *Tree) iconW() float32    { return theme.Current().Metrics.TreeIconSize }
func (t *Tree) chevW() float32    { return theme.Current().Metrics.TreeChevronSize }
func (t *Tree) labelGap() float32 { return theme.Current().Spacing.SNudge }

// scrollMetrics computes the content viewport and whether each bar should be shown.
func (t *Tree) scrollMetrics() (clientW, clientH float32, vShow, hShow bool) {
	f := t.host.Frame
	clientW, clientH = f.W, f.H
	vShow = t.contentH > clientH
	bar := t.barSize()
	if vShow {
		clientW = f.W - bar
	}
	hShow = t.contentW > clientW
	if hShow {
		clientH = f.H - bar
	}
	if t.contentH > clientH {
		vShow = true
		clientW = f.W - bar
		hShow = t.contentW > clientW
		if hShow {
			clientH = f.H - bar
		}
	}
	return clientW, clientH, vShow, hShow
}

func (t *Tree) contentViewport() render.Rect {
	f := t.host.Frame
	cw, ch, _, _ := t.scrollMetrics()
	return render.Rect{X: f.X, Y: f.Y, W: cw, H: ch}
}

func (t *Tree) syncScrollbarLayout(vShow, hShow bool) {
	if vShow {
		bottom := float32(0)
		if hShow {
			bottom = t.barSize()
		}
		t.vbar.host.Style = layout.Box().W(t.barSize()).AbsTop(0).AbsRight(0).AbsBottom(bottom)
		t.vbar.host.ReapplyStyle()
	}
	if hShow {
		right := float32(0)
		if vShow {
			right = t.barSize()
		}
		t.hbar.host.Style = layout.Box().H(t.barSize()).AbsLeft(0).AbsRight(right).AbsBottom(0)
		t.hbar.host.ReapplyStyle()
	}
}

func (t *Tree) clampScroll() {
	clientW, clientH, _, _ := t.scrollMetrics()
	t.scrollY = clampf(t.scrollY, 0, f32max(0, t.contentH-clientH))
	t.scrollX = clampf(t.scrollX, 0, f32max(0, t.contentW-clientW))
}

// computeContentSize measures the widest row and the total height.
func (t *Tree) computeContentSize() {
	var maxW float32
	for _, n := range t.visible {
		lw, _ := frameText().Measure(n.Label)
		rowW := t.padX() + float32(n.depth)*t.indent() + t.chevW() + t.iconW() + t.labelGap() + lw + t.padX()
		if rowW > maxW {
			maxW = rowW
		}
	}
	t.contentW = maxW
	t.contentH = float32(len(t.visible)) * t.rowH
}

// toggle expands/collapses a branch.
func (t *Tree) toggle(n *TreeNode) {
	if n.Leaf {
		return
	}
	n.expanded = !n.expanded
	if n.expanded {
		t.ensureLoaded(n)
	}
	t.rebuild()
	if t.OnToggle != nil {
		t.OnToggle(n)
	}
}

// branch reports whether a node should be treated as expandable.
func (n *TreeNode) branch() bool { return !n.Leaf }

func (t *Tree) iconFor(n *TreeNode) (icons.Icon, render.Color) {
	th := theme.Current()
	if t.IconFor != nil {
		return t.IconFor(n, n.expanded)
	}
	if n.branch() {
		name := n.ClosedIcon
		if name.Empty() {
			name = icons.Folder
		}
		if n.expanded {
			if !n.OpenIcon.Empty() {
				name = n.OpenIcon
			} else {
				name = icons.FolderOpen
			}
		}
		return name, th.Accent
	}
	name := n.Icon
	if name.Empty() {
		name = icons.File
	}
	return name, th.ForegroundMuted
}

func (t *Tree) paint(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := t.host.Frame
	vp := t.contentViewport()
	if t.Background != nil {
		dl.AddRect(f, *t.Background)
	}

	dl.PushClip(vp)

	first := int(t.scrollY / t.rowH)
	if first < 0 {
		first = 0
	}
	for i := first; i < len(t.visible); i++ {
		y := f.Y + float32(i)*t.rowH - t.scrollY
		if y >= vp.Y+vp.H {
			break
		}
		n := t.visible[i]

		if t.dragging && t.visible[i] == t.dragNode {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: vp.W, H: t.rowH}, th.ListHover)
		} else if t.dragging && i == t.dragTargetIdx && t.dragPos == DropInside {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: vp.W, H: t.rowH}, th.ListActive)
		} else if t.focused && i == t.selected {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: vp.W, H: t.rowH}, th.ListActive)
		} else if i == t.hover {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: vp.W, H: t.rowH}, th.ListHover)
		}

		baseX := f.X - t.scrollX + t.padX() + float32(n.depth)*t.indent()
		chevW := t.chevW()
		iconW := t.iconW()
		style := th.Typography.Body

		if n.branch() {
			chev := t.ChevronClosed
			if n.expanded {
				chev = t.ChevronOpen
			}
			r := render.Rect{X: baseX, Y: y + (t.rowH-chevW)/2, W: chevW, H: chevW}
			frameIcons().Draw(dl, chev, r, th.ForegroundMuted)
		}

		iconX := baseX + chevW
		name, col := t.iconFor(n)
		frameIcons().Draw(dl, name, render.Rect{X: iconX, Y: y + (t.rowH-iconW)/2, W: iconW, H: iconW}, col)

		_, lh := text.MeasureAt(n.Label, style.Size)
		text.DrawStringTopAt(dl, n.Label, iconX+iconW+t.labelGap(), y+(t.rowH-lh)/2, th.Foreground, style.Size)
	}

	// Draw drop indicator on top of row content.
	if t.dragging && t.dragTargetIdx >= 0 {
		tIdx := t.dragTargetIdx
		ty := f.Y + float32(tIdx)*t.rowH - t.scrollY
		switch t.dragPos {
		case DropBefore:
			dl.AddRect(render.Rect{X: f.X, Y: ty, W: vp.W, H: 2}, th.Accent)
		case DropAfter:
			dl.AddRect(render.Rect{X: f.X, Y: ty + t.rowH - 2, W: vp.W, H: 2}, th.Accent)
			// DropInside highlight is drawn in the row loop (before text) so it never covers it.
		}
	}

	dl.PopClip()

	// Draw ghost — rendered outside the scroll clip so it sits on top.
	// The layout still scissors this to the tree element's frame, which is fine
	// because drag targets are always within the tree.
	if t.dragging && t.dragNode != nil {
		src := t.dragNode
		style := th.Typography.Body
		lw, lh := text.MeasureAt(src.Label, style.Size)
		iconW := t.iconW()
		chevW := t.chevW()
		gw := t.padX() + chevW + iconW + t.labelGap() + lw + t.padX()
		gx := t.dragX + 12
		gy := t.dragY - t.rowH/2

		dl.AddRect(render.Rect{X: gx, Y: gy, W: gw, H: t.rowH}, th.ListActive)

		baseX := gx + t.padX() + chevW
		name, col := t.iconFor(src)
		frameIcons().Draw(dl, name, render.Rect{X: baseX, Y: gy + (t.rowH-iconW)/2, W: iconW, H: iconW}, col)

		text.DrawStringTopAt(dl, src.Label, baseX+iconW+t.labelGap(), gy+(t.rowH-lh)/2, th.Foreground, style.Size)
	}
}

// overScrollbar reports whether the cursor is over a currently-visible bar.
func (t *Tree) overScrollbar(m *input.Mouse) bool {
	_, _, vShow, hShow := t.scrollMetrics()
	if vShow && t.vbar.host.Frame.Contains(m.X, m.Y) {
		return true
	}
	if hShow && t.hbar.host.Frame.Contains(m.X, m.Y) {
		return true
	}
	return false
}

func (t *Tree) onMouse(el *layout.Element, m *input.Mouse) {
	// While dragging, consume events globally so the drag isn't lost when the
	// cursor leaves the widget bounds or passes over a scrollbar.
	if t.dragging {
		if m.Released {
			t.finishDrag()
		} else if m.Down {
			if el.Frame.Contains(m.X, m.Y) && !t.overScrollbar(m) {
				t.updateDragTarget(el, m)
			}
			m.Consumed = true
		} else {
			t.dragging = false
			t.dragNode = nil
			t.dragTargetIdx = -1
		}
		return
	}

	prevHover := t.hover
	t.hover = -1
	if el.Frame.Contains(m.X, m.Y) && (m.ScrollY != 0 || m.ScrollX != 0) {
		vp := t.contentViewport()
		t.vbar.ApplyWheel(m, el.Frame)
		t.hbar.ApplyWheel(m, vp)
		t.clampScroll()
		if t.markPaint != nil {
			t.markPaint()
		}
	}
	if !el.Frame.Contains(m.X, m.Y) || t.overScrollbar(m) {
		if !m.Down {
			t.dragNode = nil
		}
		if prevHover != -1 && t.markPaint != nil {
			t.markPaint()
		}
		return
	}
	idx := int((m.Y - el.Frame.Y + t.scrollY) / t.rowH)
	if idx < 0 || idx >= len(t.visible) {
		if prevHover != -1 && t.markPaint != nil {
			t.markPaint()
		}
		return
	}
	t.hover = idx
	if t.hover != prevHover && t.markPaint != nil {
		t.markPaint()
	}
	t.selected = idx
	n := t.visible[idx]

	if m.RightPressed && t.ContextMenu != nil {
		if items := t.ContextMenu(n); len(items) > 0 {
			t.menu.SetItems(items)
			t.menu.OpenAt(m.X, m.Y)
			m.Consumed = true
			if t.markPaint != nil {
				t.markPaint()
			}
			return
		}
	}

	if m.Pressed {
		t.dragNode = t.visible[idx]
		t.dragStartX = m.X
		t.dragStartY = m.Y
		m.Consumed = true
		return
	}

	// Start dragging once the cursor moves beyond the threshold.
	if m.Down && t.dragNode != nil && t.OnDrop != nil {
		dx := m.X - t.dragStartX
		dy := m.Y - t.dragStartY
		if dx*dx+dy*dy > dragThreshold*dragThreshold {
			t.dragging = true
			t.updateDragTarget(el, m)
			m.Consumed = true
			return
		}
	}

	if m.Released {
		if n.branch() {
			t.toggle(n)
		} else if t.OnActivate != nil {
			t.OnActivate(n)
		}
	}
}

func (t *Tree) updateDragTarget(el *layout.Element, m *input.Mouse) {
	t.dragX = m.X
	t.dragY = m.Y

	if len(t.visible) == 0 {
		return
	}
	idx := int((m.Y - el.Frame.Y + t.scrollY) / t.rowH)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(t.visible) {
		idx = len(t.visible) - 1
	}
	t.dragTargetIdx = idx
	rowTop := el.Frame.Y + float32(idx)*t.rowH - t.scrollY
	relY := m.Y - rowTop
	n := t.visible[idx]
	if n.branch() {
		// Top half: insert before; bottom half: drop into the branch.
		if relY < t.rowH/2 {
			t.dragPos = DropBefore
		} else {
			t.dragPos = DropInside
		}
	} else {
		if relY < t.rowH/2 {
			t.dragPos = DropBefore
		} else {
			t.dragPos = DropAfter
		}
	}

	// Auto-expand: open a collapsed branch after the cursor lingers over it.
	if n.branch() && !n.expanded {
		if t.dragHoverNode == n {
			if time.Since(t.dragHoverStart) >= dragExpandDelay {
				t.toggle(n)
				t.dragHoverNode = nil
			}
		} else {
			t.dragHoverNode = n
			t.dragHoverStart = time.Now()
		}
	} else {
		t.dragHoverNode = nil
	}
}

func (t *Tree) finishDrag() {
	tgt := t.dragTargetIdx
	if t.OnDrop != nil && t.dragNode != nil && tgt >= 0 && t.visible[tgt] != t.dragNode {
		t.OnDrop(DropEvent{
			Source: t.dragNode,
			Target: t.visible[tgt],
			Pos:    t.dragPos,
		})
	}
	t.dragging = false
	t.dragNode = nil
	t.dragTargetIdx = -1
	t.dragHoverNode = nil
}

// Focus grants keyboard focus to the tree.
func (t *Tree) Focus() {
	t.focused = true
	if t.selected < 0 && len(t.visible) > 0 {
		t.selected = 0
	}
	t.ensureSelectedVisible()
}

// Blur removes keyboard focus from the tree.
func (t *Tree) Blur() { t.focused = false }

// Focused reports whether the tree has keyboard focus.
func (t *Tree) Focused() bool { return t.focused }

// CapturesTab reports that plain Tab should move focus rather than act on the tree.
func (t *Tree) CapturesTab() bool { return false }

// FocusOnClick reports that clicking the tree should grant focus.
func (t *Tree) FocusOnClick() bool { return true }

// FocusEl returns the element used for click-to-focus hit testing.
func (t *Tree) FocusEl() *layout.Element { return t.host }

// HandleText is a no-op; the tree does not accept text input.
func (t *Tree) HandleText(_ []rune) {}

// HandleKeys processes arrow-key navigation and Enter for the focused tree.
func (t *Tree) HandleKeys(keys []input.KeyEvent) {
	if !t.focused || len(t.visible) == 0 {
		return
	}
	if t.selected < 0 {
		t.selected = 0
	}
	for _, ev := range keys {
		if ev.Mods != 0 {
			continue
		}
		switch ev.Key {
		case input.KeyUp:
			if t.selected > 0 {
				t.selected--
				t.ensureSelectedVisible()
			}
		case input.KeyDown:
			if t.selected < len(t.visible)-1 {
				t.selected++
				t.ensureSelectedVisible()
			}
		case input.KeyRight:
			n := t.visible[t.selected]
			if n.branch() && !n.expanded {
				t.toggle(n)
			} else if t.selected < len(t.visible)-1 {
				t.selected++
				t.ensureSelectedVisible()
			}
		case input.KeyLeft:
			n := t.visible[t.selected]
			if n.branch() && n.expanded {
				t.toggle(n)
			} else if n.parent != nil && n.parent != t.root {
				for i, v := range t.visible {
					if v == n.parent {
						t.selected = i
						t.ensureSelectedVisible()
						break
					}
				}
			}
		case input.KeyEnter:
			n := t.visible[t.selected]
			if n.branch() {
				t.toggle(n)
			} else if t.OnActivate != nil {
				t.OnActivate(n)
			}
		}
	}
}

func (t *Tree) ensureSelectedVisible() {
	if t.selected < 0 || t.selected >= len(t.visible) {
		return
	}
	top := float32(t.selected) * t.rowH
	bottom := top + t.rowH
	_, clientH, _, _ := t.scrollMetrics()
	if top < t.scrollY {
		t.scrollY = top
	} else if bottom > t.scrollY+clientH {
		t.scrollY = bottom - clientH
	}
	t.clampScroll()
}

// Update drives the scrollbars. Call once per frame after layout.
func (t *Tree) Update(m *input.Mouse) {
	t.computeContentSize()
	_, _, vShow, hShow := t.scrollMetrics()
	t.syncScrollbarLayout(vShow, hShow)
	vp := t.contentViewport()
	t.vbar.Update(m, vp)
	t.hbar.Update(m, vp)
	t.clampScroll()
}
