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

// DropEvent is passed to Tree.OnDrop when the user completes a drag. A nil
// Target means the empty space below the last row: the drop lands at the top
// level, as a child of the (undrawn) root, and Pos is always DropInside.
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

	// Leading, when set and returning a view, draws that view in place of a
	// node's icon: a method badge, a color dot. Return nil for the icon.
	Leading func(n *TreeNode) View
	// Trailing, when set and returning a view, draws it right-aligned in the
	// node's row, such as a count or a status dot. The label is clipped
	// before it.
	Trailing func(n *TreeNode) View

	// ContextMenu builds the right-click menu items for a node. n is nil for
	// a right-click on the empty space below the rows.
	ContextMenu func(n *TreeNode) []MenuItem

	// OnActivate fires when a leaf is clicked. OnToggle fires after a branch expands/collapses.
	OnActivate func(n *TreeNode)
	OnToggle   func(n *TreeNode)
	// OnDrop fires when the user drops a dragged node. Set this to enable
	// drag-and-drop. The handler is responsible for mutating the data model
	// and calling Rebuild as needed.
	OnDrop func(ev DropEvent)

	// CanDrop optionally vetoes a prospective drop. It is called continuously
	// while dragging; returning false suppresses the drop indicator and stops
	// OnDrop from firing for that target. Dropping a node onto itself or into
	// its own subtree is always rejected before CanDrop is consulted.
	CanDrop func(ev DropEvent) bool

	hover    int
	selected int
	active   *TreeNode // highlighted whether or not the tree has focus (Reveal, SetActive)
	focused  bool

	// Leading/Trailing views of the visible rows, in a layer clipped to the
	// content viewport. slotFirst is the row index of leads[0] and trails[0].
	slotLayer     *layout.Element
	slotFirst     int
	leads, trails []*layout.Element
	rowH          float32
	markPaint     func()

	// drag-and-drop state
	dragging      bool
	dragNode      *TreeNode // the node being dragged; pointer survives t.visible rebuilds
	dragStartX    float32
	dragStartY    float32
	dragX, dragY  float32 // current cursor position during drag (for ghost rendering)
	dragTargetIdx int     // prospective drop-target index in t.visible (-1 = the root, i.e. the empty area below the rows)
	dragPos       DropPos // placement within dragTargetIdx
	dragValid     bool    // whether the prospective drop is allowed (see allowDrop)

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
		rowH:          treeRowHeight(th),
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

	t.slotLayer = layout.New(layout.Box().AbsLeft(0).AbsTop(0))
	t.slotLayer.Clip = true
	t.host = layout.New(layout.Box().FlexGrow(1), t.slotLayer, t.vbar.host, t.hbar.host)
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
	t.vbar.tick(c, t.host.Frame)
	t.hbar.tick(c, t.host.Frame)
	t.layoutSlots(c)
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
			n.Children = removeAt(n.Children, i)
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
	if t.active != nil && (t.active == n || n.contains(t.active)) {
		t.active = nil
	}
	t.rebuild()
	return true
}

