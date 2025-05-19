package settings

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	giox "gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type View struct {
	window *app.Window

	treeViewSearchBox *widgets.TextField
	treeView          *widgets.TreeView

	split widgets.SplitView

	// modal is used to show error and messages to the user
	modal *widgets.MessageModal

	settings *safemap.Map[*widgets.Settings]

	selectedSettingIdentifier string
	selectedSettingTitle      string

	SaveButton        widget.Clickable
	CancelButton      widget.Clickable
	LoadDefaultButton widget.Clickable
	IsDataChanged     bool

	onChange func(values map[string]any)
}

func NewView(window *app.Window, theme *chapartheme.Theme) *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	search.SetBorderColor(theme.BorderColor)

	u := &View{
		window:            window,
		treeViewSearchBox: search,
		treeView:          widgets.NewTreeView([]*widgets.TreeNode{}),
		split: widgets.SplitView{
			Resize: giox.Resize{
				Ratio: 0.19,
			},
			BarWidth: unit.Dp(2),
		},
		settings: safemap.New[*widgets.Settings](),
	}

	u.Load()

	u.treeView.OnNodeClick(func(tr *widgets.TreeNode) {
		u.selectedSettingIdentifier = tr.Identifier
		u.selectedSettingTitle = tr.Text
	})

	u.treeView.SetSelectedOnClick(true)

	return u
}

func (v *View) SetOnChange(f func(values map[string]any)) {
	v.onChange = f
}

func (v *View) Load() {
	globalSettings := prefs.GetGlobalConfig().Spec

	// General settings
	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "General",
		Identifier: "general",
		IsSelected: true,
	})

	// General settings
	generalSettings := widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewHeaderItem("Request"),
		widgets.NewDropDownItem("HTTP request version", "httpRequestVersion", "Select the HTTP version to use for sending the request.", globalSettings.General.HTTPVersion,
			widgets.NewDropDownOption("HTTP/1.1").WithIdentifier("http/1.1").WithValue("http/1.1"),
			widgets.NewDropDownOption("HTTP/2").WithIdentifier("http/2").WithValue("http/2"),
		),
		widgets.NewNumberItem("Request timeout Seconds", "requestTimeoutSec", "Set how long a request need to wait for the response before timeout, zero means never", globalSettings.General.RequestTimeoutSec),
		widgets.NewNumberItem("Response Size MB", "responseSizeMb", "Maximum size of the response to download. zero mean unlimited", globalSettings.General.ResponseSizeMb),
		widgets.NewHeaderItem("Headers"),
		widgets.NewBoolItem("Send no-cache header", "SendNoCacheHeader", "Add and send no-cache header in http requests", globalSettings.General.SendNoCacheHeader),
		widgets.NewBoolItem("Send Chapar agent header", "SendChaparAgentHeader", "Add and send Chapar agent header in http requests", globalSettings.General.SendChaparAgentHeader),
	})
	generalSettings.SetOnChange(v.onChange)
	v.settings.Set("general", generalSettings)

	// Scripting settings
	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "Scripting",
		Identifier: "scripting",
	})

	dockerVisibility := func(values map[string]any) bool {
		return values["useDocker"].(bool)
	}

	localEngineVisibility := func(values map[string]any) bool {
		return !values["useDocker"].(bool)
	}

	scriptingSettings := widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewBoolItem("Enable", "enable", "Enable scripting for pre and post request triggers", globalSettings.Scripting.Enabled),
		widgets.NewDropDownItem("Language", "language", "Select the scripting language you would like to have for scripting", globalSettings.Scripting.Language,
			widgets.NewDropDownOption("Python").WithIdentifier("python").WithValue("python"),
			widgets.NewDropDownOption("Javascript").WithIdentifier("javascript").WithValue("javascript"),
		),
		widgets.NewBoolItem("Use Docker", "useDocker", "Use docker to run the scripting engine", globalSettings.Scripting.UseDocker),
		widgets.NewTextItem("Docker image", "dockerImage", "The docker image to use for the scripting engine", globalSettings.Scripting.DockerImage).MinWidth(unit.Dp(400)).TextAlignment(text.Start).SetVisibleWhen(dockerVisibility),
		widgets.NewTextItem("Executable path", "executable", "The absolute path to the executable binary", globalSettings.Scripting.ExecutablePath).MinWidth(unit.Dp(400)).TextAlignment(text.Start).SetVisibleWhen(localEngineVisibility),
		widgets.NewTextItem("Server script path", "serverScriptPath", "The absolute path to where Chapar can use to create server script", globalSettings.Scripting.ServerScriptPath).MinWidth(unit.Dp(400)).TextAlignment(text.Start).SetVisibleWhen(localEngineVisibility),
		widgets.NewNumberItem("Port", "port", "Http port that server script is listening to", globalSettings.Scripting.Port),
	})
	scriptingSettings.SetOnChange(v.onChange)
	v.settings.Set("scripting", scriptingSettings)

	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "Editor",
		Identifier: "editor",
	})

	editorSettings := widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewHeaderItem("Font"),
		widgets.NewTextItem("Font Family", "fontFamily", "The font to use for the editor", globalSettings.Editor.FontFamily).MinWidth(unit.Dp(400)).TextAlignment(text.Start),
		widgets.NewNumberItem("Font size", "fontSize", "The font size to use for the editor", globalSettings.Editor.FontSize),
		widgets.NewHeaderItem("Editing"),
		widgets.NewDropDownItem("Indentation", "indentation", "Select the indentation type to use for the editor", globalSettings.Editor.Indentation,
			widgets.NewDropDownOption("Spaces").WithIdentifier("spaces").WithValue("spaces"),
			widgets.NewDropDownOption("Tabs").WithIdentifier("tabs").WithValue("tabs"),
		),
		widgets.NewNumberItem("Tab width", "tabWidth", "The width of the tab to use for the editor", globalSettings.Editor.TabWidth),
		widgets.NewBoolItem("Auto close brackets", "autoCloseBrackets", "Automatically close brackets in the editor", globalSettings.Editor.AutoCloseBrackets),
		widgets.NewBoolItem("Auto close quotes", "autoCloseQuotes", "Automatically close quotes in the editor", globalSettings.Editor.AutoCloseQuotes),
	})
	editorSettings.SetOnChange(v.onChange)
	v.settings.Set("editor", editorSettings)

	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "Data",
		Identifier: "data",
	})

	dataSettings := widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewTextItem("Workspace path", "workspacePath", "The absolute path to the workspace folder", globalSettings.Data.WorkspacePath).MinWidth(unit.Dp(400)).TextAlignment(text.Start),
	})
	dataSettings.SetOnChange(v.onChange)
	v.settings.Set("data", dataSettings)
}

