package settings

import (
	"fmt"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/secret"
	"github.com/chapar-rest/chapar/uiv2/langsrv"
	"github.com/chapar-rest/chapar/uiv2/scriptsrv"
	"github.com/chapar-rest/chapar/uiv2/secretui"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const (
	catGeneral = iota
	catAppearance
	catScripting
	catEditor
	catLanguageServers
	catSecurity
	catData
)

// Panel is the settings dialog body. Open it with c.Dialogs().Show(DialogOpts{Body: panel.Layout, ...}).
type Panel struct {
	category     int
	draft        domain.GlobalConfig
	dirty        bool
	pathWarn     bool
	lang         *langsrv.Service
	scripts      *scriptsrv.Service
	secrets      secretui.Deps
	install      func(language string)
	onAppearance func(domain.GlobalConfigSpec)
}

// New builds the settings panel. install installs the default language
// server for a language (config key); it backs the Install buttons.
func New(lang *langsrv.Service, scripts *scriptsrv.Service, install func(language string), onAppearance func(domain.GlobalConfigSpec)) *Panel {
	return &Panel{lang: lang, scripts: scripts, install: install, onAppearance: onAppearance}
}

func (p *Panel) Prepare() {
	p.draft = prefs.GetGlobalConfig()
	if p.draft.Spec.General.UIFontSize <= 0 {
		p.draft.Spec.General.UIFontSize = 14
	}
	if p.draft.Spec.Editor.FontSize <= 0 {
		p.draft.Spec.Editor.FontSize = 12
	}
	p.fillLanguageServers()
	p.dirty = false
	p.pathWarn = false
	p.category = catGeneral
}

func (p *Panel) Dirty() bool    { return p.dirty }
func (p *Panel) PathWarn() bool { return p.pathWarn }
func (p *Panel) Draft() domain.GlobalConfig {
	return p.draft
}

func (p *Panel) LoadDefaults() {
	p.draft = *domain.GetDefaultGlobalConfig()
	p.fillLanguageServers()
	p.dirty = true
	p.previewAppearance()
}

func (p *Panel) Cancel() {
	if p.onAppearance != nil {
		saved := prefs.GetGlobalConfig()
		p.onAppearance(saved.Spec)
	}
}

func (p *Panel) Save() error {
	return prefs.UpdateGlobalConfig(p.draft)
}

func (p *Panel) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Row(
		ui.Nav("settings-nav", ui.NavVertical, ui.NavIconLeft,
			ui.NavItem{ID: "general", Label: "General", Icon: icons.House},
			ui.NavItem{ID: "appearance", Label: "Appearance", Icon: icons.Palette},
			ui.NavItem{ID: "scripting", Label: "Scripting", Icon: icons.Terminal},
			ui.NavItem{ID: "editor", Label: "Editor", Icon: icons.Pen},
			ui.NavItem{ID: "language-servers", Label: "Language servers", Icon: icons.Code},
			ui.NavItem{ID: "security", Label: "Security", Icon: icons.Lock},
			ui.NavItem{ID: "data", Label: "Data", Icon: icons.Folder},
		).Selected(p.category).OnSelectItem(func(i int, _ string) { p.category = i }).Width(200),
		ui.VLine(th.Stroke.Thin, th.Border),
		ui.Scroll("settings-form", p.form(c)).Grow(1),
	).Align(ui.AlignStretch).Grow(1).Background(ui.TokenSurface)
}

func (p *Panel) mark() { p.dirty = true }

func (p *Panel) previewAppearance() {
	if p.onAppearance != nil {
		p.onAppearance(p.draft.Spec)
	}
}

