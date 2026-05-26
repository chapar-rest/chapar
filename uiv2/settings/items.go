package settings

import (
	"cogentcore.org/core/core"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/theme"
)

func generalPanelItems() []panelItem {
	return []panelItem{
		headerItem("Request"),
		chooserItem(
			"HTTP request version",
			"Select the HTTP version to use for sending the request.",
			func(d *Data) *string { return &d.General.Request.HTTPVersion },
			[]core.ChooserItem{
				{Value: "http/1.1", Text: "HTTP/1.1"},
				{Value: "http/2", Text: "HTTP/2"},
			},
		),
		intItem(
			"Request timeout seconds",
			"Set how long a request needs to wait for the response before timeout. Zero means never.",
			func(d *Data) *int { return &d.General.Request.RequestTimeoutSec },
		),
		intItem(
			"Response size MB",
			"Maximum size of the response to download. Zero means unlimited.",
			func(d *Data) *int { return &d.General.Request.ResponseSizeMb },
		),
		boolItem(
			"Follow redirects",
			"When enabled, the HTTP client follows 3xx redirects. When disabled, the first response is returned.",
			func(d *Data) *bool { return &d.General.Request.FollowRedirects },
		),
		boolItem(
			"Validate TLS certificates",
			"When enabled, the HTTP client validates TLS certificates.",
			func(d *Data) *bool { return &d.General.Request.VaidateTLSCertificates },
		),
		headerItem("Headers"),
		boolItem(
			"Send no-cache header",
			"Add and send a no-cache header in HTTP requests.",
			func(d *Data) *bool { return &d.General.Headers.SendNoCacheHeader },
		),
		boolItem(
			"Send Chapar agent header",
			"Add and send the Chapar agent header in HTTP requests.",
			func(d *Data) *bool { return &d.General.Headers.SendChaparAgentHeader },
		),
		headerItem("User interface"),
		boolItem(
			"Use horizontal split for request and response",
			"If enabled, the request and response views are arranged top to bottom.",
			func(d *Data) *bool { return &d.General.UI.UseHorizontalSplit },
		),
	}
}

func scriptingPanelItems() []panelItem {
	dockerVisible := func(d *Data) bool { return d.Scripting.UseDocker }
	localVisible := func(d *Data) bool { return !d.Scripting.UseDocker }

	return []panelItem{
		boolItem(
			"Enable",
			"Enable scripting for pre and post request triggers.",
			func(d *Data) *bool { return &d.Scripting.Enabled },
		),
		chooserItem(
			"Language",
			"Select the scripting language you would like to use.",
			func(d *Data) *string { return &d.Scripting.Language },
			[]core.ChooserItem{{Value: "python", Text: "Python"}},
		),
		boolItem(
			"Use Docker",
			"Use Docker to run the scripting engine.",
			func(d *Data) *bool { return &d.Scripting.UseDocker },
		),
		textItemWhen(
			"Docker image",
			"The Docker image to use for the scripting engine.",
			func(d *Data) *string { return &d.Scripting.DockerImage },
			dockerVisible,
		),
		textItemWhen(
			"Executable path",
			"The absolute path to the executable binary.",
			func(d *Data) *string { return &d.Scripting.ExecutablePath },
			localVisible,
		),
		textItemWhen(
			"Server script path",
			"The absolute path to where Chapar can create server scripts.",
			func(d *Data) *string { return &d.Scripting.ServerScriptPath },
			localVisible,
		),
		intItem(
			"Port",
			"HTTP port that the server script listens on.",
			func(d *Data) *int { return &d.Scripting.Port },
		),
	}
}

func editorPanelItems() []panelItem {
	return []panelItem{
		headerItem("Font"),
		textItem(
			"Font family",
			"The font to use for the editor.",
			func(d *Data) *string { return &d.Editor.Font.FontFamily },
		),
		intItem(
			"Font size",
			"The font size to use for the editor.",
			func(d *Data) *int { return &d.Editor.Font.FontSize },
		),
		headerItem("Editing"),
		chooserItem(
			"Indentation",
			"Select the indentation type to use for the editor.",
			func(d *Data) *string { return &d.Editor.Editing.Indentation },
			[]core.ChooserItem{
				{Value: domain.IndentationSpaces, Text: "Spaces"},
				{Value: domain.IndentationTabs, Text: "Tabs"},
			},
		),
		intItem(
			"Tab width",
			"The width of the tab to use for the editor.",
			func(d *Data) *int { return &d.Editor.Editing.TabWidth },
		),
		boolItem(
			"Auto close brackets",
			"Automatically close brackets in the editor.",
			func(d *Data) *bool { return &d.Editor.Editing.AutoCloseBrackets },
		),
		boolItem(
			"Auto close quotes",
			"Automatically close quotes in the editor.",
			func(d *Data) *bool { return &d.Editor.Editing.AutoCloseQuotes },
		),
		boolItem(
			"Show line numbers",
			"Show line numbers in the editor.",
			func(d *Data) *bool { return &d.Editor.Editing.ShowLineNumbers },
		),
		boolItem(
			"Wrap lines",
			"Automatically wrap long lines in the editor.",
			func(d *Data) *bool { return &d.Editor.Editing.WrapLines },
		),
	}
}

func dataPanelItems() []panelItem {
	return []panelItem{
		textItem(
			"Workspace path",
			"The absolute path to the workspace folder.",
			func(d *Data) *string { return &d.Data.WorkspacePath },
		),
	}
}

func appearancePanelItems() []panelItem {
	names := theme.AllNames()
	items := make([]core.ChooserItem, len(names))
	for i, name := range names {
		items[i] = core.ChooserItem{Value: string(name), Text: name.Label()}
	}
	return []panelItem{
		chooserItem(
			"Theme",
			"Select the theme to use for the application.",
			func(d *Data) *string { return (*string)(&d.Appearance.Theme) },
			items,
		),
	}
}
