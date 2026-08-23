package ui

import (
	"time"

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

// ToastHost manages a bottom-right toast stack.
type ToastHost struct {
	host   *layout.Element
	toasts []toastEntry
	margin float32
	width  float32
}

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
	t.toasts = alive
}

func (t *ToastHost) variantColor(v ToastVariant) render.Color {
	th := theme.Current()
	switch v {
	case ToastSuccess:
		return th.Success
	case ToastWarning:
		return th.Warning
	case ToastError:
		return th.Error
	default:
		return th.Accent
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
	if d, ok := t.AnimationWait(); ok {
		c.Animate(d)
	}
	return layout.New(layout.Box().Size(0, 0))
}

// AnimationWait reports when a toast needs repaint for expiry.
func (t *ToastHost) AnimationWait() (time.Duration, bool) {
	t.prune()
	if len(t.toasts) == 0 {
		return 0, false
	}
	soonest := time.Until(t.toasts[0].until)
	for _, e := range t.toasts[1:] {
		d := time.Until(e.until)
		if d < soonest {
			soonest = d
		}
	}
	if soonest < 0 {
		soonest = 0
	}
	return soonest, true
}
