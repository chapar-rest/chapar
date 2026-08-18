package ui

import (
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type nodeKind int

const (
	kindColumn nodeKind = iota
	kindRow
	kindStack
	kindCenter
	kindSpacer
	kindGrid
	kindText
	kindButton
	kindTextField
	kindCheckbox
	kindIconButton
	kindWrap
	kindRaw
	kindHLine
	kindVLine
	kindIcon
	kindAlert
	kindCard
	kindSpinner
	kindRadio
	kindSegmented
	kindSelect
	kindDropdown
	kindNav
	kindTabs
	kindBreadcrumb
	kindTagEdit
	kindScroll
)

const (
	variantSecondary = iota
	variantPrimary
	variantSubtle
)

// Node is the universal view value. Layout directives and widgets return *Node
// so modifiers chain. Node implements View.
type Node struct {
	kind         nodeKind
	id           string
	children     []View
	child        View
	inner        View
	raw          *layout.Element
	text         string
	spec         Spec
	onClick      func()
	onChange     func(string)
	onSubmit     func(string)
	onToggle     func(bool)
	checked      bool
	disabled     bool
	variant      int
	cols         int
	icon         string
	hint         string
	placeholder  string
	iconStart    string
	iconEnd      string
	password     bool
	lineThick    float32
	lineColor    render.Color
	iconSize     float32
	iconColor    render.Color
	labelMuted   bool
	labelStrike  bool
	defaultFocus bool
	extra        any
	selected     int
	onSelectIdx  func(int, string)
	onCloseIdx   func(int)
}

var _ View = (*Node)(nil)

// Column arranges children vertically.
func Column(children ...View) *Node {
	return &Node{kind: kindColumn, children: children}
}

// Row arranges children horizontally, vertically centered.
func Row(children ...View) *Node {
	return &Node{kind: kindRow, children: children}
}

// Stack layers children on top of one another (z-order).
func Stack(children ...View) *Node {
	return &Node{kind: kindStack, children: children}
}

// Center expands to fill its parent and centers child on both axes.
func Center(child View) *Node {
	return &Node{kind: kindCenter, child: child}
}

// Spacer is a flex-grow filler.
func Spacer() *Node { return &Node{kind: kindSpacer} }

// Scroll clips child to the available size and scrolls when content overflows.
// id keys scroll offset across frames.
func Scroll(id string, child View) *Node {
	return &Node{kind: kindScroll, id: id, child: child}
}

// Grid arranges children in cols equal-width columns.
func Grid(cols int, children ...View) *Node {
	if cols < 1 {
		cols = 1
	}
	return &Node{kind: kindGrid, cols: cols, children: children}
}

// ViewOf wraps any View so Node modifiers can chain (Grow, Padding, …).
func ViewOf(v View) *Node {
	if v == nil {
		return &Node{kind: kindWrap}
	}
	if n, ok := v.(*Node); ok {
		return n
	}
	return &Node{kind: kindWrap, inner: v}
}

// Raw adapts a bare *layout.Element to the View tree.
func Raw(e *layout.Element) *Node {
	return &Node{kind: kindRaw, raw: e}
}

// HLine is a horizontal rule of the given thickness.
func HLine(thickness float32, color render.Color) *Node {
	return &Node{kind: kindHLine, lineThick: thickness, lineColor: color}
}

// VLine is a vertical rule of the given thickness.
func VLine(thickness float32, color render.Color) *Node {
	return &Node{kind: kindVLine, lineThick: thickness, lineColor: color}
}

// Icon draws a named sprite at size, tinted by color.
func Icon(name string, size float32, color render.Color) *Node {
	return &Node{kind: kindIcon, icon: name, iconSize: size, iconColor: color}
}

// Gap sets the gap between children.
func (n *Node) Gap(g float32) *Node {
	n.spec.gap = g
	n.spec.hasGap = true
	return n
}

// Padding sets uniform padding.
func (n *Node) Padding(v float32) *Node {
	n.spec.pad = layout.Edges{Top: v, Right: v, Bottom: v, Left: v}
	n.spec.hasPad = true
	return n
}

// PaddingXY sets horizontal and vertical padding.
func (n *Node) PaddingXY(x, y float32) *Node {
	n.spec.pad = layout.Edges{Top: y, Right: x, Bottom: y, Left: x}
	n.spec.hasPad = true
	return n
}

// PaddingLeft sets left padding.
func (n *Node) PaddingLeft(v float32) *Node { n.ensurePad(); n.spec.pad.Left = v; return n }

// PaddingRight sets right padding.
func (n *Node) PaddingRight(v float32) *Node { n.ensurePad(); n.spec.pad.Right = v; return n }

// PaddingTop sets top padding.
func (n *Node) PaddingTop(v float32) *Node { n.ensurePad(); n.spec.pad.Top = v; return n }

// PaddingBottom sets bottom padding.
func (n *Node) PaddingBottom(v float32) *Node { n.ensurePad(); n.spec.pad.Bottom = v; return n }

func (n *Node) ensurePad() {
	if !n.spec.hasPad {
		n.spec.hasPad = true
	}
}

// Margin sets uniform margin.
func (n *Node) Margin(v float32) *Node {
	n.spec.margin = layout.Edges{Top: v, Right: v, Bottom: v, Left: v}
	n.spec.hasMargin = true
	return n
}

// MarginXY sets horizontal and vertical margin.
func (n *Node) MarginXY(x, y float32) *Node {
	n.spec.margin = layout.Edges{Top: y, Right: x, Bottom: y, Left: x}
	n.spec.hasMargin = true
	return n
}

// MarginLeft sets left margin.
func (n *Node) MarginLeft(v float32) *Node { n.ensureMargin(); n.spec.margin.Left = v; return n }

// MarginRight sets right margin.
func (n *Node) MarginRight(v float32) *Node { n.ensureMargin(); n.spec.margin.Right = v; return n }

// MarginTop sets top margin.
func (n *Node) MarginTop(v float32) *Node { n.ensureMargin(); n.spec.margin.Top = v; return n }

// MarginBottom sets bottom margin.
func (n *Node) MarginBottom(v float32) *Node { n.ensureMargin(); n.spec.margin.Bottom = v; return n }

func (n *Node) ensureMargin() {
	if !n.spec.hasMargin {
		n.spec.hasMargin = true
	}
}

// Wrap enables flex wrapping for Row/Column.
func (n *Node) Wrap() *Node {
	n.spec.wrap = true
	n.spec.hasWrap = true
	return n
}
func (n *Node) Grow(v float32) *Node {
	n.spec.grow = v
	n.spec.hasGrow = true
	return n
}

// Shrink sets flex shrink.
func (n *Node) Shrink(v float32) *Node {
	n.spec.shrink = v
	n.spec.hasShrink = true
	return n
}

// Width sets a fixed width.
func (n *Node) Width(w float32) *Node {
	n.spec.width = w
	n.spec.hasW = true
	return n
}

// Height sets a fixed height.
func (n *Node) Height(h float32) *Node {
	n.spec.height = h
	n.spec.hasH = true
	if !n.spec.hasMinH {
		n.spec.minH = h
		n.spec.hasMinH = true
	}
	return n
}

// Frame sets a fixed width and height.
func (n *Node) Frame(w, h float32) *Node {
	n.spec.width, n.spec.height = w, h
	n.spec.hasW, n.spec.hasH = true, true
	return n
}

// Size on Text is font size; on other nodes it sets width and height.
func (n *Node) Size(v float32) *Node {
	if n.kind == kindText {
		n.spec.fontSize = v
		n.spec.hasFontSize = true
		return n
	}
	return n.Frame(v, v)
}

// Align sets cross-axis alignment of children.
func (n *Node) Align(a layout.Align) *Node {
	n.spec.align = a
	n.spec.hasAlign = true
	return n
}

// Justify sets main-axis distribution of children.
func (n *Node) Justify(j layout.Justify) *Node {
	n.spec.justify = j
	n.spec.hasJustify = true
	return n
}

// Style merges a visual spec onto this node.
func (n *Node) Style(s Spec) *Node {
	n.spec = n.spec.merge(s)
	return n
}

// Background sets a token fill.
func (n *Node) Background(t Token) *Node {
	n.spec = n.spec.merge(Background(t))
	return n
}

// BackgroundColor sets a literal fill.
func (n *Node) BackgroundColor(c render.Color) *Node {
	n.spec = n.spec.merge(BackgroundColor(c))
	return n
}

// DefaultFocus asks the focus scope to focus this control when nothing else is focused.
func (n *Node) DefaultFocus() *Node {
	n.defaultFocus = true
	return n
}

// OnClick sets a pointer-up handler.
func (n *Node) OnClick(fn func()) *Node {
	n.onClick = fn
	return n
}

// OnChange sets a text-field change handler.
func (n *Node) OnChange(fn func(string)) *Node {
	n.onChange = fn
	return n
}

// OnSubmit sets a text-field Enter handler.
func (n *Node) OnSubmit(fn func(string)) *Node {
	n.onSubmit = fn
	return n
}

// Disabled marks a control non-interactive.
func (n *Node) Disabled(v bool) *Node {
	n.disabled = v
	return n
}

// Primary applies the theme's primary button spec.
func (n *Node) Primary() *Node { n.variant = variantPrimary; return n }

// Secondary applies the theme's secondary button spec.
func (n *Node) Secondary() *Node { n.variant = variantSecondary; return n }

// Subtle applies the theme's subtle button spec.
func (n *Node) Subtle() *Node { n.variant = variantSubtle; return n }

// IconStart sets a leading icon name (Button / TextField).
func (n *Node) IconStart(name string) *Node { n.iconStart = name; return n }

// IconEnd sets a trailing icon name (TextField).
func (n *Node) IconEnd(name string) *Node { n.iconEnd = name; return n }

// Hint sets a keyboard-hint chip on a Button.
func (n *Node) Hint(s string) *Node { n.hint = s; return n }

// Placeholder sets TextField placeholder text.
func (n *Node) Placeholder(s string) *Node { n.placeholder = s; return n }

// Password masks TextField contents.
func (n *Node) Password(v bool) *Node { n.password = v; return n }

// Check sets Checkbox checked state (controlled).
func (n *Node) Check(v bool) *Node { n.checked = v; return n }

// OnToggle sets a Checkbox change handler.
func (n *Node) OnToggle(fn func(bool)) *Node { n.onToggle = fn; return n }

// LabelMuted draws a checkbox label in the muted foreground.
func (n *Node) LabelMuted(v bool) *Node { n.labelMuted = v; return n }

// LabelStrike draws a strikethrough on a checkbox label.
func (n *Node) LabelStrike(v bool) *Node { n.labelStrike = v; return n }

// Selected sets the active index (Select, Segmented, Nav, Tabs).
func (n *Node) Selected(i int) *Node { n.selected = i; return n }

// OnSelectItem is called with (index, id/value) for Nav, Segmented, Tabs, Select.
func (n *Node) OnSelectItem(fn func(int, string)) *Node { n.onSelectIdx = fn; return n }

// OnTabClose is called when a tab close box is clicked.
func (n *Node) OnTabClose(fn func(int)) *Node { n.onCloseIdx = fn; return n }

func layoutViews(c *Ctx, views []View) []*layout.Element {
	out := make([]*layout.Element, 0, len(views))
	for _, v := range views {
		if v == nil {
			continue
		}
		if el := v.Layout(c); el != nil {
			out = append(out, el)
		}
	}
	return out
}

// Layout materializes this node into a layout.Element.
func (n *Node) Layout(c *Ctx) *layout.Element {
	if n == nil {
		return layout.New(layout.Box())
	}
	th := c.Theme()
	switch n.kind {
	case kindColumn:
		st := applyLayoutSpec(layout.Box().Direction(layout.Column), n.spec)
		el := layout.New(st, layoutViews(c, n.children)...)
		applyVisualSpec(el, n.spec, th, interactState{})
		return el
	case kindRow:
		st := applyLayoutSpec(layout.Box().Direction(layout.Row).AlignItems(layout.AlignCenter), n.spec)
		el := layout.New(st, layoutViews(c, n.children)...)
		applyVisualSpec(el, n.spec, th, interactState{})
		return el
	case kindStack:
		st := applyLayoutSpec(layout.Box().Display(layout.DisplayStack), n.spec)
		el := layout.New(st, layoutViews(c, n.children)...)
		applyVisualSpec(el, n.spec, th, interactState{})
		return el
	case kindCenter:
		st := applyLayoutSpec(layout.Box().Direction(layout.Column).
			JustifyContent(layout.JustifyCenter).AlignItems(layout.AlignCenter).FlexGrow(1), n.spec)
		var kids []*layout.Element
		if n.child != nil {
			kids = []*layout.Element{n.child.Layout(c)}
		}
		el := layout.New(st, kids...)
		applyVisualSpec(el, n.spec, th, interactState{})
		return el
	case kindSpacer:
		st := applyLayoutSpec(layout.Box().FlexGrow(1), n.spec)
		return layout.New(st)
	case kindGrid:
		tracks := make([]layout.Track, n.cols)
		for i := range tracks {
			tracks[i] = layout.Fr(1)
		}
		st := applyLayoutSpec(layout.Box().Display(layout.DisplayGrid).GridCols(tracks...), n.spec)
		el := layout.New(st, layoutViews(c, n.children)...)
		applyVisualSpec(el, n.spec, th, interactState{})
		return el
	case kindText:
		return n.layoutText(c)
	case kindButton:
		return n.layoutButton(c)
	case kindTextField:
		return n.layoutTextField(c)
	case kindCheckbox:
		return n.layoutCheckbox(c)
	case kindIconButton:
		return n.layoutIconButton(c)
	case kindAlert:
		return n.layoutAlert(c)
	case kindCard:
		return n.layoutCard(c)
	case kindSpinner:
		return n.layoutSpinner(c)
	case kindRadio:
		return n.layoutRadio(c)
	case kindSegmented:
		return n.layoutSegmented(c)
	case kindSelect:
		return n.layoutSelect(c)
	case kindDropdown:
		return n.layoutDropdown(c)
	case kindNav:
		return n.layoutNav(c)
	case kindTabs:
		return n.layoutTabs(c)
	case kindBreadcrumb:
		return n.layoutBreadcrumb(c)
	case kindTagEdit:
		return n.layoutTagEdit(c)
	case kindScroll:
		return n.layoutScroll(c)
	case kindWrap:
		if n.inner == nil {
			return layout.New(layout.Box())
		}
		el := n.inner.Layout(c)
		if el == nil {
			return layout.New(layout.Box())
		}
		el.Style = applyLayoutSpec(el.Style, n.spec)
		applyVisualSpec(el, n.spec, th, interactState{})
		return el
	case kindRaw:
		if n.raw == nil {
			return layout.New(layout.Box())
		}
		n.raw.Style = applyLayoutSpec(n.raw.Style, n.spec)
		return n.raw
	case kindHLine:
		st := applyLayoutSpec(layout.Box().H(n.lineThick), n.spec)
		return layout.New(st).Bg(n.lineColor)
	case kindVLine:
		st := applyLayoutSpec(layout.Box().W(n.lineThick), n.spec)
		return layout.New(st).Bg(n.lineColor)
	case kindIcon:
		sz := n.iconSize
		if sz <= 0 {
			sz = th.Metrics.IconSizeSM
		}
		st := applyLayoutSpec(layout.Box().Size(sz, sz).FlexShrink(0), n.spec)
		el := layout.New(st)
		name, col := n.icon, n.iconColor
		el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
			if sheet := frameIcons(); sheet != nil {
				sheet.Draw(dl, name, el.Frame, col)
			}
		}
		return el
	default:
		return layout.New(layout.Box())
	}
}
