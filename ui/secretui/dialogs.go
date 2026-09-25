// Package secretui holds the dialogs that set up, unlock and show the key that
// encrypts secret environment values. The env editor and the settings panel
// both drive the same flows from here.
package secretui

import (
	"errors"
	"fmt"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/secret"
)

// Dialog widths. Bodies are laid out at the width minus the body padding, so
// paragraphs wrap instead of running past the edge.
const (
	setupDialogW = 520
	keyDialogW   = 560
	passDialogW  = 460
)

// bodyWidth is the usable width inside a dialog of width w.
func bodyWidth(w, pad float32) float32 { return w - 2*pad }

// Deps is what the dialogs need from the app.
type Deps struct {
	Manager *secret.Manager
	Dialogs func() *ui.DialogHost
	// Clipboard copies the key when the user asks for it. Nil hides Copy.
	Clipboard func(string)
	Toast     func(string)
	Error     func(error)
	// Changed runs after the key state changed, so callers can redraw.
	Changed func()
}

func (d Deps) host() *ui.DialogHost {
	if d.Dialogs == nil {
		return nil
	}
	return d.Dialogs()
}

func (d Deps) fail(err error) {
	if err == nil {
		return
	}
	if d.Error != nil {
		d.Error(err)
		return
	}
	if h := d.host(); h != nil {
		h.ShowError("Secret key", err.Error(), nil)
	}
}

func (d Deps) changed() {
	if d.Changed != nil {
		d.Changed()
	}
}

// EnsureKey asks the user to set up or unlock the secret key, then reports
// through done whether one is available. done runs when the user finishes with
// the dialogs, not when EnsureKey returns.
func EnsureKey(d Deps, done func(ok bool)) {
	m := d.Manager
	if m == nil {
		d.fail(errors.New("secrets are not available in this session"))
		report(done, false)
		return
	}
	if m.Unlocked() {
		report(done, true)
		return
	}
	if m.Configured() {
		Unlock(d, done)
		return
	}
	setup(d, done)
}

func report(done func(ok bool), ok bool) {
	if done != nil {
		done(ok)
	}
}

// setup offers the two ways to get a first key: generate one, or paste one from
// another machine.
func setup(d Deps, done func(ok bool)) {
	h := d.host()
	if h == nil {
		report(done, false)
		return
	}
	m := d.Manager
	storeName := m.StoreName()
	available := m.StoreAvailable()

	where := fmt.Sprintf("Chapar will keep the key in your %s and never write it to disk in plain text.", storeName)
	if !available {
		where = "No system keyring is reachable here, so the key is derived from a passphrase you enter once per session."
	}

	h.Show(ui.DialogOpts{
		Title:  "Set up a secret key",
		Width:  setupDialogW,
		Height: 240,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			return ui.Column(
				ui.Paragraph("Secret values are encrypted with one key for this installation."),
				ui.Paragraph(where).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
				ui.Paragraph("Environment files sync as plain files, so keep a copy of the key to read them on another machine.").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
			).Gap(th.Spacing.S).Padding(th.Spacing.L).Grow(1).Width(bodyWidth(setupDialogW, th.Spacing.L))
		},
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() { report(done, false) }},
			{Label: "Paste existing key", OnClick: func() { importKey(d, done) }},
			{Label: generateLabel(available), Primary: true, OnClick: func() {
				if available {
					generate(d, done)
					return
				}
				newPassphrase(d, done)
			}},
		},
		OnDismiss: func() { report(done, false) },
	})
}

func generateLabel(storeAvailable bool) string {
	if storeAvailable {
		return "Generate key"
	}
	return "Set passphrase"
}

func generate(d Deps, done func(ok bool)) {
	key, err := d.Manager.Generate(secret.ModeOS)
	if err != nil {
		d.fail(err)
		report(done, false)
		return
	}
	d.changed()
	ShowKeyOnce(d, key, "Your secret key",
		"Store this somewhere safe. You need it to read your secret values on another machine, and Chapar cannot show it again if the keyring entry is lost.",
		func() { report(done, true) })
}

