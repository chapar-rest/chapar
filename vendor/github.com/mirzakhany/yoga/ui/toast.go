package ui

import (
	"fmt"
	"time"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

// ToastVariant selects toast styling.
type ToastVariant int

const (
	ToastInfo ToastVariant = iota
	ToastSuccess
	ToastWarning
	ToastError
)

type toastEntry struct {
	message string
	variant ToastVariant
	until   time.Time
}

// ToastAction is a button on an actionable toast (see ToastHost.Notify).
type ToastAction struct {
	Label   string
	Primary bool
	OnClick func()
	// KeepOpen leaves the toast up after the click; by default any action
	// dismisses it.
	KeepOpen bool
}

// ToastOpts configures an actionable toast: a card with a title, a wrapped
// message, and buttons — the way an editor asks "Install the language
// server?". Unlike Show's one-line toasts it takes clicks, and with no
// Duration it stays until the user acts on it or closes it.
type ToastOpts struct {
	// ID identifies the toast: Notify with an ID already showing replaces
	// that toast in place, and Dismiss(ID) removes it. Empty IDs are unique.
	ID       string
	Title    string
	Message  string
	Variant  ToastVariant
	Actions  []ToastAction
	Duration time.Duration // 0 keeps the toast until dismissed
	// OnDismiss runs when the user closes the toast with its close button.
	OnDismiss func()
}

type noticeEntry struct {
	opts  ToastOpts
	key   string // widget-id prefix, unique per Notify
	until time.Time
	el    *layout.Element // this frame's card host
}

// ToastHost manages a bottom-right toast stack.
type ToastHost struct {
	host    *layout.Element
	toasts  []toastEntry
	notices []noticeEntry
	seq     int
	margin  float32
	width   float32
}

// noticeWidth is the width of an actionable toast; wider than a plain toast
// so a sentence of explanation fits in a few lines.
const noticeWidth = 380

// NewToastHost builds a toast overlay host. The window Ctx owns a default
// host (c.Toasts()); construct a dedicated one only for tests or a second
// stack, and place that one in the view tree so Layout can register overlays.
func NewToastHost() *ToastHost {
	t := &ToastHost{margin: 16, width: 280}
	t.host = layout.New(layout.Box())
	t.host.Overlay = true
	t.host.Paint = t.paint
	return t
}

// Show enqueues a toast that auto-dismisses after d.
func (t *ToastHost) Show(message string, variant ToastVariant, d time.Duration) {
	if d <= 0 {
		d = 3 * time.Second
	}
	t.toasts = append(t.toasts, toastEntry{message: message, variant: variant, until: time.Now().Add(d)})
}

// Notify shows an actionable toast. See ToastOpts.
func (t *ToastHost) Notify(opts ToastOpts) {
	t.seq++
	e := noticeEntry{opts: opts, key: fmt.Sprintf("toast-n%d", t.seq)}
	if opts.Duration > 0 {
		e.until = time.Now().Add(opts.Duration)
	}
	if opts.ID != "" {
		for i := range t.notices {
			if t.notices[i].opts.ID == opts.ID {
				e.key = t.notices[i].key // keep widget state (hover) stable
				t.notices[i] = e
				return
			}
		}
	}
	t.notices = append(t.notices, e)
}

// Dismiss removes the actionable toast with the given ID, if showing.
func (t *ToastHost) Dismiss(id string) {
	for i := range t.notices {
		if t.notices[i].opts.ID == id {
			t.removeNotice(t.notices[i].key)
			return
		}
	}
}

// Showing reports whether an actionable toast with the given ID is up.
func (t *ToastHost) Showing(id string) bool {
	for _, n := range t.notices {
		if n.opts.ID == id {
			return true
		}
	}
	return false
}

func (t *ToastHost) removeNotice(key string) {
	for i := range t.notices {
		if t.notices[i].key == key {
			t.notices = append(t.notices[:i], t.notices[i+1:]...)
			return
		}
	}
}

// Position anchors the host to the bottom-right of the viewport.
func (t *ToastHost) Position(viewW, viewH float32) {
	t.host.Style = layout.Box().Absolute(0, 0).Size(viewW, viewH)
	t.host.ReapplyStyle()
	t.host.Frame = render.Rect{X: 0, Y: 0, W: viewW, H: viewH}
}

func (t *ToastHost) prune() {
	now := time.Now()
	alive := t.toasts[:0]
	for _, e := range t.toasts {
		if now.Before(e.until) {
			alive = append(alive, e)
		}
	}
	// Release the entries that were pruned; the tail of the backing array
	// still referenced them.
	clear(t.toasts[len(alive):])
	t.toasts = alive

	notices := t.notices[:0]
	for _, n := range t.notices {
		if n.until.IsZero() || now.Before(n.until) {
			notices = append(notices, n)
		}
	}
	clear(t.notices[len(notices):])
	t.notices = notices
}

func (t *ToastHost) variantIcon(v ToastVariant) (icons.Icon, Token) {
	switch v {
	case ToastSuccess:
		return icons.CircleCheck, TokenSuccessForeground
	case ToastWarning:
		return icons.TriangleAlert, TokenWarningForeground
	case ToastError:
		return icons.CircleAlert, TokenErrorForeground
	default:
		return icons.Info, TokenInfoForeground
	}
}

// plainStackHeight is the height the one-line toasts occupy, bottom-up.
func (t *ToastHost) plainStackHeight() float32 {
	if len(t.toasts) == 0 {
		return 0
	}
	th := theme.Current()
	itemH := th.Typography.Body.LineHeight + 2*th.Spacing.S
	return float32(len(t.toasts)) * (itemH + th.Spacing.S)
}

// layoutNotices places the actionable toasts above the plain ones, newest at
// the bottom. Each card is its own overlay sized from its wrapped text, so it
// can be positioned before layout runs.
func (t *ToastHost) layoutNotices(c *Ctx) {
	if len(t.notices) == 0 {
		return
	}
	th := c.Theme()
	vw, vh := c.Viewport()
	w := f32min(noticeWidth, vw-2*t.margin)
	y := vh - t.margin - t.plainStackHeight()
	for i := len(t.notices) - 1; i >= 0; i-- {
		n := &t.notices[i]
		view, h := t.noticeCard(c, *n, w)
		y -= h
		x := vw - t.margin - w
		inner := view.Layout(c)
		host := layout.New(layout.Box().Absolute(x, y).Size(w, h), inner)
		host.Overlay = true
		host.Paint = func(dl *render.DrawList, _ *shape.Engine) {
			drawElevationShadow(dl, host.Frame, th.Radius.Large, th.Elevation.ShadowMd)
		}
		host.OnMouse = func(e *layout.Element, m *input.Mouse) {
			if e.Frame.Contains(m.X, m.Y) {
				m.ScrollX, m.ScrollY = 0, 0
				m.Consumed = true // keep clicks off the page underneath
			}
		}
		n.el = host
		c.Overlay(host)
		y -= th.Spacing.S
	}
}

// noticeCard builds the view for one actionable toast and returns its height.
func (t *ToastHost) noticeCard(c *Ctx, n noticeEntry, w float32) (View, float32) {
	th := c.Theme()
	pad := th.Spacing.M
	gap := th.Spacing.S
	body := th.Typography.Body
	ctrl := c.controlHeight()
	key := n.key
	icon, tok := t.variantIcon(n.opts.Variant)

	header := Row(
		Icon(icon, th.Metrics.IconSizeMD, tok.Resolve(th)),
		Strong(n.opts.Title).Grow(1),
		IconButton(key+"-close", icons.X).OnClick(func() {
			t.removeNotice(key)
			if n.opts.OnDismiss != nil {
				n.opts.OnDismiss()
			}
		}),
	).Gap(gap).Align(AlignCenter)
	kids := []View{header}
	h := pad + ctrl

	if n.opts.Message != "" {
		lines := wrapText(frameText(), n.opts.Message, body.Size, w-2*pad)
		rows := make([]View, 0, len(lines))
		for _, l := range lines {
			rows = append(rows, Muted(l))
		}
		kids = append(kids, Column(rows...))
		h += gap + float32(len(lines))*body.LineHeight
	}

	if len(n.opts.Actions) > 0 {
		buttons := []View{Spacer()}
		for i, act := range n.opts.Actions {
			btn := Button(fmt.Sprintf("%s-act-%d", key, i), Text(act.Label))
			if act.Primary {
				btn = btn.Primary()
			}
			btn = btn.OnClick(func() {
				if !act.KeepOpen {
					t.removeNotice(key)
				}
				if act.OnClick != nil {
					act.OnClick()
				}
			})
			buttons = append(buttons, btn)
		}
		kids = append(kids, Row(buttons...).Gap(gap))
		h += gap + ctrl
	}
	h += pad

	card := Column(kids...).Gap(gap).Padding(pad).Grow(1).
		Background(TokenChrome).
		Style(Spec{}.Radius(th.Radius.Large).Border(tok, th.Stroke.Thin))
	return card, h
}

func (t *ToastHost) variantColor(v ToastVariant) render.Color {
	th := theme.Current()
	switch v {
	case ToastSuccess:
		return th.SuccessForeground
	case ToastWarning:
		return th.WarningForeground
	case ToastError:
		return th.ErrorForeground
	default:
		return th.InfoForeground
	}
}

func (t *ToastHost) paint(dl *render.DrawList, text *shape.Engine) {
	t.prune()
	if len(t.toasts) == 0 {
		return
	}
	th := theme.Current()
	f := t.host.Frame
	pad := th.Spacing.S
	style := th.Typography.Body
	itemH := style.LineHeight + 2*pad
	y := f.Y + f.H - t.margin
	for i := len(t.toasts) - 1; i >= 0; i-- {
		e := t.toasts[i]
		y -= itemH
		x := f.X + f.W - t.margin - t.width
		rect := render.Rect{X: x, Y: y, W: t.width, H: itemH}
		accent := t.variantColor(e.variant)
		bg := accent
		bg.A = 0.2
		r := th.Radius.Medium
		drawElevationShadow(dl, rect, r, th.Elevation.ShadowMd)
		dl.AddRoundedRectBorder(rect, r, th.Stroke.Thin, th.Chrome, accent)
		_, lh := text.MeasureAt(e.message, style.Size)
		dl.PushClip(render.Rect{X: x + pad, Y: y, W: t.width - 2*pad, H: itemH})
		text.DrawStringTopAt(dl, e.message, x+pad, y+(itemH-lh)/2, th.Foreground, style.Size)
		dl.PopClip()
		y -= th.Spacing.S
	}
}

// Layout registers the toast overlay and schedules the next expiry
// repaint. The returned element is a 0×0 placeholder safe to keep in the tree.
func (t *ToastHost) Layout(c *Ctx) *layout.Element {
	w, h := c.Viewport()
	t.Position(w, h)
	c.Overlay(t.host)
	t.layoutNotices(c)
	if d, ok := t.AnimationWait(); ok {
		c.Animate(d)
	}
	return layout.New(layout.Box().Size(0, 0))
}

// AnimationWait reports when a toast needs repaint for expiry.
func (t *ToastHost) AnimationWait() (time.Duration, bool) {
	t.prune()
	soonest, ok := time.Duration(0), false
	consider := func(until time.Time) {
		if until.IsZero() {
			return
		}
		if d := time.Until(until); !ok || d < soonest {
			soonest, ok = d, true
		}
	}
	for _, e := range t.toasts {
		consider(e.until)
	}
	for _, n := range t.notices {
		consider(n.until)
	}
	if !ok {
		return 0, false
	}
	if soonest < 0 {
		soonest = 0
	}
	return soonest, true
}
