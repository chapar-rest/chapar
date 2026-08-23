package ui

import (
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
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

// Alert is an inline banner: accent bar + message.
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

func (n *Node) layoutAlert(c *Ctx) *layout.Element {
	th := c.Theme()
	d, _ := n.extra.(*alertData)
	variant := AlertInfo
	if d != nil {
		variant = d.variant
	}
	accent := alertAccent(th, variant)
	tint := accent
	tint.A = 0.15
	style := th.Typography.Body
	var tw, lh float32
	if eng := c.Text(); eng != nil {
		tw, lh = eng.MeasureAt(n.text, style.Size)
	} else {
		lh = style.LineHeight
		tw = style.Size * 0.5 * float32(len(n.text))
	}
	padY := th.Spacing.S
	msg := n.text
	el := layout.New(applyLayoutSpec(layout.Box().
		H(lh+2*padY).
		PaddingXY(th.Spacing.M, padY).
		Min(tw+2*th.Spacing.M, 0), n.spec))
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		f := el.Frame
		dl.AddRoundedRect(f, th.Radius.Small, tint)
		dl.AddRect(render.Rect{X: f.X, Y: f.Y, W: 3, H: f.H}, accent)
		_, h := text.MeasureAt(msg, style.Size)
		dl.PushClip(f)
		text.DrawStringTopAt(dl, msg, f.X+th.Spacing.M, f.Y+(f.H-h)/2, th.Foreground, style.Size)
		dl.PopClip()
	}
	if d != nil && d.dismiss {
		onDismiss := d.onDismiss
		el.OnMouse = func(e *layout.Element, m *input.Mouse) {
			if m.Released && e.Frame.Contains(m.X, m.Y) && onDismiss != nil {
				onDismiss()
			}
		}
	}
	return el
}

func alertAccent(th *theme.Theme, v AlertVariant) render.Color {
	switch v {
	case AlertWarning:
		return th.Warning
	case AlertError:
		return th.Error
	case AlertSuccess:
		return th.Success
	default:
		return th.Accent
	}
}
