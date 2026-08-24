package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// Condition is an interaction pseudo-state used with Spec.When.
type Condition int

const (
	Hovered Condition = iota
	Pressed
	Focused
	Disabled
)

// Cursor is the OS pointer shape a spec may request.
type Cursor = input.Cursor

const (
	CursorDefault  = input.CursorDefault
	CursorPointer  = input.CursorPointer
	CursorResizeEW = input.CursorResizeEW
	CursorResizeNS = input.CursorResizeNS
	CursorText     = input.CursorText
)

// colorRef is a token or literal color. Tokens win over literals when both are
// set on the same spec after a merge (the later patch decides).
type colorRef struct {
	on     bool
	token  Token
	lit    render.Color
	useLit bool
}

func (c colorRef) resolve(th *theme.Theme) (render.Color, bool) {
	if !c.on {
		return render.Color{}, false
	}
	if c.useLit {
		return c.lit, true
	}
	if c.token == TokenUnset {
		return render.Color{}, true
	}
	return c.token.Resolve(th), true
}

type whenRule struct {
	cond  Condition
	patch Spec
}

// Spec is a composable visual/layout style. Construct with Background,
// TextColor, and friends; attach interaction variants with When.
type Spec struct {
	bg, fg, border colorRef
	borderW        float32
	hasBorderW     bool
	radius         float32
	hasRadius      bool
	cursor         Cursor
	hasCursor      bool
	scaleX, scaleY float32
	hasScale       bool

	gap              float32
	hasGap           bool
	pad              layout.Edges
	hasPad           bool
	margin           layout.Edges
	hasMargin        bool
	grow, shrink     float32
	hasGrow          bool
	hasShrink        bool
	width, height    float32
	hasW, hasH       bool
	minW, minH       float32
	hasMinW, hasMinH bool
	fontSize         float32
	hasFontSize      bool
	fontWeight       int
	hasFontWeight    bool
	justify          layout.Justify
	hasJustify       bool
	align            layout.Align
	hasAlign         bool
	wrap             bool
	hasWrap          bool

	when []whenRule
}

// Background starts a spec with a token fill.
func Background(t Token) Spec {
	var s Spec
	s.bg = colorRef{on: true, token: t}
	return s
}

// BackgroundColor starts a spec with a literal fill.
func BackgroundColor(c render.Color) Spec {
	var s Spec
	s.bg = colorRef{on: true, useLit: true, lit: c}
	return s
}

// TextColor sets the inherited foreground used by nested Text.
func (s Spec) TextColor(t Token) Spec {
	s.fg = colorRef{on: true, token: t}
	return s
}

// TextColorLit sets a literal foreground.
func (s Spec) TextColorLit(c render.Color) Spec {
	s.fg = colorRef{on: true, useLit: true, lit: c}
	return s
}

// FontWeight sets the CSS-like font weight (400 Regular, 600 SemiBold).
func (s Spec) FontWeight(w int) Spec {
	s.fontWeight = w
	s.hasFontWeight = true
	return s
}

// Radius sets corner radius in logical pixels.
func (s Spec) Radius(r float32) Spec {
	s.radius = r
	s.hasRadius = true
	return s
}

// Cursor sets the pointer shape while the pointer is inside the node.
func (s Spec) Cursor(c Cursor) Spec {
	s.cursor = c
	s.hasCursor = true
	return s
}

// Scale sets a paint-time scale about the node's center (hit testing is unscaled).
func (s Spec) Scale(x, y float32) Spec {
	s.scaleX, s.scaleY = x, y
	s.hasScale = true
	return s
}

// Border sets a token stroke.
func (s Spec) Border(t Token, width float32) Spec {
	s.border = colorRef{on: true, token: t}
	s.borderW = width
	s.hasBorderW = true
	return s
}

// Padding sets uniform padding.
func (s Spec) Padding(v float32) Spec {
	s.pad = layout.Edges{Top: v, Right: v, Bottom: v, Left: v}
	s.hasPad = true
	return s
}