func (p *Panel) form(c *ui.Ctx) ui.View {
	th := c.Theme()
	g := &p.draft.Spec
	switch p.category {
	case catAppearance:
		themeOpts := yogaThemeOptions()
		return ui.Form("settings-appearance",
			ui.FormSelect("theme", "Theme", "Application color scheme", themeOpts, selectIndex(g.General.Theme, themeOpts), func(v string) {
				g.General.Theme = v
				p.mark()
				p.previewAppearance()
			}),
			ui.FormNumber("uiFontSize", "UI font size", "Application text size", float64(g.General.UIFontSize), 10, 22, 1, func(v float64) {
				g.General.UIFontSize = int(v)
				p.mark()
				p.previewAppearance()
			}),
			ui.FormNumber("editorFontSize", "Editor font size", "Code editor text size", float64(g.Editor.FontSize), 8, 32, 1, func(v float64) {
				g.Editor.FontSize = int(v)
				p.mark()
				p.previewAppearance()
			}),
			ui.FormSwitch("horizontalSplit", "Horizontal request/response split", "Stack request above response", g.General.UseHorizontalSplit, func(v bool) {
				g.General.UseHorizontalSplit = v
				p.mark()
			}),
			ui.FormSwitch("hideNavbar", "Hide navbar", "Hide the left navigation bar", g.General.HideNavbar, func(v bool) {
				g.General.HideNavbar = v
				p.mark()
				p.previewAppearance()
			}),
		).Padding(th.Spacing.M)
	case catScripting:
		return p.scriptingForm(th, g)
	case catLanguageServers:
		return p.languageServersForm(th, g)
	case catSecurity:
		return p.securityForm(th)
	case catEditor:
		indentOpts := []ui.SelectOption{
			{Label: "Spaces", Value: domain.IndentationSpaces},
			{Label: "Tabs", Value: domain.IndentationTabs},
		}
		return ui.Form("settings-editor",
			ui.FormText("fontFamily", "Font family", "Editor font", g.Editor.FontFamily, func(v string) {
				g.Editor.FontFamily = v
				p.mark()
				p.previewAppearance()
			}),
			ui.FormNumber("fontSize", "Font size", "Editor font size", float64(g.Editor.FontSize), 8, 32, 1, func(v float64) {
				g.Editor.FontSize = int(v)
				p.mark()
				p.previewAppearance()
			}),
			ui.FormSelect("indentation", "Indentation", "Spaces or tabs", indentOpts, selectIndex(g.Editor.Indentation, indentOpts), func(v string) {
				g.Editor.Indentation = v
				p.mark()
			}),
			ui.FormNumber("tabWidth", "Tab width", "Width of a tab stop", float64(g.Editor.TabWidth), 1, 16, 1, func(v float64) {
				g.Editor.TabWidth = int(v)
				p.mark()
				p.previewAppearance()
			}),
			ui.FormSwitch("autoCloseBrackets", "Auto close brackets", "Insert matching brackets", g.Editor.AutoCloseBrackets, func(v bool) {
				g.Editor.AutoCloseBrackets = v
				p.mark()
			}),
			ui.FormSwitch("autoCloseQuotes", "Auto close quotes", "Insert matching quotes", g.Editor.AutoCloseQuotes, func(v bool) {
				g.Editor.AutoCloseQuotes = v
				p.mark()
			}),
			ui.FormSwitch("showLineNumbers", "Show line numbers", "Display gutter numbers", g.Editor.ShowLineNumbers, func(v bool) {
				g.Editor.ShowLineNumbers = v
				p.mark()
			}),
			ui.FormSwitch("wrapLines", "Wrap lines", "Soft-wrap long lines", g.Editor.WrapLines, func(v bool) {
				g.Editor.WrapLines = v
				p.mark()
			}),
			ui.FormNumber("highlightLimit", "Syntax highlighting limit (KB)",
				"Larger bodies and files are shown without colors; a highlighted document takes many times its size in memory. Applies to documents opened from now on.",
				float64(g.Editor.HighlightLimitBytes()>>10), 16, 256<<10, 256, func(v float64) {
					g.Editor.HighlightLimitKB = int(v)
					p.mark()
				}),
		).Padding(th.Spacing.M)
	case catData:
		return ui.Form("settings-data",
			ui.FormText("workspacePath", "Workspace path", "Absolute path to the workspace folder", g.Data.WorkspacePath, func(v string) {
				g.Data.WorkspacePath = v
				p.pathWarn = true
				p.mark()
			}),
		).Padding(th.Spacing.M)
	default:
		httpOpts := []ui.SelectOption{
			{Label: "HTTP/1.1", Value: "http/1.1"},
			{Label: "HTTP/2", Value: "http/2"},
		}
		return ui.Form("settings-general",
			ui.FormSelect("httpVersion", "HTTP version", "Version used for HTTP requests", httpOpts, selectIndex(g.General.HTTPVersion, httpOpts), func(v string) {
				g.General.HTTPVersion = v
				p.mark()
			}),
			ui.FormNumber("timeout", "Request timeout (sec)", "Zero means never", float64(g.General.RequestTimeoutSec), 0, 600, 1, func(v float64) {
				g.General.RequestTimeoutSec = int(v)
				p.mark()
			}),
			ui.FormNumber("responseSize", "Response size (MB)", "Zero means unlimited", float64(g.General.ResponseSizeMb), 0, 1024, 1, func(v float64) {
				g.General.ResponseSizeMb = int(v)
				p.mark()
			}),
			ui.FormSwitch("followRedirects", "Follow redirects", "Follow 3xx responses", g.General.FollowRedirects, func(v bool) {
				g.General.FollowRedirects = v
				p.mark()
			}),
			ui.FormSwitch("validateTLS", "Validate TLS certificates", "Verify server certificates", g.General.VaidateTLSCertificates, func(v bool) {
				g.General.VaidateTLSCertificates = v
				p.mark()
			}),
			ui.FormSwitch("noCache", "Send no-cache header", "Add Cache-Control: no-cache", g.General.SendNoCacheHeader, func(v bool) {
				g.General.SendNoCacheHeader = v
				p.mark()
			}),
			ui.FormSwitch("agent", "Send Chapar agent header", "Identify Chapar in User-Agent", g.General.SendChaparAgentHeader, func(v bool) {
				g.General.SendChaparAgentHeader = v
				p.mark()
			}),
		).Padding(th.Spacing.M)
	}
}

