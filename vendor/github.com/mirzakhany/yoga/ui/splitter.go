package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const (
	// splitHandleSize is the handle's layout footprint: just the painted line,
	// so panes sit flush against it with no gap on either side.
	splitHandleSize = 1
	// splitHandleHit is the pointer hit strip, centered on the line. It
	// overhangs the neighbouring panes instead of taking layout space.
	splitHandleHit = 6
	minPaneSize    = 80
)

// splitView is a Column/Row of panes with draggable handles. Pane sizes live in
// the widget store keyed by id so drag state survives the per-frame rebuild.
type splitView struct {
	id            string
	axis          Axis
	panes         []View
	sizes         []float32
	percents      []float32
	usePercent    bool
	mins          []float32
	maxs          []float32
	handleOnHover bool
	spec          Spec
}

type splitState struct {
	axis          Axis
	sizes         []float32
	percents      []float32
	usePercent    bool
	mins          []float32
	maxs          []float32
	handleOnHover bool
	dragging      bool
	dragHandle    int
	dragStart     float32
	dragStartSize float32
	dragSection   int
	hover         []bool
	root          *layout.Element
	panes         []*layout.Element
}

// Splitter arranges panes along axis with draggable handles between them.
// sizes of 0 (the default) flex; Sizes() sets initial main-axis pixels.
func Splitter(id string, axis Axis, panes ...View) *splitView {
	n := len(panes)
	return &splitView{
		id:       id,
		axis:     axis,
		panes:    panes,
		sizes:    make([]float32, n),
		percents: make([]float32, n),
		mins:     make([]float32, n),
		maxs:     make([]float32, n),
	}
}

// Sizes sets initial main-axis sizes in pixels. 0 means the pane flexes.
// Dragging updates stored sizes; later Sizes() calls do not overwrite an
// already-dragged store. Clears percent mode.
func (s *splitView) Sizes(sizes ...float32) *splitView {
	s.usePercent = false
	for i := 0; i < len(sizes) && i < len(s.sizes); i++ {
		s.sizes[i] = sizes[i]
	}
	return s
}

// Percents sets initial main-axis sizes as percentages of available space
// (0–100). 0 means the pane takes an equal share of what is left of 100.
// Percents are flex ratios, so they resolve in the same layout pass (no
// first-frame jump) and are normalized when they do not sum to 100.
// Mutually exclusive with Sizes for the initial store seed.
func (s *splitView) Percents(percents ...float32) *splitView {
	s.usePercent = true
	for i := 0; i < len(percents) && i < len(s.percents); i++ {
		s.percents[i] = percents[i]
	}
	return s
}

// MinSizes sets per-pane minimum main-axis sizes in pixels. Unset or 0 keeps
// the default of 80.
func (s *splitView) MinSizes(mins ...float32) *splitView {
	for i := 0; i < len(mins) && i < len(s.mins); i++ {
		s.mins[i] = mins[i]
	}
	return s
}

// MaxSizes sets per-pane maximum main-axis sizes in pixels. 0 means no
// explicit max (still limited by sibling mins).
func (s *splitView) MaxSizes(maxs ...float32) *splitView {
	for i := 0; i < len(maxs) && i < len(s.maxs); i++ {
		s.maxs[i] = maxs[i]
	}
	return s
}

// HandleOnHover paints the handle line only while hovered or dragging.
// The hit strip stays active so drag still works when the line is hidden.
func (s *splitView) HandleOnHover() *splitView {
	s.handleOnHover = true
	return s
}

// Grow sets flex grow on the splitter itself.
func (s *splitView) Grow(v float32) *splitView {
	s.spec.grow = v
	s.spec.hasGrow = true
	return s
}

