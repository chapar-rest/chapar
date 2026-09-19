// Package layout is layer 1 of the framework: a declarative element tree driven
// by a pure-Go layout engine (flex, grid, stack) and a two-pass pipeline.
//
//	Pass 1 (Calculate): the engine solves constraints, assigning each node a
//	        position/size *relative to its parent's content box*.
//	Pass 2 (Flatten):   we walk the tree once, accumulating parent origins into
//	        absolute screen-space rectangles (Element.Frame) that the renderer
//	        and hit-testing consume.
package layout

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

// PaintFunc draws an element's visuals into the shared per-frame draw list.
type PaintFunc func(dl *render.DrawList, text *shape.Engine)

// MouseFunc receives pointer events for an element. Handlers should check
// e.Frame.Contains and may set m.Consumed to stop propagation to elements
// behind them.
type MouseFunc func(e *Element, m *input.Mouse)

// AfterLayoutFunc runs after the solver assigns cw/ch and before Flatten.
// Return true if the hook changed styles that require one more layout pass
// (e.g. scrollbar gutter margin). The engine runs at most one relayout.
type AfterLayoutFunc func(e *Element) (relayout bool)

// Element is a node in the UI tree. It couples layout style, an optional paint
// hook, and an optional input hook. Components (layer 3) build Elements and
// attach those hooks.
type Element struct {
	Style    Style
	Children []*Element

	// Frame is the absolute screen-space rectangle filled in by Flatten (pass 2).
	Frame render.Rect

	Paint       PaintFunc
	OnMouse     MouseFunc
	AfterLayout AfterLayoutFunc

	// Overlay elements (dropdowns, context menus) are painted and hit-tested
	// after — i.e. on top of — the normal tree, regardless of their position in
	// it. This implements simple Z-axis ordering.
	Overlay bool

	// ScrollOffset shifts this element's children upward during Flatten,
	// implementing content scrolling without re-running the layout solver.
	ScrollOffset float32

	// Clip marks a viewport whose subtree is scissored to its Frame during
	// Paint and whose children are skipped by Dispatch when the pointer is
	// outside the Frame (so scrolled-out children neither draw nor steal
	// clicks). Components may additionally push tighter clips in their own
	// paint hooks (the editor uses the flag to drive virtualization too).
	Clip bool

	// Computed layout (relative to parent content box); filled by the engine.
	cx, cy, cw, ch float32
}

// New creates an element with the given style and children. This is the
// declarative constructor: trees are written as nested calls, SwiftUI/Flutter
// style.
//
//	root := layout.New(layout.Box().Direction(layout.Row),
//	    sidebar,
//	    layout.New(layout.Box().FlexGrow(1), editor),
//	)
func New(style Style, children ...*Element) *Element {
	return &Element{Style: style, Children: children}
}

// WithPaint attaches a paint hook and returns the element for chaining.
func (e *Element) WithPaint(fn PaintFunc) *Element { e.Paint = fn; return e }

// WithMouse attaches an input hook and returns the element for chaining.
func (e *Element) WithMouse(fn MouseFunc) *Element { e.OnMouse = fn; return e }

// LayoutSize returns the solver-assigned width/height (cw/ch) from the last
// layoutNode pass. Prefer this over Frame when reading size before Flatten.
func (e *Element) LayoutSize() (w, h float32) {
	return e.cw, e.ch
}

// WithBackground attaches a paint hook that fills the element's frame with c.
func (e *Element) WithBackground(c render.Color) *Element {
	e.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		dl.AddRect(e.Frame, c)
	}
	return e
}

// WithBackgroundPtr fills the element's frame with the color pointed to by c,
// read fresh every frame. This lets a background track a live theme color (the
// pointer stays valid while the theme's contents change on a runtime switch).
func (e *Element) WithBackgroundPtr(c *render.Color) *Element {
	e.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		dl.AddRect(e.Frame, *c)
	}
	return e
}

// Add appends children and returns the element for chaining.
func (e *Element) Add(children ...*Element) *Element {
	e.Children = append(e.Children, children...)
	return e
}