func (p *Panel) scriptingForm(th *theme.Theme, g *domain.GlobalConfigSpec) ui.View {
	langOpts := []ui.SelectOption{{Label: "Python", Value: "python"}}
	items := []ui.FormItem{
		ui.FormSwitch("enable", "Enable", "Enable scripting for pre/post request triggers", g.Scripting.Enabled, func(v bool) {
			g.Scripting.Enabled = v
			p.mark()
		}),
		ui.FormSelect("language", "Language", "Scripting language", langOpts, selectIndex(g.Scripting.Language, langOpts), func(v string) {
			g.Scripting.Language = v
			p.mark()
		}),
		ui.FormSwitch("useDocker", "Use Docker", "Run the scripting engine in Docker", g.Scripting.UseDocker, func(v bool) {
			g.Scripting.UseDocker = v
			p.mark()
		}),
	}
	if g.Scripting.UseDocker {
		items = append(items, ui.FormText("dockerImage", "Docker image", "Image used when Docker is enabled", g.Scripting.DockerImage, func(v string) {
			g.Scripting.DockerImage = v
			p.mark()
		}))
	} else {
		items = append(items,
			ui.FormText("executablePath", "Executable path", "Local scripting binary", g.Scripting.ExecutablePath, func(v string) {
				g.Scripting.ExecutablePath = v
				p.mark()
			}),
			ui.FormText("serverScriptPath", "Server script path", "Where Chapar writes the server script", g.Scripting.ServerScriptPath, func(v string) {
				g.Scripting.ServerScriptPath = v
				p.mark()
			}),
		)
	}
	items = append(items, ui.FormNumber("port", "Port", "HTTP port for the scripting server", float64(g.Scripting.Port), 1, 65535, 1, func(v float64) {
		g.Scripting.Port = int(v)
		p.mark()
	}))
	return ui.Column(
		p.executorStatus(th),
		ui.Form("settings-scripting", items...),
	).Gap(th.Spacing.M).Padding(th.Spacing.M)
}

