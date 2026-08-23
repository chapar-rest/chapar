package layout

import (
	"math"

	"github.com/mirzakhany/yoga/render"
)

// Layout enums are defined locally so callers never depend on an external
// solver. The custom engine in engine.go reads Style directly.

type FlexDirection int

const (
	Row FlexDirection = iota
	RowReverse
	Column
	ColumnReverse
)

type Justify int

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifyBetween
	JustifyAround
	JustifyEvenly
)

type Align int

const (
	AlignAuto Align = iota
	AlignStart
	AlignCenter
	AlignEnd
	AlignStretch
)

type Wrap int

const (
	NoWrap Wrap = iota
	DoWrap
)

type Position int

const (
	PositionRelative Position = iota
	PositionAbsolute
)

type Display int

const (
	DisplayFlex Display = iota
	DisplayGrid
	DisplayStack
)

type TrackKind int

const (
	TrackFixed TrackKind = iota
	TrackFr
	TrackAuto
)

// Px is re-exported from render so layout callers can write layout.Px without
// importing the render package directly.
type Px = render.Px

// Track describes a grid row or column track.
type Track struct {
	Kind  TrackKind
	Value float32
}

func Fixed(v float32) Track { return Track{Kind: TrackFixed, Value: v} }
func Fr(v float32) Track    { return Track{Kind: TrackFr, Value: v} }
func Auto() Track           { return Track{Kind: TrackAuto} }

// nan marks a style dimension as "unset / auto". We use NaN because 0 is a
// meaningful explicit size.
var nan = float32(math.NaN())

func isUnset(v float32) bool { return math.IsNaN(float64(v)) }

// Edges holds a per-side pixel value (used for margin and padding).
type Edges struct {
	Top, Right, Bottom, Left Px
}

// Style is the declarative styling for an Element. Construct it with Box() and
// refine it with the fluent setters below, e.g.
//
//	layout.Box().Direction(layout.Row).FlexGrow(1).PaddingAll(8)
//
// All size fields default to "auto" (NaN); margin/padding default to 0.
type Style struct {
	Disp Display

	Dir     FlexDirection
	Justify Justify
	Align   Align
	SelfAlign Align
	Wrap    Wrap
	Pos     Position

	Grow   float32
	Shrink float32
	Basis  Px

	Width, Height                         Px
	MinWidth, MinHeight, MaxWidth, MaxHeight Px

	Margin  Edges
	Padding Edges

	RowGap, ColGap Px

	// Grid container tracks and auto-row template.
	Cols     []Track
	Rows     []Track
	AutoRows Track

	// Grid item placement (1-based; 0 = auto).
	ColStart, ColSpan int
	RowStart, RowSpan int

	// Stack container alignment (defaults: center/center in Box()).
	StackJustify Justify
	StackAlign   Align

	// Absolute-position offsets, used when Pos == PositionAbsolute. Unset = NaN.
	Left, Top, Right, Bottom float32

	// Visual decoration painted by the layout pass before an element's own Paint
	// hook and children. A zero-alpha color counts as unset.
	BgColor     render.Color
	BorderColor render.Color
	BorderWidth Px
	Radius      Px
}

// Box returns a Style with sensible flexbox defaults: a column container that
// stretches its children, shrink factor 1, and all sizes auto.
func Box() Style {
	return Style{
		Disp:    DisplayFlex,
		Dir:     Column,
		Justify: JustifyStart,
		Align:   AlignStretch,
		SelfAlign: AlignAuto,
		Wrap:    NoWrap,
		Pos:     PositionRelative,
		Grow:    0,
		Shrink:  1,
		Basis:   nan,

		Width: nan, Height: nan,
		MinWidth: nan, MinHeight: nan, MaxWidth: nan, MaxHeight: nan,
		Left: nan, Top: nan, Right: nan, Bottom: nan,

		StackJustify: JustifyCenter,
		StackAlign:   AlignCenter,
		AutoRows:     Fr(1),
	}
}

