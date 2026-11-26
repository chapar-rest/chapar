package ui

import (
	"github.com/yogaui/yoga/icons"
	"github.com/yogaui/yoga/input"
	"github.com/yogaui/yoga/layout"
	"github.com/yogaui/yoga/render"
	"github.com/yogaui/yoga/shape"
	"github.com/yogaui/yoga/theme"
)

// AlertVariant selects alert severity styling.
type AlertVariant int

const (
	AlertInfo AlertVariant = iota
	AlertWarning
	AlertError
	AlertSuccess
)

type alertData struct {
	variant   AlertVariant
	onDismiss func()
	dismiss   bool
}

// Alert is an inline banner: a status icon and a message on the variant's
// tinted surface, with a close mark when Dismissable.
func Alert(message string, variant AlertVariant) *Node {
	return &Node{kind: kindAlert, text: message, extra: &alertData{variant: variant}}
}

// Dismissable enables click-to-dismiss and sets the callback.
func (n *Node) Dismissable(fn func()) *Node {
	if d, ok := n.extra.(*alertData); ok {
		d.dismiss = true
		d.onDismiss = fn
	}
	return n
}

type alertState struct{ hovered bool }

func (n *Node) layoutAlert(c *Ctx) *layout.Element {
	th := c.Theme()
	d, _ := n.extra.(*alertData)
	variant := AlertInfo
	if d != nil {
		variant = d.variant
	}
	accent, tint := alertColors(th, variant)
	icon := alertIcon(variant)
	dismiss := d != nil && d.dismiss
	style := th.Typography.Body
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = eng.MeasureAt(n.text, style.Size)
	} else {
		lh = style.LineHeight
		tw = style.Size * 0.5 * float32(len(n.text))
	}
	isz := th.Metrics.IconSizeSM
	gap := th.Spacing.S
	padX, padY := th.Spacing.M, th.Spacing.S
	closeW := float32(0)
	if dismiss {
		closeW = gap + isz
	}
	var st *alertState
	if dismiss {
		id := n.id
		if id == "" {
			id = autoID(c, "alert")
		}
		st = c.Widget(id, func() any { return &alertState{} }).(*alertState)
	}
	msg := n.text
	el := layout.New(applyLayoutSpec(layout.Box().
		H(max(lh, isz)+2*padY).
		PaddingXY(padX, padY).
		Min(tw+2*padX+isz+gap+closeW, 0), n.spec))
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		// The border is the status color blended into the tint, so the
		// banner reads as one shape without a loud stripe.
		border := theme.Flatten(alpha(accent, 0.45), tint)
		dl.AddRoundedRectBorder(f, th.Radius.Medium, th.Stroke.Thin, tint, border)
		cy := f.Y + f.H/2
		x := f.X + padX
		sheet := FrameIcons()
		if sheet != nil {
			sheet.Draw(dl, icon, render.Rect{X: x, Y: cy - isz/2, W: isz, H: isz}, accent)
		}
		x += isz + gap
		right := f.X + f.W - padX - closeW
		_, h := text.MeasureAt(msg, style.Size)
		dl.PushClip(render.Rect{X: x, Y: f.Y, W: max(0, right-x), H: f.H})
		text.DrawStringTopAt(dl, msg, x, cy-h/2, th.Foreground, style.Size)
		dl.PopClip()
		if dismiss && sheet != nil {
			cr := render.Rect{X: right + gap, Y: cy - isz/2, W: isz, H: isz}
			if st.hovered {
				hit := render.Rect{X: cr.X - 2, Y: cr.Y - 2, W: cr.W + 4, H: cr.H + 4}
				dl.AddRoundedRect(hit, th.Radius.Small, th.HoverOn(tint))
			}
			sheet.Draw(dl, icons.X, cr, th.Foreground)
		}
	}
	if dismiss {
		onDismiss := d.onDismiss
		el.OnMouse = func(e *layout.Element, m *input.Mouse) {
			inside := e.Frame.Contains(m.X, m.Y)
			trackHover(c, &st.hovered, inside)
			if inside {
				m.SetCursor(CursorPointer)
			}
			if m.Released && inside && onDismiss != nil {
				onDismiss()
				c.MarkNeedsPaint()
			}
		}
	}
	return el
}

// alertIcon is the status glyph for a variant, so severity does not rest on
// color alone.
func alertIcon(v AlertVariant) icons.Icon {
	switch v {
	case AlertWarning:
		return icons.TriangleAlert
	case AlertError:
		return icons.CircleX
	case AlertSuccess:
		return icons.CircleCheck
	default:
		return icons.Info
	}
}

// alertColors returns the rule color and the panel fill for a variant. The
// fill is the palette's status surface, which is built so Foreground still
// reads on it.
func alertColors(th *theme.Theme, v AlertVariant) (accent, fill render.Color) {
	switch v {
	case AlertWarning:
		return th.WarningForeground, th.WarningSurface
	case AlertError:
		return th.ErrorForeground, th.ErrorSurface
	case AlertSuccess:
		return th.SuccessForeground, th.SuccessSurface
	default:
		return th.InfoForeground, th.InfoSurface
	}
}
