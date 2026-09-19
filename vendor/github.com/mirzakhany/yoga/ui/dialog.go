package ui

import (
	"fmt"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
)

// DialogAction is a button in a dialog footer.
type DialogAction struct {
	Label    string
	Primary  bool
	Disabled bool
	OnClick  func()
}

// DialogSeverity styles the title/border of message dialogs.
type DialogSeverity int

const (
	DialogSeverityNone DialogSeverity = iota
	DialogSeverityInfo
	DialogSeverityWarning
	DialogSeverityError
)

// DialogOpts configures one Show of a DialogHost.
type DialogOpts struct {
	Title     string
	Width     float32 // default 480 if 0
	Height    float32 // default 360 if 0
	Body      func(c *Ctx) View
	Actions   []DialogAction
	OnDismiss func() // Escape
	Severity  DialogSeverity
}

// DialogHost manages modal dialogs with scrim.
type DialogHost struct {
	scrim *Scrim
	panel *layout.Element
	Open  bool
	opts  DialogOpts
	// inputValue holds the controlled text for ShowInput.
	inputValue string
}

var _ View = (*DialogHost)(nil)
var _ Focusable = (*DialogHost)(nil)

// NewDialogHost builds a dialog host. The window Ctx owns a default host
// (c.Dialogs()); construct a dedicated one only for tests or a second
// picker, and place that one in the view tree so Layout can register overlays.
func NewDialogHost() *DialogHost {
	return &DialogHost{scrim: NewScrim()}
}

// Show opens a dialog with custom size, body layout, and footer actions.
func (d *DialogHost) Show(opts DialogOpts) {
	d.opts = opts
	d.Open = true
}

// showMessage opens a sized message dialog with the given footer actions.
func (d *DialogHost) showMessage(title, message string, severity DialogSeverity, actions []DialogAction, onDismiss func()) {
	th := theme.Current()
	width := float32(360)
	pad := th.Spacing.L
	style := th.Typography.Body
	bodyLines := wrapText(frameText(), message, style.Size, width-2*pad)
	bodyH := float32(len(bodyLines))*style.LineHeight + pad
	titleH := th.Typography.Subtitle.LineHeight
	if severity != DialogSeverityNone {
		titleH = f32max(titleH, th.Metrics.IconSizeMD)
	}
	footerH := th.Metrics.ControlHeight + pad
	height := pad + titleH + pad + bodyH + pad + footerH

	d.Show(DialogOpts{
		Title:    title,
		Width:    width,
		Height:   height,
		Severity: severity,
		Body: func(c *Ctx) View {
			th := c.Theme()
			pad := th.Spacing.L
			lines := make([]View, 0, len(bodyLines))
			for _, line := range bodyLines {
				lines = append(lines, Muted(line))
			}
			return Column(lines...).Padding(pad)
		},
		Actions:   actions,
		OnDismiss: onDismiss,
	})
}

// showOKMessage opens a message dialog with a single primary OK button.
func (d *DialogHost) showOKMessage(title, message string, severity DialogSeverity, onOK func()) {
	dismiss := func() {
		if onOK != nil {
			onOK()
		}
	}
	d.showMessage(title, message, severity,
		[]DialogAction{{Label: "OK", Primary: true, OnClick: dismiss}},
		dismiss,
	)
}

// ShowInfo opens an informational message dialog.
func (d *DialogHost) ShowInfo(title, message string, onOK func()) {
	d.showOKMessage(title, message, DialogSeverityInfo, onOK)
}

// ShowWarning opens a warning message dialog.
func (d *DialogHost) ShowWarning(title, message string, onOK func()) {
	d.showOKMessage(title, message, DialogSeverityWarning, onOK)
}

// ShowError opens an error message dialog.
func (d *DialogHost) ShowError(title, message string, onOK func()) {
	d.showOKMessage(title, message, DialogSeverityError, onOK)
}

// ShowAction opens a Yes/No confirmation dialog. Escape runs onNo.
func (d *DialogHost) ShowAction(title, message string, onYes, onNo func()) {
	d.showMessage(title, message, DialogSeverityNone,
		[]DialogAction{
			{Label: "No", OnClick: onNo},
			{Label: "Yes", Primary: true, OnClick: onYes},
		},
		onNo,
	)
}

func dialogSeverityStyle(s DialogSeverity) (icon icons.Icon, color Token) {
	switch s {
	case DialogSeverityInfo:
		return icons.CircleCheck, TokenAccent
	case DialogSeverityWarning:
		return icons.TriangleAlert, TokenWarning
	case DialogSeverityError:
		return icons.CircleAlert, TokenError
	default:
		return icons.Icon{}, TokenUnset
	}
}