func (v *View) showError(err error) {
	v.modal = widgets.NewMessageModal("Error", err.Error(), widgets.MessageModalTypeErr, func(_ string) {
		v.modal.Hide()
	}, widgets.ModalOption{Text: "Ok"})
	v.modal.Show()
}

func (v *View) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return v.split.Layout(gtx, theme,
		func(gtx layout.Context) layout.Dimensions {
			return v.settingsList(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return v.settingDetail(gtx, theme)
		},
	)
}

func (v *View) settingsList(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeViewSearchBox.Layout(gtx, theme)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeView.Layout(gtx, theme)
				})
			}),
		)
	})
}

func (v *View) settingDetail(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	v.modal.Layout(gtx, theme)

	if v.selectedSettingIdentifier == "" {
		v.selectedSettingIdentifier = "general"
		v.selectedSettingTitle = "General"
	}

	setting, ok := v.settings.Get(v.selectedSettingIdentifier)
	if !ok {
		return layout.Dimensions{}
	}

	setting.SetOnChange(v.onChange)

	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Rigid(material.H5(theme.Material(), v.selectedSettingTitle).Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return v.layoutActions(gtx, theme)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			widgets.DrawLineFlex(theme.SeparatorColor, unit.Dp(1), unit.Dp(gtx.Constraints.Max.X)),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return setting.Layout(gtx, theme)
				})
			}),
		)
	})
}

type button struct {
	Clickable *widget.Clickable
	Text      string
	Icon      *widget.Icon

	IsDataChanged bool
}

func (b *button) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	border := widget.Border{
		Color:        theme.Palette.ContrastFg,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	bt := widgets.Button(theme.Material(), b.Clickable, b.Icon, widgets.IconPositionStart, b.Text)
	if b.IsDataChanged {
		bt.Color = theme.Palette.ContrastFg
		border.Width = unit.Dp(1)
		border.Color = theme.Palette.ContrastFg
	} else {
		bt.Color = widgets.Disabled(theme.Palette.ContrastFg)
		border.Color = widgets.Disabled(theme.Palette.ContrastFg)
		border.Width = 0
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return bt.Layout(gtx, theme)
	})
}

func (v *View) layoutActions(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(15), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := &button{
					Clickable:     &v.LoadDefaultButton,
					Text:          "Load Defaults",
					IsDataChanged: v.IsDataChanged,
					Icon:          widgets.RefreshIcon,
				}
				return b.Layout(gtx, theme)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := &button{
					Clickable:     &v.CancelButton,
					Text:          "Cancel",
					IsDataChanged: v.IsDataChanged,
					Icon:          widgets.CloseIcon,
				}
				return b.Layout(gtx, theme)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := &button{
					Clickable:     &v.SaveButton,
					Text:          "Save",
					IsDataChanged: v.IsDataChanged,
					Icon:          widgets.SaveIcon,
				}
				return b.Layout(gtx, theme)
			}),
		)
	})
}