// contains reports whether d is somewhere below n.
func (n *TreeNode) contains(d *TreeNode) bool {
	for _, c := range n.Children {
		if c == d || c.contains(d) {
			return true
		}
	}
	return false
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
	clear(t.visible) // release nodes dropped from the flattened view
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
	vBar, hBar := t.vbar.gutter(t.barSize()), t.hbar.gutter(t.barSize())
	if vShow {
		clientW = f.W - vBar
	}
	hShow = t.contentW > clientW
	if hShow {
		clientH = f.H - hBar
	}
	if t.contentH > clientH {
		vShow = true
		clientW = f.W - vBar
		hShow = t.contentW > clientW
		if hShow {
			clientH = f.H - hBar
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
			bottom = t.hbar.strip(t.barSize())
		}
		t.vbar.host.Style = layout.Box().W(t.vbar.strip(t.barSize())).AbsTop(0).AbsRight(0).AbsBottom(bottom)
		t.vbar.host.ReapplyStyle()
	}
	if hShow {
		right := float32(0)
		if vShow {
			right = t.vbar.strip(t.barSize())
		}
		t.hbar.host.Style = layout.Box().H(t.hbar.strip(t.barSize())).AbsLeft(0).AbsRight(right).AbsBottom(0)
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

// Expanded reports whether a branch is currently open.
func (n *TreeNode) Expanded() bool { return n.expanded }

// SetExpanded opens or closes a branch, loading its children on first open.
// Use it to restore the open branches after replacing the root.
func (t *Tree) SetExpanded(n *TreeNode, open bool) {
	if n == nil || n.Leaf || n.expanded == open {
		return
	}
	t.toggle(n)
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
		} else if t.dragging && t.dragValid && i == t.dragTargetIdx && t.dragPos == DropInside {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: vp.W, H: t.rowH}, th.ListActive)
		} else if t.focused && i == t.selected {
			dl.AddRect(render.Rect{X: f.X, Y: y, W: vp.W, H: t.rowH}, th.ListActive)
		} else if n == t.active {
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
		labelX := iconX + iconW + t.labelGap()
		if lead := t.slotAt(t.leads, i); lead != nil {
			labelX = lead.Frame.X + lead.Frame.W + t.labelGap()
		} else {
			name, col := t.iconFor(n)
			frameIcons().Draw(dl, name, render.Rect{X: iconX, Y: y + (t.rowH-iconW)/2, W: iconW, H: iconW}, col)
		}

		_, lh := text.MeasureAt(n.Label, style.Size)
		if trail := t.slotAt(t.trails, i); trail != nil {
			dl.PushClip(render.Rect{X: vp.X, Y: y, W: f32max(0, trail.Frame.X-t.labelGap()-vp.X), H: t.rowH})
			text.DrawStringTopAt(dl, n.Label, labelX, y+(t.rowH-lh)/2, th.Foreground, style.Size)
			dl.PopClip()
		} else {
			text.DrawStringTopAt(dl, n.Label, labelX, y+(t.rowH-lh)/2, th.Foreground, style.Size)
		}
	}

	// Draw drop indicator on top of row content.
	if t.dragging && t.dragValid {
		if t.dragTargetIdx < 0 {
			// Root target: outline the whole panel, since a line under the last
			// row would read as "after that row" and say nothing about depth.
			dl.AddRect(render.Rect{X: vp.X, Y: vp.Y, W: vp.W, H: 2}, th.Accent)
			dl.AddRect(render.Rect{X: vp.X, Y: vp.Y + vp.H - 2, W: vp.W, H: 2}, th.Accent)
			dl.AddRect(render.Rect{X: vp.X, Y: vp.Y, W: 2, H: vp.H}, th.Accent)
			dl.AddRect(render.Rect{X: vp.X + vp.W - 2, Y: vp.Y, W: 2, H: vp.H}, th.Accent)
		} else {
			ty := f.Y + float32(t.dragTargetIdx)*t.rowH - t.scrollY
			switch t.dragPos {
			case DropBefore:
				dl.AddRect(render.Rect{X: f.X, Y: ty, W: vp.W, H: 2}, th.Accent)
			case DropAfter:
				dl.AddRect(render.Rect{X: f.X, Y: ty + t.rowH - 2, W: vp.W, H: 2}, th.Accent)
				// DropInside highlight is drawn in the row loop (before text) so it never covers it.
			}
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
	if vShow && t.vbar.hit(m.X, m.Y) {
		return true
	}
	if hShow && t.hbar.hit(m.X, m.Y) {
		return true
	}
	return false
}

// SetScrollbars overrides the theme's ScrollbarMode for this tree.
func (t *Tree) SetScrollbars(m theme.ScrollbarMode) {
	t.vbar.SetMode(m)
	t.hbar.SetMode(m)
}

func (t *Tree) onMouse(el *layout.Element, m *input.Mouse) {
	// While dragging, consume events globally so the drag isn't lost when the
	// cursor leaves the widget bounds or passes over a scrollbar.
	if t.dragging {
		if m.Released {
			t.finishDrag()
			if t.markPaint != nil {
				t.markPaint()
			}
		} else if m.Down {
			if el.Frame.Contains(m.X, m.Y) && !t.overScrollbar(m) {
				t.updateDragTarget(el, m)
				// The ghost follows the cursor, so every move needs a frame.
				if t.markPaint != nil {
					t.markPaint()
				}
			}
			m.Consumed = true
		} else {
			t.dragging = false
			t.dragNode = nil
			t.dragTargetIdx = -1
			t.dragValid = false
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
		if m.RightPressed && t.openMenu(nil, m) {
			return
		}
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

	if m.RightPressed && t.openMenu(n, m) {
		return
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

// openMenu opens the context menu for n (nil for empty space) at the cursor
// and reports whether it opened.
func (t *Tree) openMenu(n *TreeNode, m *input.Mouse) bool {
	if t.ContextMenu == nil {
		return false
	}
	items := t.ContextMenu(n)
	if len(items) == 0 {
		return false
	}
	t.menu.SetItems(items)
	t.menu.OpenAt(m.X, m.Y)
	m.Consumed = true
	if t.markPaint != nil {
		t.markPaint()
	}
	return true
}

func (t *Tree) updateDragTarget(el *layout.Element, m *input.Mouse) {
	t.dragX = m.X
	t.dragY = m.Y

	idx := int((m.Y - el.Frame.Y + t.scrollY) / t.rowH)
	if idx < 0 {
		idx = 0
	}
	// Past the last row (or on an empty tree) the drop lands at the top level,
	// which is how a node is dragged out of the branch that holds it.
	if idx >= len(t.visible) {
		t.dragTargetIdx = -1
		t.dragPos = DropInside
		t.dragValid = t.allowDrop(nil, DropInside)
		t.dragHoverNode = nil
		return
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
	t.dragValid = t.allowDrop(n, t.dragPos)

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

// allowDrop reports whether the dragged node may land on target (nil = the
// root). A node can never be dropped on itself or into its own subtree; beyond
// that the application decides through CanDrop.
func (t *Tree) allowDrop(target *TreeNode, pos DropPos) bool {
	src := t.dragNode
	if src == nil {
		return false
	}
	for n := target; n != nil; n = n.parent {
		if n == src {
			return false
		}
	}
	if t.CanDrop != nil {
		return t.CanDrop(DropEvent{Source: src, Target: target, Pos: pos})
	}
	return true
}

func (t *Tree) finishDrag() {
	if t.OnDrop != nil && t.dragNode != nil && t.dragValid {
		var target *TreeNode
		if t.dragTargetIdx >= 0 && t.dragTargetIdx < len(t.visible) {
			target = t.visible[t.dragTargetIdx]
		}
		t.OnDrop(DropEvent{
			Source: t.dragNode,
			Target: target,
			Pos:    t.dragPos,
		})
	}
	t.dragging = false
	t.dragNode = nil
	t.dragTargetIdx = -1
	t.dragValid = false
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

// slotAt is row i's view element in a Leading/Trailing list, or nil. Its
// Frame holds where the view was laid out.
func (t *Tree) slotAt(els []*layout.Element, i int) *layout.Element {
	i -= t.slotFirst
	if i < 0 || i >= len(els) {
		return nil
	}
	return els[i]
}

// layoutSlots builds the Leading and Trailing views of the visible rows. Each
// sits in a row-wide strip so its own frame, read back at paint, says where
// the label goes.
func (t *Tree) layoutSlots(c *Ctx) {
	clear(t.slotLayer.Children)
	t.slotLayer.Children = t.slotLayer.Children[:0]
	clear(t.leads)
	clear(t.trails)
	t.leads, t.trails = t.leads[:0], t.trails[:0]
	if t.Leading == nil && t.Trailing == nil {
		return
	}
	clientW, clientH, _, _ := t.scrollMetrics()
	t.slotLayer.Style.Width, t.slotLayer.Style.Height = clientW, clientH
	t.slotLayer.ReapplyStyle()
	t.slotFirst = max(0, int(t.scrollY/t.rowH))
	// strip lays v out in a row-wide strip and returns v's own element.
	strip := func(v View, left, w, top float32, end bool) *layout.Element {
		kids, at := []View{v, Spacer()}, 0
		if end {
			kids, at = []View{Spacer(), v}, 1
		}
		el := Row(kids...).Align(AlignCenter).Layout(c)
		el.Style.Pos = layout.PositionAbsolute
		el.Style.Left, el.Style.Top = left, top
		el.Style.Width, el.Style.Height = f32max(0, w), t.rowH
		el.ReapplyStyle()
		t.slotLayer.Children = append(t.slotLayer.Children, el)
		return el.Children[at]
	}
	for i := t.slotFirst; i < len(t.visible); i++ {
		top := float32(i)*t.rowH - t.scrollY
		if top >= clientH {
			break
		}
		n := t.visible[i]
		var lead, trail *layout.Element
		if t.Leading != nil {
			if v := t.Leading(n); v != nil {
				left := t.padX() + float32(n.depth)*t.indent() + t.chevW() - t.scrollX
				lead = strip(v, left, clientW-left, top, false)
			}
		}
		if t.Trailing != nil {
			if v := t.Trailing(n); v != nil {
				trail = strip(v, 0, clientW-t.padX(), top, true)
			}
		}
		t.leads = append(t.leads, lead)
		t.trails = append(t.trails, trail)
	}
}

// Selected is the node under the tree's cursor (the keyboard selection), or
// nil.
func (t *Tree) Selected() *TreeNode {
	if t.selected < 0 || t.selected >= len(t.visible) {
		return nil
	}
	return t.visible[t.selected]
}

// Active is the node set by Reveal or SetActive, or nil.
func (t *Tree) Active() *TreeNode { return t.active }

// SetActive highlights n (nil clears it) whether or not the tree has focus,
// as an editor's sidebar marks the open file. It does not expand or scroll;
// see Reveal.
func (t *Tree) SetActive(n *TreeNode) { t.active = n }

// Reveal expands n's ancestors, makes n the selected and active node, and
// scrolls it into view, centering it when it was off screen. It reports
// false, changing nothing, when n is not in the tree (or is hidden by the
// filter).
func (t *Tree) Reveal(n *TreeNode) bool {
	path := t.pathTo(n)
	if path == nil {
		return false
	}
	for _, a := range path[:len(path)-1] {
		if !a.expanded {
			a.expanded = true
			t.ensureLoaded(a)
		}
	}
	t.rebuild()
	idx := -1
	for i, v := range t.visible {
		if v == n {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	t.selected = idx
	t.active = n
	top := float32(idx) * t.rowH
	_, clientH, _, _ := t.scrollMetrics()
	if top < t.scrollY || top+t.rowH > t.scrollY+clientH {
		t.scrollY = top + t.rowH/2 - clientH/2
	}
	t.clampScroll()
	if t.markPaint != nil {
		t.markPaint()
	}
	return true
}

// pathTo returns the nodes from a top-level row down to n, searching the
// loaded children, or nil when n is not among them.
func (t *Tree) pathTo(n *TreeNode) []*TreeNode {
	if n == nil {
		return nil
	}
	var path []*TreeNode
	var walk func(p *TreeNode) bool
	walk = func(p *TreeNode) bool {
		for _, c := range p.Children {
			path = append(path, c)
			if c == n || walk(c) {
				return true
			}
			path = path[:len(path)-1]
		}
		return false
	}
	if !walk(t.root) {
		return nil
	}
	return path
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

// treeRowHeight is a tree row's height for th.
func treeRowHeight(th *theme.Theme) float32 {
	if th.Metrics.TreeRowHeight > 0 {
		return th.Metrics.TreeRowHeight
	}
	return th.Typography.Body.LineHeight + th.Spacing.S
}

// Update drives the scrollbars. Call once per frame after layout.
func (t *Tree) Update(m *input.Mouse) {
	t.rowH = treeRowHeight(theme.Current())
	t.computeContentSize()
	_, _, vShow, hShow := t.scrollMetrics()
	t.syncScrollbarLayout(vShow, hShow)
	vp := t.contentViewport()
	t.vbar.Update(m, vp)
	t.hbar.Update(m, vp)
	t.clampScroll()
}
