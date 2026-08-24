package settings

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const (
	catGeneral = iota
	catAppearance
	catScripting
	catEditor
	catData
)

// Panel is the settings dialog body. Open it with c.Dialogs().Show(DialogOpts{Body: panel.Layout, ...}).
type Panel struct {
	category int
	draft    domain.GlobalConfig
	dirty    bool
	pathWarn bool
	onTheme  func(string)
}

func New(onTheme func(string)) *Panel {
	return &Panel{onTheme: onTheme}
}

func (p *Panel) Prepare() {
	p.draft = prefs.GetGlobalConfig()
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
	p.dirty = true
	if p.onTheme != nil {
		p.onTheme(p.draft.Spec.General.Theme)
	}
}

func (p *Panel) Cancel() {
	if p.onTheme != nil {
		p.onTheme(prefs.GetGlobalConfig().Spec.General.Theme)
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
			ui.NavItem{ID: "data", Label: "Data", Icon: icons.Folder},
		).Selected(p.category).OnSelectItem(func(i int, _ string) { p.category = i }).Width(200),
		ui.VLine(th.Stroke.Thin, th.Border),
		ui.Scroll("settings-form", p.form(c)).Grow(1),
	).Align(ui.AlignStretch).Grow(1).Background(ui.TokenSurface)
}

func (p *Panel) mark() { p.dirty = true }

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
				if p.onTheme != nil {
					p.onTheme(v)
				}
			}),
			ui.FormSwitch("horizontalSplit", "Horizontal request/response split", "Stack request above response", g.General.UseHorizontalSplit, func(v bool) {
				g.General.UseHorizontalSplit = v
				p.mark()
			}),
		).Padding(th.Spacing.M)
	case catScripting:
		return p.scriptingForm(th, g)
	case catEditor:
		indentOpts := []ui.SelectOption{
			{Label: "Spaces", Value: domain.IndentationSpaces},
			{Label: "Tabs", Value: domain.IndentationTabs},
		}
		return ui.Form("settings-editor",
			ui.FormText("fontFamily", "Font family", "Editor font", g.Editor.FontFamily, func(v string) {
				g.Editor.FontFamily = v
				p.mark()
			}),
			ui.FormNumber("fontSize", "Font size", "Editor font size", float64(g.Editor.FontSize), 8, 32, 1, func(v float64) {
				g.Editor.FontSize = int(v)
				p.mark()
			}),
			ui.FormSelect("indentation", "Indentation", "Spaces or tabs", indentOpts, selectIndex(g.Editor.Indentation, indentOpts), func(v string) {
				g.Editor.Indentation = v
				p.mark()
			}),
			ui.FormNumber("tabWidth", "Tab width", "Width of a tab stop", float64(g.Editor.TabWidth), 1, 16, 1, func(v float64) {
				g.Editor.TabWidth = int(v)
				p.mark()
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
	return ui.Form("settings-scripting", items...).Padding(th.Spacing.M)
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