func (s Style) Display(d Display) Style         { s.Disp = d; return s }
func (s Style) Direction(d FlexDirection) Style { s.Dir = d; return s }
func (s Style) JustifyContent(j Justify) Style  { s.Justify = j; return s }
func (s Style) AlignItems(a Align) Style { s.Align = a; return s }
func (s Style) AlignSelf(a Align) Style  { s.SelfAlign = a; return s }
func (s Style) FlexWrap(w Wrap) Style           { s.Wrap = w; return s }
func (s Style) FlexGrow(v float32) Style  { s.Grow = v; return s }
func (s Style) FlexShrink(v float32) Style { s.Shrink = v; return s }
func (s Style) FlexBasis(v Px) Style       { s.Basis = v; return s }
func (s Style) W(v Px) Style               { s.Width = v; return s }
func (s Style) H(v Px) Style               { s.Height = v; return s }
func (s Style) Size(w, h Px) Style         { s.Width, s.Height = w, h; return s }
func (s Style) Min(w, h Px) Style          { s.MinWidth, s.MinHeight = w, h; return s }
func (s Style) Max(w, h Px) Style          { s.MaxWidth, s.MaxHeight = w, h; return s }

func (s Style) Gap(v Px) Style          { s.RowGap, s.ColGap = v, v; return s }
func (s Style) GapXY(col, row Px) Style { s.ColGap, s.RowGap = col, row; return s }

func (s Style) GridCols(tracks ...Track) Style { s.Cols = append([]Track(nil), tracks...); return s }
func (s Style) GridRows(tracks ...Track) Style { s.Rows = append([]Track(nil), tracks...); return s }
func (s Style) GridAutoRows(t Track) Style     { s.AutoRows = t; return s }
func (s Style) GridArea(colStart, colSpan, rowStart, rowSpan int) Style {
	s.ColStart, s.ColSpan = colStart, colSpan
	s.RowStart, s.RowSpan = rowStart, rowSpan
	return s
}
func (s Style) Col(start, span int) Style { s.ColStart, s.ColSpan = start, span; return s }
func (s Style) Row(start, span int) Style { s.RowStart, s.RowSpan = start, span; return s }

func (s Style) StackPosition(jx Justify, ay Align) Style {
	s.StackJustify, s.StackAlign = jx, ay
	return s
}

// PaddingAll sets a uniform padding on all edges.
func (s Style) PaddingAll(v Px) Style {
	s.Padding = Edges{v, v, v, v}
	return s
}

// PaddingXY sets horizontal and vertical padding.
func (s Style) PaddingXY(x, y Px) Style {
	s.Padding = Edges{Top: y, Right: x, Bottom: y, Left: x}
	return s
}

// MarginAll sets a uniform margin on all edges.
func (s Style) MarginAll(v Px) Style {
	s.Margin = Edges{v, v, v, v}
	return s
}

// Background fills the element's frame with c (rounded by Radius if set). The
// layout pass paints it automatically — no Paint hook needed.
func (s Style) Background(c render.Color) Style { s.BgColor = c; return s }

// Border draws a stroke of the given color and width around the element, with
// corners rounded by radius. Painted automatically by the layout pass. Combine
// with Background to fill inside the stroke.
func (s Style) Border(c render.Color, width, radius Px) Style {
	s.BorderColor, s.BorderWidth, s.Radius = c, width, radius
	return s
}

// CornerRadius rounds the element's Background corners without adding a border.
func (s Style) CornerRadius(r Px) Style { s.Radius = r; return s }

// Absolute positions the element out of normal flow at (left, top).
func (s Style) Absolute(left, top Px) Style {
	s.Pos = PositionAbsolute
	s.Left, s.Top = left, top
	return s
}

// AbsLeft/AbsTop/AbsRight/AbsBottom pin individual edges of an absolutely
// positioned element. Pinning opposite edges (e.g. top and bottom) stretches
// the element between them.
func (s Style) AbsLeft(v Px) Style   { s.Pos = PositionAbsolute; s.Left = v; return s }
func (s Style) AbsTop(v Px) Style    { s.Pos = PositionAbsolute; s.Top = v; return s }
func (s Style) AbsRight(v Px) Style  { s.Pos = PositionAbsolute; s.Right = v; return s }
func (s Style) AbsBottom(v Px) Style { s.Pos = PositionAbsolute; s.Bottom = v; return s }
