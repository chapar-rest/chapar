package ui

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
)

type disclosureData struct {
	title string
	body  View
	open  bool
}

type disclosureState struct {
	hovered, focused bool
	el               *layout.Element
	toggle           func()
}

func (s *disclosureState) Focus()                      { s.focused = true }
func (s *disclosureState) Blur()                       { s.focused = false }
func (s *disclosureState) Focused() bool               { return s.focused }
func (s *disclosureState) HandleKeys([]input.KeyEvent) {}
func (s *disclosureState) CapturesTab() bool           { return false }
func (s *disclosureState) FocusOnClick() bool          { return true }
func (s *disclosureState) FocusEl() *layout.Element    { return s.el }

func (s *disclosureState) HandleText(runes []rune) {
	if !s.focused {
		return
	}
	for _, r := range runes {
		if (r == ' ' || r == '\n') && s.toggle != nil {
			s.toggle()
		}
	}
}

var _ Focusable = (*disclosureState)(nil)

// Disclosure is a single collapsible section (title + body).
func Disclosure(id, title string, body View) *Node {
	return &Node{kind: kindDisclosure, id: id, extra: &disclosureData{title: title, body: body}}
}

func (n *Node) layoutDisclosure(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "disclosure")
	}
	d, _ := n.extra.(*disclosureData)
	if d == nil {
		d = &disclosureData{}
	}
	open := d.open
	st := c.Widget(id, func() any { return &disclosureState{} }).(*disclosureState)
	if c.Focus() != nil {
		c.Focus().Add(st)
	}

	th := c.Theme()
	headerH := th.Metrics.ControlHeight
	iconSz := th.Metrics.IconSizeSM
	title := d.title
	onToggle := n.onToggle

	header := layout.New(layout.Box().H(headerH).FlexShrink(0))
	st.el = header
	st.toggle = func() {
		if onToggle != nil {
			onToggle(!open)
		}
	}
	header.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := header.Frame
		if st.hovered {
			dl.AddRoundedRect(f, th.Radius.Small, th.ListHover)
		}
		if st.focused {
			paintFocusRing(dl, f, th.Surface, th)
		}
		chev := icons.ChevronDown
		if open {
			chev = icons.ChevronUp
		}
		pad := th.Spacing.S
		cy := f.Y + f.H/2
		if sheet := frameIcons(); sheet != nil {
			sheet.Draw(dl, chev, render.Rect{X: f.X + pad, Y: cy - iconSz/2, W: iconSz, H: iconSz}, th.ForegroundMuted)
		}
		style := th.Typography.BodyStrong
		_, lh := text.MeasureAt(title, style.Size)
		text.DrawStringTopAt(dl, title, f.X+pad+iconSz+th.Spacing.S, cy-lh/2, th.Foreground, style.Size)
	}
	header.OnMouse = func(e *layout.Element, m *input.Mouse) {
		inside := e.Frame.Contains(m.X, m.Y)
		trackHover(c, &st.hovered, inside)
		if inside {
			m.SetCursor(CursorPointer)
		}
		if inside && m.Released {
			if onToggle != nil {
				onToggle(!open)
			}
			c.MarkNeedsPaint()
			m.Consumed = true
		}
		if inside && m.Pressed {
			m.Consumed = true
		}
	}

	kids := []*layout.Element{header}
	if open && d.body != nil {
		bodyEl := d.body.Layout(c)
		if bodyEl != nil {
			padded := layout.New(layout.Box().PaddingXY(th.Spacing.L, th.Spacing.S), bodyEl)
			kids = append(kids, padded)
		}
	}
	return layout.New(applyLayoutSpec(layout.Box().Direction(layout.Column), n.spec), kids...)
}

// AccordionItem is one section in an Accordion.
type AccordionItem struct {
	ID, Title string
	Body      View
}

type accordionData struct {
	items     []AccordionItem
	openIDs   map[string]bool
	exclusive bool
	onToggle  func(id string, open bool)
}

// Accordion is a column of Disclosure sections.
func Accordion(id string, items ...AccordionItem) *Node {
	return &Node{kind: kindAccordion, id: id, extra: &accordionData{items: items, openIDs: map[string]bool{}}}
}

// OpenIDs marks which Accordion sections are expanded.
func (n *Node) OpenIDs(ids ...string) *Node {
	d, ok := n.extra.(*accordionData)
	if !ok {
		return n
	}
	d.openIDs = make(map[string]bool, len(ids))
	for _, id := range ids {
		d.openIDs[id] = true
	}
	return n
}

// Exclusive keeps at most one Accordion section open when the app honors
// OnAccordionToggle by storing a single open ID.
func (n *Node) Exclusive() *Node {
	if d, ok := n.extra.(*accordionData); ok {
		d.exclusive = true
	}
	return n
}

// OnAccordionToggle is called when a section open state changes.
func (n *Node) OnAccordionToggle(fn func(id string, open bool)) *Node {
	if d, ok := n.extra.(*accordionData); ok {
		d.onToggle = fn
	}
	return n
}

func (n *Node) layoutAccordion(c *Ctx) *layout.Element {
	d, _ := n.extra.(*accordionData)
	if d == nil {
		d = &accordionData{}
	}
	th := c.Theme()
	baseID := n.id
	if baseID == "" {
		baseID = autoID(c, "accordion")
	}
	rows := make([]View, 0, len(d.items))
	openIDs := d.openIDs
	exclusive := d.exclusive
	onToggle := d.onToggle
	for _, item := range d.items {
		item := item
		open := openIDs[item.ID]
		disc := Disclosure(baseID+"-"+item.ID, item.Title, item.Body).
			Open(open).
			OnToggle(func(v bool) {
				if onToggle == nil {
					return
				}
				if exclusive && v {
					onToggle(item.ID, true)
					return
				}
				onToggle(item.ID, v)
			})
		rows = append(rows, disc)
	}
	return Column(rows...).Gap(th.Spacing.XS).Style(n.spec).Layout(c)
}