// ── Style modifiers ───────────────────────────────────────────────────────────
// SwiftUI-style modifiers that mutate this element's Style in place and return
// the element for chaining. They let callers configure any element — including a
// component's own .El — without the read-modify-write dance of reassigning
// `el.Style = el.Style.X()`. Each mirrors the like-named Style builder.
//
//	app.input.El.Grow(1)
//	root := layout.VStack(8, title, body).Grow(1).Padding(16).Bg(th.Background)

// Grow sets the flex grow factor (how much spare space this element claims).
func (e *Element) Grow(v float32) *Element { e.Style = e.Style.FlexGrow(v); return e }

// Shrink sets the flex shrink factor.
func (e *Element) Shrink(v float32) *Element { e.Style = e.Style.FlexShrink(v); return e }

// Width sets a fixed width.
func (e *Element) Width(v Px) *Element { e.Style = e.Style.W(v); return e }

// Height sets a fixed height.
func (e *Element) Height(v Px) *Element { e.Style = e.Style.H(v); return e }

// Size sets a fixed width and height.
func (e *Element) Size(w, h Px) *Element { e.Style = e.Style.Size(w, h); return e }

// Padding sets uniform padding on all edges.
func (e *Element) Padding(v Px) *Element { e.Style = e.Style.PaddingAll(v); return e }

// PaddingXY sets horizontal and vertical padding.
func (e *Element) PaddingXY(x, y Px) *Element { e.Style = e.Style.PaddingXY(x, y); return e }

// Gap sets a uniform gap between children.
func (e *Element) Gap(v Px) *Element { e.Style = e.Style.Gap(v); return e }

// Justify sets main-axis distribution of children.
func (e *Element) Justify(j Justify) *Element { e.Style = e.Style.JustifyContent(j); return e }

// Align sets cross-axis alignment of children.
func (e *Element) Align(a Align) *Element { e.Style = e.Style.AlignItems(a); return e }

// Bg fills the element's frame with c (rounded by any Radius). Painted by the
// layout pass — no Paint hook needed. Use BgPtr to track a live theme color.
func (e *Element) Bg(c render.Color) *Element { e.Style = e.Style.Background(c); return e }

// Radius rounds the element's background/border corners.
func (e *Element) Radius(r Px) *Element { e.Style = e.Style.CornerRadius(r); return e }

// Border strokes the element with the given color, width, and corner radius.
func (e *Element) Border(c render.Color, width, radius Px) *Element {
	e.Style = e.Style.Border(c, width, radius)
	return e
}

// BgPtr is the chainable form of WithBackgroundPtr: it fills the frame with the
// color pointed to by c, read fresh each frame so the fill tracks a live theme.
func (e *Element) BgPtr(c *render.Color) *Element { return e.WithBackgroundPtr(c) }

// FlexGrow is an alias for Grow, matching the like-named Style builder and the
// SwiftUI/CSS vocabulary.
func (e *Element) FlexGrow(v float32) *Element { return e.Grow(v) }

// FlexShrink is an alias for Shrink.
func (e *Element) FlexShrink(v float32) *Element { return e.Shrink(v) }

// Per-side padding. These complement Padding (all edges) and PaddingXY.
func (e *Element) PaddingLeft(v Px) *Element   { e.Style.Padding.Left = v; return e }
func (e *Element) PaddingRight(v Px) *Element  { e.Style.Padding.Right = v; return e }
func (e *Element) PaddingTop(v Px) *Element    { e.Style.Padding.Top = v; return e }
func (e *Element) PaddingBottom(v Px) *Element { e.Style.Padding.Bottom = v; return e }

// Margin sets uniform margin on all edges.
func (e *Element) Margin(v Px) *Element { e.Style = e.Style.MarginAll(v); return e }

// MarginXY sets horizontal (x) and vertical (y) margin.
func (e *Element) MarginXY(x, y Px) *Element {
	e.Style.Margin = Edges{Top: y, Bottom: y, Left: x, Right: x}
	return e
}

