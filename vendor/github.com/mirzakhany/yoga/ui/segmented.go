package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// SegmentItem is one choice in a Segmented control.
type SegmentItem struct {
	Icon  string
	Label string
	Value string
}

type segmentedData struct {
	items []SegmentItem
}

type segmentedState struct {
	hover int
}

const (
	segPad          = 3
	segCellPadX     = 8
	segCellGap      = 2
	segIconLabelGap = 6
)

// Segmented is a single-select switch. Selection is controlled via .Selected(i).
func Segmented(id string, items ...SegmentItem) *Node {
	return &Node{kind: kindSegmented, id: id, extra: &segmentedData{items: items}}
}

func (n *Node) layoutSegmented(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "segmented")
	}
	st := c.Widget(id, func() any { return &segmentedState{hover: -1} }).(*segmentedState)
	d, _ := n.extra.(*segmentedData)
	items := []SegmentItem{}
	if d != nil {
		items = d.items
	}
	th := c.Theme()
	cellW := computeSegCellW(c, items)
	nn := float32(len(items))
	w := nn*cellW + f32max(0, nn-1)*segCellGap + 2*segPad
	el := layout.New(applyLayoutSpec(layout.Box().W(w).H(th.Metrics.ControlHeight).FlexShrink(0), n.spec))
	selected := n.selected
	onChange := n.onChange
	onSelectIdx := n.onSelectIdx
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		paintSegmented(dl, text, el.Frame, items, selected, st.hover, cellW)
	}
	el.OnMouse = func(_ *layout.Element, m *input.Mouse) {
		st.hover = -1
		for i := range items {
			cell := segCellRect(el.Frame, i, cellW)
			if !cell.Contains(m.X, m.Y) {
				continue
			}
			st.hover = i
			m.SetCursor(CursorPointer)
			if m.Pressed {
				m.Consumed = true
			}
			if m.Released && i != selected {
				if onSelectIdx != nil {
					onSelectIdx(i, items[i].Value)
				}
				if onChange != nil {
					onChange(items[i].Value)
				}
			}
		}
	}
	return el
}

func computeSegCellW(c *Ctx, items []SegmentItem) float32 {
	th := c.Theme()
	text := c.Text()
	iconSz := th.Metrics.IconSizeSM
	var maxContent float32
	for _, it := range items {
		var cw float32
		if it.Icon != "" {
			cw += iconSz
		}
		if it.Label != "" && text != nil {
			lw, _ := text.MeasureAt(it.Label, th.Typography.Body.Size)
			if it.Icon != "" {
				cw += segIconLabelGap
			}
			cw += lw
		}
		if cw > maxContent {
			maxContent = cw
		}
	}
	cellW := maxContent + 2*segCellPadX
	if min := th.Metrics.ControlHeight - 2*segPad; cellW < min {
		cellW = min
	}
	return cellW
}

func segCellRect(f render.Rect, i int, cellW float32) render.Rect {
	x := f.X + segPad + float32(i)*(cellW+segCellGap)
	return render.Rect{X: x, Y: f.Y + segPad, W: cellW, H: f.H - 2*segPad}
}

func paintSegmented(dl *render.DrawList, text *shape.Engine, f render.Rect, items []SegmentItem, selected, hover int, cellW float32) {
	th := theme.Current()
	dl.AddRoundedRectBorder(f, th.Radius.Medium, th.Stroke.Thin, th.ChromeMuted, th.Border)
	iconSz := th.Metrics.IconSizeSM
	style := th.Typography.Body
	for i, it := range items {
		active := i == selected
		cell := segCellRect(f, i, cellW)
		switch {
		case active:
			dl.AddRoundedRect(cell, th.Radius.Small, th.ListActive)
		case i == hover:
			dl.AddRoundedRect(cell, th.Radius.Small, th.ListHover)
		}
		fg := th.ForegroundSubtle
		if active {
			fg = th.Foreground
		}
		var content float32
		var lw, lh float32
		if it.Icon != "" {
			content += iconSz
		}
		if it.Label != "" {
			lw, lh = text.MeasureAt(it.Label, style.Size)
			if it.Icon != "" {
				content += segIconLabelGap
			}
			content += lw
		}
		x := cell.X + (cell.W-content)/2
		cy := cell.Y + cell.H/2
		if it.Icon != "" && frameIcons() != nil {
			frameIcons().Draw(dl, it.Icon, render.Rect{X: x, Y: cy - iconSz/2, W: iconSz, H: iconSz}, fg)
			x += iconSz + segIconLabelGap
		}
		if it.Label != "" {
			text.DrawStringTopAt(dl, it.Label, x, cy-lh/2, fg, style.Size)
		}
	}
}