// PaddingXY sets horizontal and vertical padding.
func (s Spec) PaddingXY(x, y float32) Spec {
	s.pad = layout.Edges{Top: y, Right: x, Bottom: y, Left: x}
	s.hasPad = true
	return s
}

// PaddingLeft sets left padding, preserving other edges already set on s.
func (s Spec) PaddingLeft(v float32) Spec {
	if !s.hasPad {
		s.hasPad = true
	}
	s.pad.Left = v
	return s
}

// PaddingRight sets right padding, preserving other edges already set on s.
func (s Spec) PaddingRight(v float32) Spec {
	if !s.hasPad {
		s.hasPad = true
	}
	s.pad.Right = v
	return s
}

// PaddingTop sets top padding, preserving other edges already set on s.
func (s Spec) PaddingTop(v float32) Spec {
	if !s.hasPad {
		s.hasPad = true
	}
	s.pad.Top = v
	return s
}

// PaddingBottom sets bottom padding, preserving other edges already set on s.
func (s Spec) PaddingBottom(v float32) Spec {
	if !s.hasPad {
		s.hasPad = true
	}
	s.pad.Bottom = v
	return s
}

// Gap sets flex/grid gap. Also available as a Node modifier.
func (s Spec) Gap(v float32) Spec {
	s.gap = v
	s.hasGap = true
	return s
}

// When merges patch on top of s whenever cond is active. Later When calls
// override earlier ones of the same condition. Pressed is applied after
// Hovered; Disabled is applied last.
func (s Spec) When(cond Condition, patch Spec) Spec {
	s.when = append(append([]whenRule(nil), s.when...), whenRule{cond: cond, patch: patch})
	return s
}

func (s Spec) merge(p Spec) Spec {
	if p.bg.on {
		s.bg = p.bg
	}
	if p.fg.on {
		s.fg = p.fg
	}
	if p.border.on {
		s.border = p.border
	}
	if p.hasBorderW {
		s.borderW = p.borderW
		s.hasBorderW = true
	}
	if p.hasRadius {
		s.radius = p.radius
		s.hasRadius = true
	}
	if p.hasCursor {
		s.cursor = p.cursor
		s.hasCursor = true
	}
	if p.hasScale {
		s.scaleX, s.scaleY = p.scaleX, p.scaleY
		s.hasScale = true
	}
	if p.hasGap {
		s.gap = p.gap
		s.hasGap = true
	}
	if p.hasPad {
		s.pad = p.pad
		s.hasPad = true
	}
	if p.hasMargin {
		s.margin = p.margin
		s.hasMargin = true
	}
	if p.hasGrow {
		s.grow = p.grow
		s.hasGrow = true
	}
	if p.hasShrink {
		s.shrink = p.shrink
		s.hasShrink = true
	}
	if p.hasW {
		s.width = p.width
		s.hasW = true
	}
	if p.hasH {
		s.height = p.height
		s.hasH = true
	}
	if p.hasMinW {
		s.minW = p.minW
		s.hasMinW = true
	}
	if p.hasMinH {
		s.minH = p.minH
		s.hasMinH = true
	}
	if p.hasFontSize {
		s.fontSize = p.fontSize
		s.hasFontSize = true
	}
	if p.hasFontWeight {
		s.fontWeight = p.fontWeight
		s.hasFontWeight = true
	}
	if p.hasJustify {
		s.justify = p.justify
		s.hasJustify = true
	}
	if p.hasAlign {
		s.align = p.align
		s.hasAlign = true
	}
	if len(p.when) > 0 {
		s.when = append(append([]whenRule(nil), s.when...), p.when...)
	}
	return s
}

type interactState struct {
	hovered, pressed, focused, disabled bool
}