// Per-side margin.
func (e *Element) MarginLeft(v Px) *Element   { e.Style.Margin.Left = v; return e }
func (e *Element) MarginRight(v Px) *Element  { e.Style.Margin.Right = v; return e }
func (e *Element) MarginTop(v Px) *Element    { e.Style.Margin.Top = v; return e }
func (e *Element) MarginBottom(v Px) *Element { e.Style.Margin.Bottom = v; return e }

// Engine abstracts the layout solver so the backend is swappable.
type Engine interface {
	// Compute runs both layout passes for the tree rooted at root inside the
	// (width, height) box, leaving absolute rectangles in each Element.Frame.
	Compute(root *Element, width, height float32)
}

// DefaultEngine is the active layout engine.
var DefaultEngine Engine = customEngine{}

// Calculate runs the two-pass pipeline using the default engine.
func (e *Element) Calculate(width, height float32) {
	DefaultEngine.Compute(e, width, height)
}

// MarkDirty clears computed geometry so the next Calculate relayouts the subtree.
// Call after structural changes such as opening a menu or adding children.
func (e *Element) MarkDirty() {
	e.cx, e.cy, e.cw, e.ch = 0, 0, 0, 0
	for _, c := range e.Children {
		c.MarkDirty()
	}
}

// ReapplyStyle is a no-op for the custom engine (styles are read fresh each
// Calculate). Kept for API compatibility with components that call it.
func (e *Element) ReapplyStyle() {}

func flatten(e *Element, originX, originY float32) {
	x := originX + e.cx
	y := originY + e.cy
	e.Frame = render.Rect{
		X: x, Y: y,
		W: e.cw,
		H: e.ch,
	}
	contentX := x + e.Style.Padding.Left
	contentY := y + e.Style.Padding.Top - e.ScrollOffset
	for _, c := range e.Children {
		flatten(c, contentX, contentY)
	}
}

// Paint walks the tree and emits geometry in painter's-algorithm order: the
// normal tree first, then overlay subtrees on top.
func Paint(root *Element, dl *render.DrawList, text *shape.Engine) {
	paintBase(root, dl, text)
	forEachOverlayRoot(root, func(o *Element) {
		paintAll(o, dl, text)
	})
}

func paintBase(e *Element, dl *render.DrawList, text *shape.Engine) {
	if e.Overlay {
		return
	}
	if e.Clip {
		dl.PushClip(e.Frame)
	}
	paintDecoration(e, dl)
	if e.Paint != nil {
		e.Paint(dl, text)
	}
	for _, c := range e.Children {
		paintBase(c, dl, text)
	}
	if e.Clip {
		dl.PopClip()
	}
}

func paintAll(e *Element, dl *render.DrawList, text *shape.Engine) {
	if e.Clip {
		dl.PushClip(e.Frame)
	}
	paintDecoration(e, dl)
	if e.Paint != nil {
		e.Paint(dl, text)
	}
	for _, c := range e.Children {
		paintAll(c, dl, text)
	}
	if e.Clip {
		dl.PopClip()
	}
}

// paintDecoration draws the Style-driven background and border declared via
// Style.Background / Style.Border, before the element's own Paint hook.
func paintDecoration(e *Element, dl *render.DrawList) {
	s := e.Style
	widths := s.EffectiveBorderWidths()
	radii := s.EffectiveRadii()
	hasBg := s.BgColor.A > 0
	hasBorder := s.BorderColor.A > 0 && widths.AnyPositive()
	if !hasBg && !hasBorder && !radii.AnyPositive() {
		return
	}
	rc := render.Corners{
		TopLeft: radii.TopLeft, TopRight: radii.TopRight,
		BottomRight: radii.BottomRight, BottomLeft: radii.BottomLeft,
	}
	bw := render.BorderEdges{
		Top: widths.Top, Right: widths.Right,
		Bottom: widths.Bottom, Left: widths.Left,
	}
	fill := s.BgColor
	if !hasBg {
		fill = render.Color{}
	}
	border := s.BorderColor
	if !hasBorder {
		border = render.Color{}
		bw = render.BorderEdges{}
	}
	dl.PaintBox(e.Frame, rc, bw, fill, border, s.BorderStyle)
}