// executorStatus shows whether the script executor runs, with a Restart
// button. Like language servers, Restart applies the saved settings.
func (p *Panel) executorStatus(th *theme.Theme) ui.View {
	if p.scripts == nil {
		return nil
	}
	saved := prefs.GetGlobalConfig().Spec.Scripting
	status := p.scripts.Status()
	if !saved.Enabled {
		status = "Stopped · scripting is disabled"
	}
	state, _ := p.scripts.State()
	color := ui.TokenForegroundMuted
	if saved.Enabled && state == scriptsrv.Failed {
		color = ui.TokenError
	}
	return ui.Column(
		ui.Strong("Executor"),
		wrapped(th, "Runs pre/post-request scripts. Changes apply on Save; Restart applies to the saved settings."),
		ui.Row(
			ui.Paragraph(status).Size(th.Typography.Caption.Size).
				Style(ui.Spec{}.TextColor(color)).Grow(1),
			ui.Button("scripting-restart", ui.Text("Restart")).IconStart(icons.RefreshCw).
				Disabled(!saved.Enabled || state == scriptsrv.Starting).
				OnClick(func() { p.scripts.Restart(prefs.GetGlobalConfig().Spec.Scripting) }),
		).Gap(th.Spacing.S),
	).Gap(th.Spacing.S)
}

// fillLanguageServers gives the draft an entry for every language, so the
// form edits concrete values rather than implicit defaults.
func (p *Panel) fillLanguageServers() {
	ls := &p.draft.Spec.LanguageServers
	ls.Servers = langsrv.Effective(*ls)
}

func (p *Panel) languageServersForm(th *theme.Theme, g *domain.GlobalConfigSpec) ui.View {
	rows := []ui.View{
		ui.Paragraph("Language servers add completion, hover, and diagnostics to code editors. " +
			"A server starts when an editor of its language is first shown and stops when the last one closes. " +
			"Changes apply on Save; Restart applies to the saved settings.").
			Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
	}
	for i := range g.LanguageServers.Servers {
		s := &g.LanguageServers.Servers[i]
		l, ok := langsrv.ByID(s.Language)
		if !ok {
			continue
		}
		id := l.ID
		status := ""
		if p.lang != nil {
			status = p.lang.Status(*s)
		}
		actions := []ui.View{
			ui.Paragraph(status).Size(th.Typography.Caption.Size).
				Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)).Grow(1),
		}
		if _, ok := l.Installer(); ok && p.install != nil && p.lang != nil &&
			s.Command == l.Command && p.lang.Missing(*s) {
			label := "Install"
			if p.lang.Installing(id) {
				label = "Installing…"
			}
			actions = append(actions, ui.Button("lsp-install-"+id, ui.Text(label)).Primary().
				Disabled(p.lang.Installing(id)).OnClick(func() { p.install(id) }))
		}
		actions = append(actions, ui.Button("lsp-restart-"+id, ui.Text("Restart")).Ghost().HoverFill().OnClick(func() {
			if p.lang != nil {
				p.lang.Restart(id)
			}
		}))
		rows = append(rows, ui.Column(
			ui.HLine(th.Stroke.Thin, th.Border),
			ui.Row(
				ui.Strong(l.Name),
				ui.Caption(l.Note),
			).Gap(th.Spacing.S),
			ui.Form("settings-lsp-"+id,
				ui.FormSwitch("lsp-enabled-"+id, "Enabled", "Run this server for "+l.Name+" editors", s.Enabled, func(v bool) {
					s.Enabled = v
					p.mark()
				}),
				ui.FormText("lsp-command-"+id, "Command", "Default: "+l.Command, s.Command, func(v string) {
					s.Command = strings.TrimSpace(v)
					p.mark()
				}),
				ui.FormText("lsp-args-"+id, "Arguments", "Quote arguments with spaces", langsrv.JoinArgs(s.Args), func(v string) {
					s.Args = langsrv.SplitArgs(v)
					p.mark()
				}),
			),
			ui.Row(actions...).Gap(th.Spacing.S),
		).Gap(th.Spacing.S))
	}
	return ui.Column(rows...).Gap(th.Spacing.M).Padding(th.Spacing.M)
}