func (s *splitView) Layout(c *Ctx) *layout.Element {
	if len(s.panes) < 2 {
		panic("splitter: need at least two panes")
	}
	id := s.id
	if id == "" {
		id = autoID(c, "split")
	}
	initSizes := append([]float32(nil), s.sizes...)
	initPercents := append([]float32(nil), s.percents...)
	initMins := append([]float32(nil), s.mins...)
	initMaxs := append([]float32(nil), s.maxs...)
	initAxis := s.axis
	initUsePercent := s.usePercent
	nPanes := len(s.panes)
	st := c.Widget(id, func() any {
		return &splitState{
			axis:       initAxis,
			sizes:      initSizes,
			percents:   initPercents,
			usePercent: initUsePercent,
			mins:       initMins,
			maxs:       initMaxs,
			hover:      make([]bool, nPanes-1),
		}
	}).(*splitState)
	st.axis = s.axis
	st.handleOnHover = s.handleOnHover
	st.syncConstraints(s.mins, s.maxs)
	if len(st.sizes) != nPanes {
		st.sizes = append([]float32(nil), s.sizes...)
		st.percents = append([]float32(nil), s.percents...)
		st.usePercent = s.usePercent
		st.mins = append([]float32(nil), s.mins...)
		st.maxs = append([]float32(nil), s.maxs...)
		st.hover = make([]bool, nPanes-1)
	}

	paneEls := layoutViews(c, s.panes)
	st.applyPaneStyles(paneEls)

	dir := layout.Column
	if st.axis == Horizontal {
		dir = layout.Row
	}
	children := make([]*layout.Element, 0, len(paneEls)*2-1)
	for i, p := range paneEls {
		children = append(children, p)
		if i < len(paneEls)-1 {
			h := layout.New(handleStyle(st.axis))
			idx := i
			h.Paint = paintSplitHandle(st, idx)
			h.OnMouse = mouseSplitHandle(c, st, idx)
			children = append(children, h)
		}
	}
	box := applyLayoutSpec(layout.Box().Direction(dir).FlexGrow(1), s.spec)
	root := layout.New(box, children...)
	if st.usePercent {
		root.AfterLayout = func(*layout.Element) bool { return st.enforcePaneLimits(paneEls) }
	}
	st.root = root
	st.panes = paneEls
	return root
}

func (st *splitState) syncConstraints(mins, maxs []float32) {
	if len(st.mins) != len(mins) {
		st.mins = append([]float32(nil), mins...)
	} else {
		copy(st.mins, mins)
	}
	if len(st.maxs) != len(maxs) {
		st.maxs = append([]float32(nil), maxs...)
	} else {
		copy(st.maxs, maxs)
	}
}

func (st *splitState) minFor(i int) float32 {
	if i >= 0 && i < len(st.mins) && st.mins[i] > 0 {
		return st.mins[i]
	}
	return minPaneSize
}

func (st *splitState) userMaxFor(i int) float32 {
	if i >= 0 && i < len(st.maxs) && st.maxs[i] > 0 {
		return st.maxs[i]
	}
	return 0
}

func (st *splitState) availableMain() float32 {
	if st.root == nil {
		return 0
	}
	var total float32
	if st.axis == Horizontal {
		total = st.root.Frame.W
	} else {
		total = st.root.Frame.H
	}
	if total <= 0 {
		return 0
	}
	return total - float32(len(st.sizes)-1)*splitHandleSize - st.paneMargins()
}

// paneMargins sums the panes' main-axis margins, which take space from the
// splitter before the panes are sized.
func (st *splitState) paneMargins() float32 {
	var sum float32
	for _, p := range st.panes {
		if p == nil {
			continue
		}
		m := p.Style.Margin
		if st.axis == Horizontal {
			sum += m.Left + m.Right
		} else {
			sum += m.Top + m.Bottom
		}
	}
	return sum
}

// applyPaneStyles styles panes for the current mode. Percent panes become flex
// items with basis 0 and grow equal to their percent, so the solver splits the
// splitter's own size in the same pass that computes it; nothing depends on a
// previous frame's geometry. Pixel mode uses fixed sizes from st.sizes.
func (st *splitState) applyPaneStyles(panes []*layout.Element) {
	if !st.usePercent {
		for i, p := range panes {
			applyPaneStyle(p, st.axis, st.sizes[i], st.minFor(i))
		}
		return
	}
	grows := st.percentGrows()
	for i, p := range panes {
		applyPercentPaneStyle(p, grows[i])
	}
}

// percentGrows returns per-pane flex-grow factors in percent mode. Panes with a
// percent use it directly; unset (0) panes share whatever is left of 100.
func (st *splitState) percentGrows() []float32 {
	grows := make([]float32, len(st.sizes))
	var sum float32
	flex := 0
	for i := range grows {
		if i < len(st.percents) && st.percents[i] > 0 {
			grows[i] = st.percents[i]
			sum += grows[i]
		} else {
			flex++
		}
	}
	if flex > 0 {
		share := f32max(0, 100-sum) / float32(flex)
		for i := range grows {
			if i >= len(st.percents) || st.percents[i] <= 0 {
				grows[i] = share
			}
		}
	}
	return grows
}

