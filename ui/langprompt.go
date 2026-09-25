package ui

import (
	"fmt"
	"time"

	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/ui/langsrv"
)

func installToastID(l langsrv.Language) string { return "lsp-install-" + l.ID }

// languageServerConfig returns the settings in force for l.
func languageServerConfig(l langsrv.Language) (command string, custom bool) {
	for _, s := range langsrv.Effective(prefs.GetGlobalConfig().Spec.LanguageServers) {
		if s.Language == l.ID {
			return s.Command, s.Command != l.Command
		}
	}
	return l.Command, false
}

// promptInstall asks the user to install a missing language server, the way
// an editor offers to install a missing extension. Closing the prompt keeps
// it from coming back this session; Settings still offers Install.
func (a *App) promptInstall(l langsrv.Language) {
	host := a.toasts()
	if host == nil || a.lspPromptClosed[l.ID] || a.lang.Installing(l.ID) {
		return
	}
	command, custom := languageServerConfig(l)
	inst, canInstall := l.Installer()
	canInstall = canInstall && !custom

	var msg string
	switch {
	case custom:
		msg = fmt.Sprintf("Chapar could not find %s, the command set for %s in Settings.", command, l.Name)
	case canInstall:
		msg = fmt.Sprintf("Completion, hover, and diagnostics in %s editors need %s. Install it by running:\n%s",
			l.Name, command, inst)
	default:
		msg = fmt.Sprintf("Completion, hover, and diagnostics in %s editors need %s. Install it with: %s",
			l.Name, command, l.Install)
	}

	var actions []ui.ToastAction
	if canInstall {
		actions = append(actions, ui.ToastAction{Label: "Install", Primary: true, OnClick: func() {
			a.startInstall(l, inst)
		}})
	}
	if custom {
		actions = append(actions, ui.ToastAction{Label: "Open settings", Primary: true, OnClick: func() {
			if a.uiCtx != nil {
				a.openSettings(a.uiCtx)
			}
		}})
	} else if l.URL != "" {
		actions = append(actions, ui.ToastAction{Label: "Learn more", KeepOpen: true, OnClick: func() {
			if err := langsrv.OpenURL(l.URL); err != nil {
				logger.Error(fmt.Sprintf("opening %s: %v", l.URL, err))
			}
		}})
	}
	actions = append(actions, ui.ToastAction{Label: "Disable", OnClick: func() { a.disableLanguageServer(l) }})

	host.Notify(ui.ToastOpts{
		ID:        installToastID(l),
		Title:     l.Name + " language server is not installed",
		Message:   msg,
		Variant:   ui.ToastWarning,
		Actions:   actions,
		OnDismiss: func() { a.lspPromptClosed[l.ID] = true },
	})
}

// installLanguageServer installs the server for the language with config key
// id using its preferred installer. It backs the Install button in Settings.
func (a *App) installLanguageServer(id string) {
	l, ok := langsrv.ByID(id)
	if !ok {
		return
	}
	if inst, ok := l.Installer(); ok {
		a.startInstall(l, inst)
	}
}

func (a *App) startInstall(l langsrv.Language, inst langsrv.Installer) {
	if !a.lang.Install(l, inst) {
		return // already running
	}
	a.notifs.Add(fmt.Sprintf("Installing the %s language server: %s", l.Name, inst), ui.ToastInfo)
	if host := a.toasts(); host != nil {
		host.Notify(ui.ToastOpts{
			ID:      installToastID(l),
			Title:   "Installing the " + l.Name + " language server…",
			Message: "Running: " + inst.String() + "\nThe output is in the Console.",
			Variant: ui.ToastInfo,
			Actions: []ui.ToastAction{{Label: "Show console", KeepOpen: true, OnClick: a.console.Show}},
		})
	}
}

// reportInstalls logs installer output and reports how installs ended.
func (a *App) reportInstalls() {
	for _, ev := range a.lang.DrainInstalls() {
		l := ev.Language
		if !ev.Done {
			logger.Print(ev.Line)
			continue
		}
		host := a.toasts()
		if ev.Err == nil {
			msg := fmt.Sprintf("Installed the %s language server.", l.Name)
			logger.Info(msg)
			a.notifs.Add(msg, ui.ToastSuccess)
			a.lang.Restart(l.ID)
			if host != nil {
				host.Notify(ui.ToastOpts{
					ID:       installToastID(l),
					Title:    l.Name + " language server installed",
					Message:  "It starts in open " + l.Name + " editors now.",
					Variant:  ui.ToastSuccess,
					Duration: 5 * time.Second,
				})
			}
			continue
		}
		msg := fmt.Sprintf("Installing the %s language server failed: %v", l.Name, ev.Err)
		logger.Error(msg)
		a.notifs.Add(msg, ui.ToastError)
		if host != nil {
			inst := ev.Installer
			host.Notify(ui.ToastOpts{
				ID:      installToastID(l),
				Title:   "Could not install the " + l.Name + " language server",
				Message: ev.Err.Error() + "\nThe installer's output is in the Console.",
				Variant: ui.ToastError,
				Actions: []ui.ToastAction{
					{Label: "Show console", KeepOpen: true, OnClick: a.console.Show},
					{Label: "Retry", Primary: true, OnClick: func() { a.startInstall(l, inst) }},
				},
			})
		}
	}
}

// disableLanguageServer turns the server off in the saved settings.
func (a *App) disableLanguageServer(l langsrv.Language) {
	cfg := prefs.GetGlobalConfig()
	servers := langsrv.Effective(cfg.Spec.LanguageServers)
	for i := range servers {
		if servers[i].Language == l.ID {
			servers[i].Enabled = false
		}
	}
	cfg.Spec.LanguageServers.Servers = servers
	if err := prefs.UpdateGlobalConfig(cfg); err != nil {
		a.showError(err)
		return
	}
	msg := l.Name + " language server disabled. Turn it back on in Settings → Language servers."
	logger.Info(msg)
	a.notifs.Add(msg, ui.ToastInfo)
	if host := a.toasts(); host != nil {
		host.Notify(ui.ToastOpts{
			ID:       installToastID(l),
			Title:    l.Name + " language server disabled",
			Message:  "Turn it back on in Settings → Language servers.",
			Duration: 5 * time.Second,
		})
	}
}
