package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const (
	splitHandleHit = 6
	minPaneSize    = 80
)

// splitView is a Column/Row of panes with draggable handles. Pane sizes live in
// the widget store keyed by id so drag state survives the per-frame rebuild.
type splitView struct {
	id    string
	axis  Axis
	panes []View
	sizes []float32
	spec  Spec
}

type splitState struct {
	axis          Axis
	sizes         []float32
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
	sizes := make([]float32, len(panes))
	return &splitView{id: id, axis: axis, panes: panes, sizes: sizes}
}

// Sizes sets initial main-axis sizes. 0 means the pane flexes. Dragging updates
// stored sizes; later Sizes() calls do not overwrite an already-dragged store.
func (s *splitView) Sizes(sizes ...float32) *splitView {
	for i := 0; i < len(sizes) && i < len(s.sizes); i++ {
		s.sizes[i] = sizes[i]
	}
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
	initAxis := s.axis
	nPanes := len(s.panes)
	st := c.Widget(id, func() any {
		return &splitState{
			axis:  initAxis,
			sizes: initSizes,
			hover: make([]bool, nPanes-1),
		}
	}).(*splitState)
	st.axis = s.axis
	if len(st.sizes) != nPanes {
		st.sizes = append([]float32(nil), s.sizes...)
		st.hover = make([]bool, nPanes-1)
	}

	paneEls := layoutViews(c, s.panes)
	for i := range paneEls {
		applyPaneStyle(paneEls[i], st.axis, st.sizes[i])
	}

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
			h.OnMouse = mouseSplitHandle(st, idx)
			children = append(children, h)
		}
	}
	box := applyLayoutSpec(layout.Box().Direction(dir).FlexGrow(1), s.spec)
	root := layout.New(box, children...)
	st.root = root
	st.panes = paneEls
	return root
}

func handleStyle(axis Axis) layout.Style {
	style := layout.Box().FlexShrink(0)
	if axis == Horizontal {
		return style.W(splitHandleHit)
	}
	return style.H(splitHandleHit)
}

func applyPaneStyle(el *layout.Element, axis Axis, size float32) {
	style := layout.Box().FlexShrink(0)
	if size > 0 {
		if axis == Horizontal {
			style = style.W(size)
		} else {
			style = style.H(size)
		}
	} else {
		style = style.FlexGrow(1)
		if axis == Horizontal {
			style.MinWidth = minPaneSize
		} else {
			style.MinHeight = minPaneSize
		}
	}
	el.Style = style
	el.ReapplyStyle()
}

func (st *splitState) resizeTarget(handleIdx int) int {
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
		return minPaneSize
	}
	if st.axis == Horizontal {
		total = st.root.Frame.W
	} else {
		total = st.root.Frame.H
	}
	total -= float32(len(st.sizes)-1) * splitHandleHit
	for j, sz := range st.sizes {
		if j == i {
			continue
		}
		if sz > 0 {
			total -= sz
		} else {
			total -= minPaneSize
		}
	}
	return f32max(minPaneSize, total)
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
		th := theme.Current()
		col := th.Border
		if active {
			col = th.Accent
		}
		var line render.Rect
		if st.axis == Horizontal {
			cx := f.X + (f.W-1)/2
			line = render.Rect{X: cx, Y: f.Y, W: 1, H: f.H}
		} else {
			cy := f.Y + (f.H-1)/2
			line = render.Rect{X: f.X, Y: cy, W: f.W, H: 1}
		}
		dl.AddRect(line, col)
	}
}

func mouseSplitHandle(st *splitState, idx int) layout.MouseFunc {
	return func(e *layout.Element, m *input.Mouse) {
		inside := e.Frame.Contains(m.X, m.Y)
		if idx < len(st.hover) {
			st.hover[idx] = inside
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
				newSize = clampf(newSize, minPaneSize, st.maxSizeForSection(st.dragSection))
				st.sizes[st.dragSection] = newSize
				if st.dragSection < len(st.panes) {
					applyPaneStyle(st.panes[st.dragSection], st.axis, newSize)
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
			st.dragStartSize = st.sizes[st.dragSection]
			m.Consumed = true
		}
	}
}