func yogaThemeOptions() []ui.SelectOption {
	names := theme.Names()
	opts := make([]ui.SelectOption, 0, len(names))
	for _, name := range names {
		opts = append(opts, ui.SelectOption{Label: name, Value: name})
	}
	return opts
}

func selectIndex(v string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}

// wrapped is muted body text that wraps to the width of the settings pane; a
// plain Text would run past its right edge.
func wrapped(th *theme.Theme, s string) ui.View {
	_ = th
	return ui.Row(
		ui.Paragraph(s).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)).Grow(1),
	).Grow(0)
}

// SetSecrets wires the secret key flows into the Security category.
func (p *Panel) SetSecrets(d secretui.Deps) { p.secrets = d }

// securityForm shows the state of the secret key and the actions that change
// it. None of it goes through the settings draft: the key is applied at once.
func (p *Panel) securityForm(th *theme.Theme) ui.View {
	m := p.secrets.Manager
	if m == nil {
		return ui.Column(
			ui.Strong("Secret key"),
			wrapped(th, "Secrets are not available in this session."),
		).Gap(th.Spacing.S).Padding(th.Spacing.M)
	}

	rows := []ui.View{
		ui.Strong("Secret key"),
		wrapped(th, "Environment values marked secret are encrypted with this key. Chapar never writes the key itself to disk."),
	}

	switch {
	case !m.Configured():
		rows = append(rows,
			ui.Text("No key is set up yet."),
			ui.Row(
				ui.Button("secret-setup", ui.Text("Set up a key")).Primary().IconStart(icons.Key).
					OnClick(func() { secretui.EnsureKey(p.secrets, nil) }),
			).Gap(th.Spacing.S),
		)
	default:
		where := m.StoreName()
		if m.Mode() == secret.ModePassphrase {
			where = "a passphrase, entered once per session"
		}
		state := "unlocked"
		if !m.Unlocked() {
			state = "locked"
		}
		rows = append(rows,
			wrapped(th, fmt.Sprintf("Key %s, kept in %s (%s).", m.KeyID(), where, state)),
			wrapped(th, fmt.Sprintf("Created %s.", m.CreatedAt().Local().Format("2 Jan 2006"))),
			ui.Row(
				ui.Button("secret-reveal", ui.Text("Show key")).IconStart(icons.Eye).
					Disabled(!m.Unlocked()).
					OnClick(p.revealKey),
				ui.Button("secret-import", ui.Text("Replace with pasted key")).IconStart(icons.Key).
					OnClick(func() { secretui.Unlock(p.secrets, nil) }),
				ui.Button("secret-forget", ui.Text("Remove from this machine")).IconStart(icons.Trash2).
					OnClick(p.forgetKey),
			).Gap(th.Spacing.S),
		)
	}

	return ui.Column(rows...).Gap(th.Spacing.S).Padding(th.Spacing.M)
}

func (p *Panel) revealKey() {
	host := p.secrets.Dialogs
	if host == nil || host() == nil {
		return
	}
	host().ShowAction("Show secret key",
		"The key will be displayed in this window. Make sure nobody is looking over your shoulder or recording your screen.",
		func() {
			key, err := p.secrets.Manager.Reveal()
			if err != nil {
				if p.secrets.Error != nil {
					p.secrets.Error(err)
				}
				return
			}
			secretui.ShowKeyOnce(p.secrets, key, "Your secret key",
				"Keep this somewhere safe. Anyone holding it can read your secret values.", nil)
		}, nil)
}

func (p *Panel) forgetKey() {
	host := p.secrets.Dialogs
	if host == nil || host() == nil {
		return
	}
	m := p.secrets.Manager
	host().ShowAction("Remove the secret key",
		fmt.Sprintf("The key is deleted from your %s. Secret values stay encrypted and unreadable until you paste the key back, so make sure you have a copy.", m.StoreName()),
		func() {
			if err := m.Forget(); err != nil && p.secrets.Error != nil {
				p.secrets.Error(err)
			}
		}, nil)
}
