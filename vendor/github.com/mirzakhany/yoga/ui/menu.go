package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ----------------------------------------------------------------------------
// Menu: an absolutely-positioned overlay that is painted and
// hit-tested on top of the normal tree (Z-axis ordering). The menu paints its
// own item rows rather than nesting child elements, which keeps overlay
// geometry self-contained.
// ----------------------------------------------------------------------------

// MenuItem is a single selectable menu entry.
type MenuItem struct {
	Label    string
	OnSelect func()
	// Shortcut is a key hint painted right-aligned, such as "⌘C".
	Shortcut string
	// Disabled items paint dimmed and ignore clicks.
	Disabled bool
	// Separator makes the entry a thin divider line; other fields are ignored.
	Separator bool
}

// MenuSeparator is a divider between groups of menu items.
var MenuSeparator = MenuItem{Separator: true}

type Menu struct {
	host  *layout.Element
	items []MenuItem
	width float32

	Open  bool
	hover int

	markPaint func()
}

// triggerMenuWidth returns the menu width for a trigger: at least the styled
// width and as wide as the laid-out trigger frame when Grow expands it.
func triggerMenuWidth(styleW, frameW float32) float32 {
	w := styleW
	if frameW > w {
		w = frameW
	}
	return w
}

// NewMenu builds a closed overlay menu. Overlay positioning uses absolute
// Left/Top as screen coordinates.
func NewMenu(width float32, items []MenuItem) *Menu {
	mu := &Menu{items: items, width: width, hover: -1}
	mu.host = layout.New(layout.Box())
	mu.host.Overlay = true // render above and hit-test before the base tree
	mu.host.Paint = mu.paint
	mu.host.OnMouse = mu.onMouse
	return mu
}

// BindPaint marks the menu so hover/open changes request a frame present.
func (mu *Menu) BindPaint(c *Ctx) {
	if c != nil {
		mu.markPaint = c.MarkNeedsPaint
	}
}

func (mu *Menu) itemHeight() float32 {
	th := theme.Current()
	if th.Metrics.MenuItemHeight > 0 {
		return th.Metrics.MenuItemHeight
	}
	return th.Metrics.ControlHeight
}

func (mu *Menu) separatorHeight() float32 { return theme.Current().Spacing.S + 1 }

func (mu *Menu) rowHeight(i int) float32 {
	if mu.items[i].Separator {
		return mu.separatorHeight()
	}
	return mu.itemHeight()
}

func (mu *Menu) height() float32 {
	var h float32
	for i := range mu.items {
		h += mu.rowHeight(i)
	}
	return h
}

// itemAt returns the index of the row at screen y, or -1 past the last row.
func (mu *Menu) itemAt(y float32) int {
	top := mu.host.Frame.Y
	for i := range mu.items {
		top += mu.rowHeight(i)
		if y < top {
			return i
		}
	}
	return -1
}

// selectable reports whether row i is an enabled item.
func (mu *Menu) selectable(i int) bool {
	return i >= 0 && i < len(mu.items) && !mu.items[i].Separator && !mu.items[i].Disabled
}

// OpenAt positions and shows the menu at the given screen coordinates, shifted
// to stay inside the viewport recorded via SetViewport (if any).
func (mu *Menu) OpenAt(x, y float32) {
	mu.Open = true
	h := mu.height()
	x, y = clampToViewport(x, y, mu.width, h)
	mu.host.Style = layout.Box().Absolute(x, y).Size(mu.width, h)
	mu.host.ReapplyStyle()
	mu.host.Frame = render.Rect{X: x, Y: y, W: mu.width, H: h}
}

func (mu *Menu) overlay() *layout.Element { return mu.host }

// Close hides the menu.
func (mu *Menu) Close() { mu.Open = false; mu.hover = -1 }

// SetItems replaces the menu's entries. Call before OpenAt when the items depend
// on context (e.g. which tree row was right-clicked). If the menu is already
// open, its height is refreshed in place to match the new item count.
func (mu *Menu) SetItems(items []MenuItem) {
	mu.items = items
	if mu.Open {
		h := mu.height()
		mu.host.Style.Height = h
		mu.host.Frame.H = h
	}
}

func (mu *Menu) paint(dl *render.DrawList, text *shape.Engine) {
	if !mu.Open {
		return
	}
	th := theme.Current()
	f := mu.host.Frame
	itemH := mu.itemHeight()
	padX := th.Spacing.MNudge
	r := th.Radius.Medium
	drawElevationShadow(dl, f, r, th.Elevation.ShadowMd)
	dl.AddRoundedRectBorder(f, r, th.Stroke.Thin, th.Chrome, th.Border)
	// Labels wider than the configured menu width are clipped to the frame.
	dl.PushClip(f)
	style := th.Typography.Body
	y := f.Y
	for i, it := range mu.items {
		if it.Separator {
			sepH := mu.separatorHeight()
			line := render.Rect{X: f.X + padX/2, Y: y + sepH/2, W: f.W - padX, H: th.Stroke.Thin}
			dl.AddRect(line, th.Border)
			y += sepH
			continue
		}
		row := render.Rect{X: f.X, Y: y, W: f.W, H: itemH}
		y += itemH
		if i == mu.hover && !it.Disabled {
			dl.AddRect(row, th.ListHover)
		}
		col := th.Foreground
		if it.Disabled {
			col = th.ForegroundDisabled
		}
		_, lh := text.MeasureAt(it.Label, style.Size)
		text.DrawStringTopAt(dl, it.Label, row.X+padX, row.Y+(itemH-lh)/2, col, style.Size)
		if it.Shortcut != "" {
			hint := th.ForegroundMuted
			if it.Disabled {
				hint = th.ForegroundDisabled
			}
			sw, _ := text.MeasureAt(it.Shortcut, style.Size)
			text.DrawStringTopAt(dl, it.Shortcut, row.X+row.W-padX-sw, row.Y+(itemH-lh)/2, hint, style.Size)
		}
	}
	dl.PopClip()
}

func (mu *Menu) onMouse(e *layout.Element, m *input.Mouse) {
	if !mu.Open {
		return
	}
	prev := mu.hover
	if e.Frame.Contains(m.X, m.Y) {
		idx := mu.itemAt(m.Y)
		mu.hover = idx
		m.Consumed = true // block all events (including hover) from reaching layers below
		if m.Pressed || m.RightPressed {
			m.KeepFocus = true // the item acts on whatever owns the menu
		}
		if m.Released && mu.selectable(idx) {
			if fn := mu.items[idx].OnSelect; fn != nil {
				fn()
			}
			mu.Close()
			if mu.markPaint != nil {
				mu.markPaint()
			}
		}
	} else {
		mu.hover = -1
		if m.Pressed { // click outside closes the menu
			mu.Close()
			m.Consumed = true
			m.KeepFocus = true
			if mu.markPaint != nil {
				mu.markPaint()
			}
		} else if m.RightPressed {
			// Right-click elsewhere closes this menu and lets the widget under
			// the pointer open its own.
			mu.Close()
			if mu.markPaint != nil {
				mu.markPaint()
			}
		}
	}
	if mu.hover != prev && mu.markPaint != nil {
		mu.markPaint()
	}
}