// ShowKeyOnce presents a key with a Copy button.
func ShowKeyOnce(d Deps, key, title, note string, done func()) {
	h := d.host()
	if h == nil {
		if done != nil {
			done()
		}
		return
	}
	actions := []ui.DialogAction{}
	if d.Clipboard != nil {
		actions = append(actions, ui.DialogAction{Label: "Copy", OnClick: func() {
			d.Clipboard(key)
			if d.Toast != nil {
				d.Toast("Secret key copied")
			}
		}})
	}
	actions = append(actions, ui.DialogAction{Label: "Done", Primary: true, OnClick: func() {
		if done != nil {
			done()
		}
	}})

	h.Show(ui.DialogOpts{
		Title:  title,
		Width:  keyDialogW,
		Height: 220,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			return ui.Column(
				ui.TextField("secret-key-once", key).Disabled(true).Grow(1),
				ui.Paragraph(note).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
			).Gap(th.Spacing.S).Padding(th.Spacing.L).Grow(1).Width(bodyWidth(keyDialogW, th.Spacing.L))
		},
		Actions:   actions,
		OnDismiss: func() {},
	})
}

// importKey asks for a key generated elsewhere.
func importKey(d Deps, done func(ok bool)) {
	h := d.host()
	if h == nil {
		report(done, false)
		return
	}
	h.ShowInput("Paste your secret key", "base64 key", func(value string) {
		if err := d.Manager.Import(value, modeFor(d.Manager)); err != nil {
			d.fail(err)
			report(done, false)
			return
		}
		d.changed()
		report(done, true)
	}, func() { report(done, false) })
}

func modeFor(m *secret.Manager) secret.Mode {
	if m.StoreAvailable() {
		return secret.ModeOS
	}
	return secret.ModePassphrase
}

func newPassphrase(d Deps, done func(ok bool)) {
	h := d.host()
	if h == nil {
		report(done, false)
		return
	}
	askPassphrase(d, "Choose a passphrase", "Used to derive your secret key", func(value string) {
		if err := d.Manager.SetPassphrase(value); err != nil {
			d.fail(err)
			report(done, false)
			return
		}
		d.changed()
		report(done, true)
	}, func() { report(done, false) })
}

// Unlock asks for whatever the configured mode needs to bring the key back.
func Unlock(d Deps, done func(ok bool)) {
	m := d.Manager
	if m.Mode() == secret.ModePassphrase {
		askPassphrase(d, "Unlock secrets", "Your passphrase", func(value string) {
			if err := m.UnlockWithPassphrase(value); err != nil {
				d.fail(err)
				report(done, false)
				return
			}
			d.changed()
			report(done, true)
		}, func() { report(done, false) })
		return
	}
	// OS mode with no readable key: the entry was removed, or this is a new
	// machine holding synced environment files.
	h := d.host()
	if h == nil {
		report(done, false)
		return
	}
	h.Show(ui.DialogOpts{
		Title:  "Secret key not found",
		Width:  setupDialogW,
		Height: 220,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			return ui.Column(
				ui.Paragraph(fmt.Sprintf("Chapar could not read the secret key from your %s.", m.StoreName())),
				ui.Paragraph("Paste the key you saved when it was created to read your secret values again.").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
			).Gap(th.Spacing.S).Padding(th.Spacing.L).Grow(1).Width(bodyWidth(setupDialogW, th.Spacing.L))
		},
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() { report(done, false) }},
			{Label: "Paste key", Primary: true, OnClick: func() { importKey(d, done) }},
		},
		OnDismiss: func() { report(done, false) },
	})
}

// askPassphrase shows a masked single-field dialog.
func askPassphrase(d Deps, title, placeholder string, onOK func(string), onCancel func()) {
	h := d.host()
	if h == nil {
		if onCancel != nil {
			onCancel()
		}
		return
	}
	value := ""
	h.Show(ui.DialogOpts{
		Title:  title,
		Width:  passDialogW,
		Height: 180,
		Body: func(c *ui.Ctx) ui.View {
			th := c.Theme()
			return ui.Column(
				ui.TextField("secret-passphrase", value).
					Placeholder(placeholder).
					Password(true).
					IconStart(icons.Key).
					OnChange(func(s string) { value = s }).
					Grow(1),
			).Gap(th.Spacing.S).Padding(th.Spacing.L).Grow(1).Width(bodyWidth(passDialogW, th.Spacing.L))
		},
		Actions: []ui.DialogAction{
			{Label: "Cancel", OnClick: func() {
				if onCancel != nil {
					onCancel()
				}
			}},
			{Label: "Unlock", Primary: true, OnClick: func() {
				if value == "" {
					return
				}
				onOK(value)
			}},
		},
		OnDismiss: func() {
			if onCancel != nil {
				onCancel()
			}
		},
	})
}