// enforcePaneLimits runs after the percent split is solved. Panes the split
// pushed outside their min/max are pinned at that bound; the relayout pass then
// re-splits the remaining space among the others by ratio. Min/max cannot live
// on the pane style itself because the solver clamps the flex basis to the min,
// which would skew the ratios.
func (st *splitState) enforcePaneLimits(panes []*layout.Element) (relayout bool) {
	for i, p := range panes {
		w, h := p.LayoutSize()
		size := h
		if st.axis == Horizontal {
			size = w
		}
		lo, hi := st.minFor(i), st.userMaxFor(i)
		target := size
		if size < lo-0.5 {
			target = lo
		} else if hi > 0 && size > hi+0.5 {
			target = hi
		}
		if target != size {
			applyPaneStyle(p, st.axis, target, lo)
			relayout = true
		}
	}
	return relayout
}

// syncPercentsFromFrames rewrites percents to match the panes as laid out, so a
// drag starts from what is on screen (after min/max pinning and ratio
// normalization) rather than from the seeded values.
func (st *splitState) syncPercentsFromFrames() {
	avail := st.availableMain()
	if avail <= 0 {
		return
	}
	for i, p := range st.panes {
		if i >= len(st.percents) || st.percents[i] <= 0 || p == nil {
			continue
		}
		st.percents[i] = 100 * st.paneMain(p) / avail
	}
}

func (st *splitState) paneMain(p *layout.Element) float32 {
	if st.axis == Horizontal {
		return p.Frame.W
	}
	return p.Frame.H
}

func (st *splitState) clampSize(i int, size float32) float32 {
	lo := st.minFor(i)
	hi := st.maxSizeForSection(i)
	if um := st.userMaxFor(i); um > 0 {
		hi = f32min(hi, um)
	}
	if hi < lo {
		hi = lo
	}
	return clampf(size, lo, hi)
}

func handleStyle(axis Axis) layout.Style {
	style := layout.Box().FlexShrink(0)
	if axis == Horizontal {
		return style.W(splitHandleSize)
	}
	return style.H(splitHandleSize)
}

// paneItemStyle keeps the pane's own container styling (direction, padding,
// background, margin, …) and resets only its flex-item fields, which the
// splitter owns.
// Replacing the whole style would, e.g., turn a nested splitter row into a
// column.
func paneItemStyle(el *layout.Element) layout.Style {
	def := layout.Box()
	s := el.Style
	s.SelfAlign, s.Pos = def.SelfAlign, def.Pos
	s.Left, s.Top, s.Right, s.Bottom = def.Left, def.Top, def.Right, def.Bottom
	s.Grow, s.Shrink, s.Basis = def.Grow, 0, def.Basis
	s.Width, s.Height = def.Width, def.Height
	s.MinWidth, s.MinHeight, s.MaxWidth, s.MaxHeight = def.MinWidth, def.MinHeight, def.MaxWidth, def.MaxHeight
	return s
}

func applyPercentPaneStyle(el *layout.Element, grow float32) {
	el.Style = paneItemStyle(el).FlexGrow(grow).FlexBasis(0)
	el.ReapplyStyle()
}

func applyPaneStyle(el *layout.Element, axis Axis, size, min float32) {
	style := paneItemStyle(el)
	if size > 0 {
		if axis == Horizontal {
			style = style.W(size)
		} else {
			style = style.H(size)
		}
	} else {
		style = style.FlexGrow(1)
		if axis == Horizontal {
			style.MinWidth = min
		} else {
			style.MinHeight = min
		}
	}
	el.Style = style
	el.ReapplyStyle()
}

func (st *splitState) resizeTarget(handleIdx int) int {
	if st.usePercent {
		if handleIdx < len(st.percents) && st.percents[handleIdx] > 0 {
			return handleIdx
		}
		if handleIdx+1 < len(st.percents) && st.percents[handleIdx+1] > 0 {
			return handleIdx + 1
		}
		return handleIdx
	}
	if st.sizes[handleIdx] > 0 {
		return handleIdx
	}
	if handleIdx+1 < len(st.sizes) && st.sizes[handleIdx+1] > 0 {
		return handleIdx + 1
	}
	return handleIdx
}

func (st *splitState) maxSizeForSection(i int) float32 {
	var total float32
	if st.root == nil {
		return st.minFor(i)
	}
	if st.axis == Horizontal {
		total = st.root.Frame.W
	} else {
		total = st.root.Frame.H
	}
	total -= float32(len(st.sizes)-1)*splitHandleSize + st.paneMargins()
	for j, sz := range st.sizes {
		if j == i {
			continue
		}
		// Percent-sized siblings shrink with the drag, so only reserve their min.
		// Pixel-fixed siblings stay put and reserve their full size (flex uses min).
		if st.usePercent && j < len(st.percents) && st.percents[j] > 0 {
			total -= st.minFor(j)
			continue
		}
		if sz > 0 {
			total -= sz
		} else {
			total -= st.minFor(j)
		}
	}
	return f32max(st.minFor(i), total)
}

