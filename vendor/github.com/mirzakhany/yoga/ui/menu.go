package ui

import (
	"github.com/mirzakhany/yoga/icons"
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
	// Checked paints a check mark before the label. Menus with any checked
	// item indent every label so they stay aligned.
	Checked bool
}

// MenuSeparator is a divider between groups of menu items.
var MenuSeparator = MenuItem{Separator: true}

type Menu struct {
	host  *layout.Element
	items []MenuItem
	width float32

	Open  bool
	hover int

	// scrollY scrolls a menu taller than the room it has; maxH is that room,
	// fixed when the menu opens.
	scrollY float32
	maxH    float32

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

// visibleHeight is the menu frame height: every row, or maxH when the rows do
// not fit and the menu scrolls.
func (mu *Menu) visibleHeight() float32 {
	h := mu.height()
	if mu.maxH > 0 && h > mu.maxH {
		return mu.maxH
	}
	return h
}

func (mu *Menu) clampScroll() {
	if max := mu.height() - mu.host.Frame.H; mu.scrollY > max {
		mu.scrollY = max
	}
	if mu.scrollY < 0 {
		mu.scrollY = 0
	}
}

// itemAt returns the index of the row at screen y, or -1 past the last row.
func (mu *Menu) itemAt(y float32) int {
	top := mu.host.Frame.Y - mu.scrollY
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
// to stay inside the viewport recorded via SetViewport (if any). A menu too
// tall for the viewport scrolls. When there is room for a good number of rows
// below y it opens there with a shorter height rather than shifting up over
// whatever opened it, so the release of the opening click cannot land on an
// item. A checked item is scrolled into view.
func (mu *Menu) OpenAt(x, y float32) {
	mu.Open = true
	mu.scrollY = 0
	mu.maxH = 0
	h := mu.height()
	if viewportH > 0 {
		margin := theme.Current().Spacing.S
		mu.maxH = viewportH - 2*margin
		below := viewportH - y - margin
		if h > below && below >= f32min(h, 8*mu.itemHeight()) {
			mu.maxH = below
		}
		h = mu.visibleHeight()
	}
	x, y = clampToViewport(x, y, mu.width, h)
	mu.host.Style = layout.Box().Absolute(x, y).Size(mu.width, h)
	mu.host.ReapplyStyle()
	mu.host.Frame = render.Rect{X: x, Y: y, W: mu.width, H: h}
	top := float32(0)
	for i, it := range mu.items {
		if it.Checked && !it.Separator {
			mu.scrollY = top - (h-mu.rowHeight(i))/2
			break
		}
		top += mu.rowHeight(i)
	}
	mu.clampScroll()
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
		h := mu.visibleHeight()
		mu.host.Style.Height = h
		mu.host.Frame.H = h
		mu.clampScroll()
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
	// Labels wider than the configured menu width are clipped to the frame, and
	// rows stop inside the border so a hover fill cannot paint over it.
	bw := float32(th.Stroke.Thin)
	inner := render.Rect{X: f.X + bw, Y: f.Y + bw, W: f.W - 2*bw, H: f.H - 2*bw}
	dl.PushClip(inner)
	style := th.Typography.Body
	labelX := padX
	checkSz := th.Metrics.IconSizeSM
	if mu.hasChecked() {
		labelX += checkSz + th.Spacing.S
	}
	y := f.Y - mu.scrollY
	for i, it := range mu.items {
		if y >= f.Y+f.H {
			break
		}
		if y+mu.rowHeight(i) <= f.Y {
			y += mu.rowHeight(i)
			continue
		}
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
		if it.Checked {
			if sheet := frameIcons(); sheet != nil {
				box := render.Rect{X: row.X + padX, Y: row.Y + (itemH-checkSz)/2, W: checkSz, H: checkSz}
				sheet.Draw(dl, icons.Check, box, col)
			}
		}
		_, lh := text.MeasureAt(it.Label, style.Size)
		text.DrawStringTopAt(dl, it.Label, row.X+labelX, row.Y+(itemH-lh)/2, col, style.Size)
		if it.Shortcut != "" {
			hint := th.ForegroundMuted
			if it.Disabled {
				hint = th.ForegroundDisabled
			}
			sw, _ := text.MeasureAt(it.Shortcut, style.Size)
			text.DrawStringTopAt(dl, it.Shortcut, row.X+row.W-padX-sw, row.Y+(itemH-lh)/2, hint, style.Size)
		}
	}
	if total := mu.height(); total > f.H {
		// Scroll position thumb along the right edge.
		thumbH := f32max(f.H*f.H/total, 2*th.Spacing.M)
		thumbY := f.Y + (f.H-thumbH)*mu.scrollY/(total-f.H)
		thumbW := th.Spacing.XS
		dl.AddRoundedRect(render.Rect{X: f.X + f.W - thumbW - 2, Y: thumbY, W: thumbW, H: thumbH}, thumbW/2, th.ForegroundMuted)
	}
	dl.PopClip()
}

func (mu *Menu) hasChecked() bool {
	for _, it := range mu.items {
		if it.Checked && !it.Separator {
			return true
		}
	}
	return false
}

func (mu *Menu) onMouse(e *layout.Element, m *input.Mouse) {
	if !mu.Open {
		return
	}
	prev := mu.hover
	if e.Frame.Contains(m.X, m.Y) {
		if m.ScrollY != 0 && mu.height() > e.Frame.H {
			mu.scrollY -= m.ScrollY * 3 * 14
			mu.clampScroll()
			m.ScrollY = 0
			if mu.markPaint != nil {
				mu.markPaint()
			}
		}
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