func forEachOverlayRoot(e *Element, fn func(*Element)) {
	if e.Overlay {
		fn(e)
		return // treat the whole subtree as part of this overlay
	}
	for _, c := range e.Children {
		forEachOverlayRoot(c, fn)
	}
}

// forEachOverlayRootFront walks overlay roots last-to-first so the overlay
// painted on top (registered later) is hit-tested first. Paint still uses
// forEachOverlayRoot so earlier overlays stay behind later ones.
func forEachOverlayRootFront(e *Element, fn func(*Element)) {
	if e.Overlay {
		fn(e)
		return
	}
	for i := len(e.Children) - 1; i >= 0; i-- {
		forEachOverlayRootFront(e.Children[i], fn)
	}
}

// Dispatch delivers a mouse event to the tree, front-to-back: overlay subtrees
// receive it before the base tree so a click on an open menu is not also seen
// by the widgets beneath it. Later overlays (painted on top) are hit-tested
// before earlier ones — otherwise a full-window scrim would swallow clicks
// meant for the dialog sitting on it. Handlers may set m.Consumed to halt
// propagation.
func Dispatch(root *Element, m *input.Mouse) {
	if m == nil || m.Consumed {
		return
	}
	stop := false
	forEachOverlayRootFront(root, func(o *Element) {
		if stop {
			return
		}
		dispatchTree(o, m, &stop)
	})
	if !stop {
		dispatchBase(root, m, &stop)
	}
}

func dispatchTree(e *Element, m *input.Mouse, stop *bool) {
	// A clipped viewport's children are invisible outside its frame; skip them
	// so scrolled-out widgets cannot consume events meant for what is actually
	// painted there. The element's own handler still runs (it may track drags).
	if !(e.Clip && !e.Frame.Contains(m.X, m.Y)) {
		// Deepest/last children are visually on top, so visit them first.
		for i := len(e.Children) - 1; i >= 0; i-- {
			dispatchTree(e.Children[i], m, stop)
			if *stop {
				return
			}
		}
	}
	if e.OnMouse != nil {
		e.OnMouse(e, m)
		if m.Consumed {
			*stop = true
		}
	}
}

func dispatchBase(e *Element, m *input.Mouse, stop *bool) {
	if e.Overlay {
		return
	}
	if !(e.Clip && !e.Frame.Contains(m.X, m.Y)) {
		for i := len(e.Children) - 1; i >= 0; i-- {
			dispatchBase(e.Children[i], m, stop)
			if *stop {
				return
			}
		}
	}
	if e.OnMouse != nil {
		e.OnMouse(e, m)
		if m.Consumed {
			*stop = true
		}
	}
}

// ── Layout combinators ────────────────────────────────────────────────────────
// These are thin wrappers around New+Box that encode the most common layout
// patterns. They have no dependency on theme, text, or components — they are
// pure geometry helpers and belong at the layout layer.

// HStack arranges children in a horizontal row, vertically centered, with a
// uniform gap between them.
//
//	layout.HStack(th.Spacing.M, saveBtn.El, cancelBtn.El)
func HStack(gap float32, children ...*Element) *Element {
	return New(Box().Direction(Row).Gap(gap).AlignItems(AlignCenter), children...)
}

// VStack arranges children in a vertical column with a uniform gap.
//
//	layout.VStack(th.Spacing.S, titleEl, bodyEl, footerEl)
func VStack(gap float32, children ...*Element) *Element {
	return New(Box().Direction(Column).Gap(gap), children...)
}

// ZStack layers children on top of each other using the Stack display mode,
// centered both horizontally and vertically by default.
//
//	layout.ZStack(backgroundEl, overlayEl)
func ZStack(children ...*Element) *Element {
	return New(Box().Display(DisplayStack), children...)
}

// Spacer returns a flex-grow element that fills remaining space in a flex
// container — the equivalent of SwiftUI's Spacer().
//
//	layout.HStack(0, labelEl, layout.Spacer(), closeBtn.El)
func Spacer() *Element {
	return New(Box().FlexGrow(1))
}
