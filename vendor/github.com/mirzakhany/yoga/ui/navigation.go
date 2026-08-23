package ui

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// NavOrientation is the axis along which navigation items are arranged.
type NavOrientation int

const (
	NavVertical NavOrientation = iota
	NavHorizontal
)

// NavItemLayout controls how icon and label are arranged inside each item.
type NavItemLayout int

const (
	NavIconLeft NavItemLayout = iota
	NavIconRight
	NavIconTop
	NavIconBottom
)

// NavItem is one navigation entry.
type NavItem struct {
	ID    string
	Label string
	Icon  icons.Icon
}

type navData struct {
	orient NavOrientation
	layout NavItemLayout
	items  []NavItem
	bg     *render.Color
}

type navState struct {
	hover int
}

// Nav is a selectable list of icon+text items. Selection is controlled via .Selected(i).
func Nav(id string, orient NavOrientation, itemLayout NavItemLayout, items ...NavItem) *Node {
	return &Node{kind: kindNav, id: id, extra: &navData{orient: orient, layout: itemLayout, items: items}}
}

// NavBackground overrides the strip fill.
func (n *Node) NavBackground(c *render.Color) *Node {
	if d, ok := n.extra.(*navData); ok {
		d.bg = c
	}
	return n
}

func (n *Node) layoutNav(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "nav")
	}
	st := c.Widget(id, func() any { return &navState{hover: -1} }).(*navState)
	d, _ := n.extra.(*navData)
	if d == nil {
		d = &navData{}
	}
	th := c.Theme()
	pad := th.Spacing.S
	gap := th.Spacing.XS
	box := layout.Box().Gap(gap).PaddingAll(pad).FlexShrink(0)
	if d.orient == NavVertical {
		box = box.Direction(layout.Column).AlignItems(layout.AlignStretch)
	} else {
		box = box.Direction(layout.Row).AlignItems(layout.AlignCenter)
	}
	h := navItemHeight(d.layout)
	selected := n.selected
	onSelect := n.onSelectIdx
	items := d.items
	var children []*layout.Element
	for i, item := range items {
		idx, it := i, item
		var style layout.Style
		if d.orient == NavVertical {
			style = layout.Box().H(h).FlexShrink(0)
		} else {
			style = layout.Box().W(navItemWidth(d.layout, it)).H(h).FlexShrink(0)
		}
		el := layout.New(style)
		el.Paint = func(dl *render.DrawList, text *shape.Engine) {
			paintNavItem(dl, text, el.Frame, d.layout, it, idx == selected, idx == st.hover)
		}
		el.OnMouse = func(e *layout.Element, m *input.Mouse) {
			if !e.Frame.Contains(m.X, m.Y) {
				return
			}
			st.hover = idx
			m.SetCursor(CursorPointer)
			if m.Released && onSelect != nil {
				onSelect(idx, it.ID)
				m.Consumed = true
			}
		}
		children = append(children, el)
	}
	el := layout.New(applyLayoutSpec(box, n.spec), children...)
	el.Clip = true
	bg := th.Chrome
	if d.bg != nil {
		bg = *d.bg
	}
	el.Paint = func(dl *render.DrawList, _ *shape.Engine) {
		dl.AddRect(el.Frame, bg)
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if !e.Frame.Contains(m.X, m.Y) {
			st.hover = -1
		}
	}
	return el
}

func navLabelStyle(itemLayout NavItemLayout) theme.TypographyStyle {
	th := theme.Current()
	switch itemLayout {
	case NavIconTop, NavIconBottom:
		return th.Typography.Caption
	default:
		return th.Typography.Body
	}
}

func navItemHeight(itemLayout NavItemLayout) float32 {
	th := theme.Current()
	pad := th.Spacing.M
	iconSz := th.Metrics.IconSizeSM
	eng := frameText()
	var lh float32
	if eng != nil {
		_, lh = eng.MeasureAt("Ag", navLabelStyle(itemLayout).Size)
	} else {
		lh = th.Typography.Caption.LineHeight
	}
	switch itemLayout {
	case NavIconTop, NavIconBottom:
		return pad*2 + iconSz + th.Spacing.S + lh
	default:
		return th.Metrics.ControlHeight
	}
}