type resolvedSpec struct {
	bg, fg, border  render.Color
	hasBg, hasFg    bool
	hasBorder       bool
	borderW, radius float32
	hasRadius       bool
	cursor          Cursor
	hasCursor       bool
	scaleX, scaleY  float32
	fontSize        float32
	hasFontSize     bool
	fontWeight      int
	hasFontWeight   bool
}

func (s Spec) resolve(th *theme.Theme, st interactState) resolvedSpec {
	out := s
	var hover, press, focus, dis Spec
	var hasH, hasP, hasF, hasD bool
	for _, w := range s.when {
		switch w.cond {
		case Hovered:
			hover, hasH = w.patch, true
		case Pressed:
			press, hasP = w.patch, true
		case Focused:
			focus, hasF = w.patch, true
		case Disabled:
			dis, hasD = w.patch, true
		}
	}
	if st.hovered && hasH && !st.disabled {
		out = out.merge(hover)
	}
	if st.pressed && hasP && !st.disabled {
		out = out.merge(press)
	}
	if st.focused && hasF {
		out = out.merge(focus)
	}
	if st.disabled && hasD {
		out = out.merge(dis)
	}

	var r resolvedSpec
	r.bg, r.hasBg = out.bg.resolve(th)
	r.fg, r.hasFg = out.fg.resolve(th)
	r.border, r.hasBorder = out.border.resolve(th)
	if out.hasBorderW {
		r.borderW = out.borderW
	}
	if out.hasRadius {
		r.radius = out.radius
		r.hasRadius = true
	}
	if out.hasCursor {
		r.cursor = out.cursor
		r.hasCursor = true
	}
	r.scaleX, r.scaleY = 1, 1
	if out.hasScale {
		r.scaleX, r.scaleY = out.scaleX, out.scaleY
	}
	if out.hasFontSize {
		r.fontSize = out.fontSize
		r.hasFontSize = true
	}
	if out.hasFontWeight {
		r.fontWeight = out.fontWeight
		r.hasFontWeight = true
	}
	return r
}

func applyLayoutSpec(st layout.Style, s Spec) layout.Style {
	if s.hasGap {
		st = st.Gap(s.gap)
	}
	if s.hasGrow {
		st = st.FlexGrow(s.grow)
	}
	if s.hasShrink {
		st = st.FlexShrink(s.shrink)
	}
	if s.hasW {
		st = st.W(s.width)
	}
	if s.hasH {
		st = st.H(s.height)
	}
	if s.hasMinW || s.hasMinH {
		mw, mh := st.MinWidth, st.MinHeight
		if s.hasMinW {
			mw = s.minW
		}
		if s.hasMinH {
			mh = s.minH
		}
		st = st.Min(mw, mh)
	}
	if s.hasPad {
		st.Padding = s.pad
	}
	if s.hasMargin {
		st.Margin = s.margin
	}
	if s.hasJustify {
		st = st.JustifyContent(s.justify)
	}
	if s.hasAlign {
		st = st.AlignItems(s.align)
	}
	if s.hasWrap && s.wrap {
		st = st.FlexWrap(layout.DoWrap)
	}
	return st
}

func applyVisualSpec(el *layout.Element, s Spec, th *theme.Theme, st interactState) {
	r := s.resolve(th, st)
	if r.hasBg {
		el.Style.BgColor = r.bg
	}
	if r.hasRadius {
		el.Style.Radius = r.radius
	}
	if r.hasBorder {
		el.Style.BorderColor = r.border
		el.Style.BorderWidth = r.borderW
		if r.hasRadius {
			el.Style.Radius = r.radius
		}
	}
}

func scaledFrame(f render.Rect, sx, sy float32) render.Rect {
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	if sx == 1 && sy == 1 {
		return f
	}
	dw := f.W * (1 - sx) / 2
	dh := f.H * (1 - sy) / 2
	return render.Rect{X: f.X + dw, Y: f.Y + dh, W: f.W * sx, H: f.H * sy}
}
