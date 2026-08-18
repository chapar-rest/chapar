package settings

import (
	"strconv"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/ui"
)

type Dialog struct {
	Open     bool
	category int
	draft    domain.GlobalConfig
	dirty    bool
	pathWarn bool
	onTheme  func(string)
	onSaved  func()
	err      func(error)
	toast    func(string)
}

func New(onTheme func(string), errFn func(error), toast func(string), onSaved func()) *Dialog {
	return &Dialog{onTheme: onTheme, err: errFn, toast: toast, onSaved: onSaved}
}

func (d *Dialog) Show() {
	d.draft = prefs.GetGlobalConfig()
	d.dirty = false
	d.pathWarn = false
	d.category = 0
	d.Open = true
}

func (d *Dialog) Layout(c *ui.Ctx) ui.View {
	if !d.Open {
		return nil
	}
	th := c.Theme()
	cats := []string{"General", "Scripting", "Editor", "Data"}
	var catItems []ui.View
	for i, name := range cats {
		i, name := i, name
		btn := ui.Button("set-cat-"+name, ui.Text(name)).Subtle()
		if d.category == i {
			btn = btn.Primary()
		}
		btn.OnClick(func() { d.category = i })
		catItems = append(catItems, btn)
	}

	panel := ui.Column(
		ui.Row(
			ui.Title("Settings"),
			ui.Spacer(),
			ui.IconButton("set-close", "close").OnClick(func() { d.Open = false }),
		).Gap(th.Spacing.S),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Row(
			ui.Column(catItems...).Gap(th.Spacing.S).Width(160).Padding(th.Spacing.S),
			ui.VLine(th.Stroke.Thin, th.Border),
			ui.Scroll("set-form", d.form(c)).Grow(1),
		).Align(ui.AlignStretch).Grow(1),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Row(
			ui.Button("set-defaults", ui.Text("Load defaults")).OnClick(func() {
				d.draft = *domain.GetDefaultGlobalConfig()
				d.dirty = true
				if d.onTheme != nil {
					d.onTheme(d.draft.Spec.General.Theme)
				}
			}),
			ui.Spacer(),
			ui.Button("set-cancel", ui.Text("Cancel")).OnClick(func() {
				if d.onTheme != nil {
					d.onTheme(prefs.GetGlobalConfig().Spec.General.Theme)
				}
				d.Open = false
			}),
			ui.Button("set-save", ui.Text("Save")).Primary().Disabled(!d.dirty).OnClick(d.save),
		).Gap(th.Spacing.S),
	).Gap(th.Spacing.S).Padding(th.Spacing.L).Width(760).Height(520).Background(ui.TokenChrome)

	return ui.Center(panel).Grow(1).BackgroundColor(render.RGBA8(0, 0, 0, 140))
}

func (d *Dialog) save() {
	old := prefs.GetGlobalConfig()
	if err := prefs.UpdateGlobalConfig(d.draft); err != nil {
		if d.err != nil {
			d.err(err)
		}
		return
	}
	d.dirty = false
	d.Open = false
	if d.pathWarn || old.Spec.Data.WorkspacePath != d.draft.Spec.Data.WorkspacePath {
		if d.toast != nil {
			d.toast("Workspace path changed. Restart the application to apply.")
		}
	}
	if d.onSaved != nil {
		d.onSaved()
	}
}

func (d *Dialog) mark() { d.dirty = true }