func navItemWidth(itemLayout NavItemLayout, item NavItem) float32 {
	th := theme.Current()
	padX := th.Spacing.M
	iconSz := th.Metrics.IconSizeSM
	style := navLabelStyle(itemLayout)
	var tw float32
	if eng := frameText(); eng != nil {
		tw, _ = eng.MeasureAt(item.Label, style.Size)
	}
	gap := th.Spacing.S
	switch itemLayout {
	case NavIconTop, NavIconBottom:
		contentW := tw
		if !item.Icon.Empty() && iconSz > contentW {
			contentW = iconSz
		}
		return contentW + 2*padX
	default:
		w := tw + 2*padX
		if !item.Icon.Empty() {
			w += gap + iconSz
		}
		return w
	}
}

type navItemGeom struct {
	icon    render.Rect
	hasIcon bool
	textX   float32
	textY   float32
}

func navItemGeomOf(f render.Rect, itemLayout NavItemLayout, item NavItem) navItemGeom {
	th := theme.Current()
	padX := th.Spacing.M
	iconSz := th.Metrics.IconSizeSM
	style := navLabelStyle(itemLayout)
	var tw, lh float32
	if eng := frameText(); eng != nil {
		tw, lh = eng.MeasureAt(item.Label, style.Size)
	}
	gap := th.Spacing.S
	if itemLayout == NavIconTop || itemLayout == NavIconBottom {
		gap = th.Spacing.XS
	}
	hasIcon := !item.Icon.Empty()
	var g navItemGeom
	g.hasIcon = hasIcon
	switch itemLayout {
	case NavIconRight:
		g.textX, g.textY = f.X+padX, f.Y+(f.H-lh)/2
		if hasIcon {
			g.icon = render.Rect{X: g.textX + tw + gap, Y: f.Y + (f.H-iconSz)/2, W: iconSz, H: iconSz}
		}
	case NavIconTop:
		contentH := lh
		if hasIcon {
			contentH = iconSz + gap + lh
		}
		startY := f.Y + (f.H-contentH)/2
		if hasIcon {
			g.icon = render.Rect{X: f.X + (f.W-iconSz)/2, Y: startY, W: iconSz, H: iconSz}
			g.textX = f.X + (f.W-tw)/2
			g.textY = startY + iconSz + gap
		} else {
			g.textX = f.X + (f.W-tw)/2
			g.textY = startY
		}
	case NavIconBottom:
		contentH := lh
		if hasIcon {
			contentH = lh + gap + iconSz
		}
		startY := f.Y + (f.H-contentH)/2
		g.textX = f.X + (f.W-tw)/2
		g.textY = startY
		if hasIcon {
			g.icon = render.Rect{X: f.X + (f.W-iconSz)/2, Y: startY + lh + gap, W: iconSz, H: iconSz}
		}
	default:
		x := f.X + padX
		if hasIcon {
			g.icon = render.Rect{X: x, Y: f.Y + (f.H-iconSz)/2, W: iconSz, H: iconSz}
			x += iconSz + gap
		}
		g.textX, g.textY = x, f.Y+(f.H-lh)/2
	}
	return g
}

func paintNavItem(dl *render.DrawList, text *shape.Engine, f render.Rect, itemLayout NavItemLayout, item NavItem, selected, hovered bool) {
	th := theme.Current()
	r := th.Radius.Large
	switch {
	case selected:
		dl.AddRoundedRect(f, r, th.ListActive)
	case hovered:
		dl.AddRoundedRect(f, r, th.ListHover)
	}
	geom := navItemGeomOf(f, itemLayout, item)
	style := navLabelStyle(itemLayout)
	iconCol := th.ForegroundMuted
	if selected || hovered {
		iconCol = th.Foreground
	}
	if geom.hasIcon && frameIcons() != nil {
		frameIcons().Draw(dl, item.Icon, geom.icon, iconCol)
	}
	if item.Label != "" {
		text.DrawStringTopAt(dl, item.Label, geom.textX, geom.textY, th.Foreground, style.Size)
	}
}