// ShowInput opens a dialog with a text field.
func (d *DialogHost) ShowInput(title, placeholder string, onOK func(value string), onCancel func()) {
	th := theme.Current()
	pad := th.Spacing.L
	titleH := th.Typography.Subtitle.LineHeight
	bodyH := th.Metrics.ControlHeight + pad
	footerH := th.Metrics.ControlHeight + pad
	height := pad + titleH + pad + bodyH + pad + footerH

	d.inputValue = ""
	d.Show(DialogOpts{
		Title:  title,
		Width:  360,
		Height: height,
		Body: func(c *Ctx) View {
			th := c.Theme()
			pad := th.Spacing.L
			return Column(
				TextField("__dialog-input", d.inputValue).
					Placeholder(placeholder).
					OnChange(func(s string) { d.inputValue = s }).
					DefaultFocus(),
			).Padding(pad).Gap(th.Spacing.S)
		},
		Actions: []DialogAction{
			{Label: "Cancel", OnClick: onCancel},
			{Label: "OK", Primary: true, OnClick: func() {
				if onOK != nil {
					onOK(d.inputValue)
				}
			}},
		},
		OnDismiss: onCancel,
	})
}

// Layout registers the scrim and panel while open.
func (d *DialogHost) Layout(c *Ctx) *layout.Element {
	if !d.Open {
		return layout.New(layout.Box().Size(0, 0))
	}
	vw, vh := c.Viewport()
	th := c.Theme()
	dw, dh := clampModalSize(vw, vh, d.opts.Width, d.opts.Height, 0, 0, th)
	d.panel = layoutModalPanel(c, d.scrim, d.chrome(c), dw, dh)
	if c.Focus() != nil {
		c.Focus().SetModal(d)
	}
	return layout.New(layout.Box().Size(0, 0))
}

func (d *DialogHost) chrome(c *Ctx) View {
	th := c.Theme()
	var body View
	if d.opts.Body != nil {
		body = d.opts.Body(c)
	}

	buttons := make([]View, 0, len(d.opts.Actions)+1)
	buttons = append(buttons, Spacer())
	for i, act := range d.opts.Actions {
		btn := Button(fmt.Sprintf("dlg-act-%d", i), Text(act.Label))
		if act.Primary {
			btn = btn.Primary()
		}
		if act.Disabled {
			btn = btn.Disabled(true)
		}
		actCopy := act
		btn = btn.OnClick(func() {
			d.Close()
			if actCopy.OnClick != nil {
				actCopy.OnClick()
			}
		})
		buttons = append(buttons, btn)
	}
	footer := Row(buttons...).Gap(th.Spacing.S).Padding(th.Spacing.M)

	kids := make([]View, 0, 4)
	borderTok := TokenBorder
	if d.opts.Title != "" {
		titleColor := TokenForeground
		var titleKids []View
		if icon, tok := dialogSeverityStyle(d.opts.Severity); !icon.Empty() {
			titleColor = tok
			borderTok = tok
			titleKids = append(titleKids, Icon(icon, th.Metrics.IconSizeMD, tok.Resolve(th)))
		}
		titleKids = append(titleKids, Text(d.opts.Title).Style(Spec{}.TextColor(titleColor)))
		kids = append(kids,
			Row(titleKids...).Gap(th.Spacing.S).Padding(th.Spacing.M),
			HLine(th.Stroke.Thin, th.Border),
		)
	}
	if body != nil {
		kids = append(kids, ViewOf(body).Grow(1))
	}
	kids = append(kids, footer)

	return Column(kids...).Grow(1).Background(TokenChrome).
		Style(Spec{}.Radius(th.Radius.Large).Border(borderTok, th.Stroke.Thin))
}

// Close hides the dialog.
func (d *DialogHost) Close() {
	d.Open = false
	d.scrim.Hide()
}

func (d *DialogHost) Focus()                   {}
func (d *DialogHost) Blur()                    {}
func (d *DialogHost) Focused() bool            { return d.Open }
func (d *DialogHost) CapturesTab() bool        { return false }
func (d *DialogHost) FocusOnClick() bool       { return false }
func (d *DialogHost) FocusEl() *layout.Element { return d.panel }
func (d *DialogHost) HandleText([]rune)        {}

func (d *DialogHost) HandleKeys(keys []input.KeyEvent) {
	if !d.Open {
		return
	}
	for _, ev := range keys {
		if ev.Key == input.KeyEscape {
			d.Close()
			if d.opts.OnDismiss != nil {
				d.opts.OnDismiss()
			}
			return
		}
	}
}