func (d *Dialog) form(c *ui.Ctx) ui.View {
	th := c.Theme()
	g := &d.draft.Spec
	switch d.category {
	case 1:
		return ui.Column(
			ui.Checkbox("set-script-on", "Enable scripting").Check(g.Scripting.Enabled).OnToggle(func(v bool) { g.Scripting.Enabled = v; d.mark() }),
			selectRow("Language", "set-lang", g.Scripting.Language, []string{"python"}, func(v string) { g.Scripting.Language = v; d.mark() }),
			ui.Checkbox("set-docker", "Use Docker").Check(g.Scripting.UseDocker).OnToggle(func(v bool) { g.Scripting.UseDocker = v; d.mark() }),
			ui.TextField("set-dimg", g.Scripting.DockerImage).Placeholder("Docker image").OnChange(func(s string) { g.Scripting.DockerImage = s; d.mark() }),
			ui.TextField("set-exec", g.Scripting.ExecutablePath).Placeholder("Executable path").OnChange(func(s string) { g.Scripting.ExecutablePath = s; d.mark() }),
			ui.TextField("set-sscript", g.Scripting.ServerScriptPath).Placeholder("Server script path").OnChange(func(s string) { g.Scripting.ServerScriptPath = s; d.mark() }),
			numberField("Port", "set-port", g.Scripting.Port, func(n int) { g.Scripting.Port = n; d.mark() }),
		).Gap(th.Spacing.M).Padding(th.Spacing.M)
	case 2:
		return ui.Column(
			ui.TextField("set-font", g.Editor.FontFamily).Placeholder("Font family").OnChange(func(s string) { g.Editor.FontFamily = s; d.mark() }),
			numberField("Font size", "set-fsize", g.Editor.FontSize, func(n int) { g.Editor.FontSize = n; d.mark() }),
			selectRow("Indentation", "set-indent", g.Editor.Indentation, []string{domain.IndentationSpaces, domain.IndentationTabs}, func(v string) { g.Editor.Indentation = v; d.mark() }),
			numberField("Tab width", "set-tabw", g.Editor.TabWidth, func(n int) { g.Editor.TabWidth = n; d.mark() }),
			ui.Checkbox("set-brackets", "Auto close brackets").Check(g.Editor.AutoCloseBrackets).OnToggle(func(v bool) { g.Editor.AutoCloseBrackets = v; d.mark() }),
			ui.Checkbox("set-quotes", "Auto close quotes").Check(g.Editor.AutoCloseQuotes).OnToggle(func(v bool) { g.Editor.AutoCloseQuotes = v; d.mark() }),
			ui.Checkbox("set-lines", "Show line numbers").Check(g.Editor.ShowLineNumbers).OnToggle(func(v bool) { g.Editor.ShowLineNumbers = v; d.mark() }),
			ui.Checkbox("set-wrap", "Wrap lines").Check(g.Editor.WrapLines).OnToggle(func(v bool) { g.Editor.WrapLines = v; d.mark() }),
		).Gap(th.Spacing.M).Padding(th.Spacing.M)
	case 3:
		return ui.Column(
			ui.TextField("set-wspath", g.Data.WorkspacePath).Placeholder("Workspace path").OnChange(func(s string) {
				g.Data.WorkspacePath = s
				d.pathWarn = true
				d.mark()
			}),
		).Gap(th.Spacing.M).Padding(th.Spacing.M)
	default:
		themes := []string{"light", "github-light", "dark", "github-dark", "catppuccin-mocha", "catppuccin-frappe"}
		return ui.Column(
			selectRow("HTTP version", "set-httpv", g.General.HTTPVersion, []string{"http/1.1", "http/2"}, func(v string) { g.General.HTTPVersion = v; d.mark() }),
			numberField("Request timeout (sec)", "set-timeout", g.General.RequestTimeoutSec, func(n int) { g.General.RequestTimeoutSec = n; d.mark() }),
			numberField("Response size (MB)", "set-respsz", g.General.ResponseSizeMb, func(n int) { g.General.ResponseSizeMb = n; d.mark() }),
			ui.Checkbox("set-redir", "Follow redirects").Check(g.General.FollowRedirects).OnToggle(func(v bool) { g.General.FollowRedirects = v; d.mark() }),
			ui.Checkbox("set-tls", "Validate TLS certificates").Check(g.General.VaidateTLSCertificates).OnToggle(func(v bool) { g.General.VaidateTLSCertificates = v; d.mark() }),
			ui.Checkbox("set-nocache", "Send no-cache header").Check(g.General.SendNoCacheHeader).OnToggle(func(v bool) { g.General.SendNoCacheHeader = v; d.mark() }),
			ui.Checkbox("set-agent", "Send Chapar agent header").Check(g.General.SendChaparAgentHeader).OnToggle(func(v bool) { g.General.SendChaparAgentHeader = v; d.mark() }),
			selectRow("Theme", "set-theme", g.General.Theme, themes, func(v string) {
				g.General.Theme = v
				d.mark()
				if d.onTheme != nil {
					d.onTheme(v)
				}
			}),
			ui.Checkbox("set-hsplit", "Use horizontal split for request and response").Check(g.General.UseHorizontalSplit).OnToggle(func(v bool) { g.General.UseHorizontalSplit = v; d.mark() }),
		).Gap(th.Spacing.M).Padding(th.Spacing.M)
	}
}

func selectRow(label, id, value string, values []string, on func(string)) ui.View {
	opts := make([]ui.SelectOption, len(values))
	sel := 0
	for i, v := range values {
		opts[i] = ui.SelectOption{Label: v, Value: v}
		if v == value {
			sel = i
		}
	}
	return ui.Row(
		ui.Text(label).Width(220),
		ui.Select(id, opts).Width(240).Selected(sel).OnChange(on),
	).Gap(8)
}

func numberField(label, id string, n int, on func(int)) ui.View {
	return ui.Row(
		ui.Text(label).Width(220),
		ui.TextField(id, strconv.Itoa(n)).Width(120).OnChange(func(s string) {
			v, err := strconv.Atoi(s)
			if err == nil {
				on(v)
			}
		}),
	).Gap(8)
}