func (st *splitState) pointerAlong(m *input.Mouse) float32 {
	if st.axis == Horizontal {
		return m.X
	}
	return m.Y
}

func paintSplitHandle(st *splitState, idx int) layout.PaintFunc {
	return func(dl *render.DrawList, _ *shape.Engine) {
		// The handle is the paint target; look it up from the current root.
		if st.root == nil || idx < 0 {
			return
		}
		hi := idx*2 + 1
		if hi >= len(st.root.Children) {
			return
		}
		f := st.root.Children[hi].Frame
		active := (idx < len(st.hover) && st.hover[idx]) || (st.dragging && st.dragHandle == idx)
		if st.handleOnHover && !active {
			return
		}
		th := theme.Current()
		col := th.Border
		if active {
			col = th.Accent
		}
		dl.AddRect(f, col)
	}
}

// handleHitRect widens the handle frame to splitHandleHit along the main axis,
// centered on the line.
func handleHitRect(f render.Rect, axis Axis) render.Rect {
	pad := float32(splitHandleHit-splitHandleSize) / 2
	if axis == Horizontal {
		f.X -= pad
		f.W += 2 * pad
	} else {
		f.Y -= pad
		f.H += 2 * pad
	}
	return f
}

func mouseSplitHandle(c *Ctx, st *splitState, idx int) layout.MouseFunc {
	return func(e *layout.Element, m *input.Mouse) {
		inside := handleHitRect(e.Frame, st.axis).Contains(m.X, m.Y)
		if idx < len(st.hover) {
			trackHover(c, &st.hover[idx], inside)
		}
		setResize := func() {
			if st.axis == Horizontal {
				m.SetCursor(input.CursorResizeEW)
			} else {
				m.SetCursor(input.CursorResizeNS)
			}
		}

		if st.dragging && st.dragHandle == idx {
			setResize()
			if m.Down {
				delta := st.pointerAlong(m) - st.dragStart
				newSize := st.dragStartSize
				if st.dragSection <= st.dragHandle {
					newSize += delta
				} else {
					newSize -= delta
				}
				newSize = st.clampSize(st.dragSection, newSize)
				st.sizes[st.dragSection] = newSize
				if st.usePercent {
					if avail := st.availableMain(); avail > 0 {
						oldPct := st.percents[st.dragSection]
						newPct := 100 * newSize / avail
						st.percents[st.dragSection] = newPct
						// Keep the pair across this handle summing to the same total.
						other := st.dragHandle + 1
						if st.dragSection == st.dragHandle+1 {
							other = st.dragHandle
						}
						if other >= 0 && other < len(st.percents) && st.percents[other] > 0 {
							st.percents[other] = f32max(0, st.percents[other]-(newPct-oldPct))
						}
					}
					st.applyPaneStyles(st.panes)
				} else if st.dragSection < len(st.panes) {
					applyPaneStyle(st.panes[st.dragSection], st.axis, newSize, st.minFor(st.dragSection))
				}
				// Marking paint during input rebuilds Body so the new sizes are
				// solved this frame instead of presenting the stale input tree.
				if c != nil {
					c.MarkNeedsPaint()
				}
				m.Consumed = true
			} else {
				st.dragging = false
			}
			return
		}

		if inside {
			setResize()
		}
		if m.Pressed && inside {
			st.dragging = true
			st.dragHandle = idx
			st.dragStart = st.pointerAlong(m)
			st.dragSection = st.resizeTarget(idx)
			if st.usePercent {
				st.syncPercentsFromFrames()
				if st.dragSection < len(st.panes) && st.panes[st.dragSection] != nil {
					st.sizes[st.dragSection] = st.paneMain(st.panes[st.dragSection])
				}
			}
			// Ensure the drag target has a concrete pixel size (flex panes start at 0).
			if st.sizes[st.dragSection] <= 0 && st.dragSection < len(st.panes) && st.panes[st.dragSection] != nil {
				if st.axis == Horizontal {
					st.sizes[st.dragSection] = st.panes[st.dragSection].Frame.W
				} else {
					st.sizes[st.dragSection] = st.panes[st.dragSection].Frame.H
				}
			}
			st.dragStartSize = st.sizes[st.dragSection]
			m.Consumed = true
		}
	}
}
