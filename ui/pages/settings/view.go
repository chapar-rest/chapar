package settings

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	giox "gioui.org/x/component"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
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
	treeViewSetupDone bool

	onChange       func(values map[string]any)
	onSave         func()
	onCancel       func()
	onLoadDefaults func()
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

	u.Load(prefs.GetGlobalConfig())

	u.treeView.OnNodeClick(func(tr *widgets.TreeNode) {
		u.selectedSettingIdentifier = tr.Identifier
		u.selectedSettingTitle = tr.Text
	})

	u.treeView.SetSelectedOnClick(true)

	return u
}

func (v *View) ShowError(err error) {
	v.modal = widgets.NewMessageModal("Error", err.Error(), widgets.MessageModalTypeErr, func(_ string) {
		v.modal.Hide()
	}, widgets.ModalOption{Text: "Ok"})
	v.modal.Show()
}

func (v *View) ShowInfo(title, message string) {
	v.modal = widgets.NewMessageModal(title, message, widgets.MessageModalTypeInfo, func(_ string) {
		v.modal.Hide()
	}, widgets.ModalOption{Text: "Ok"})
	v.modal.Show()
}

func (v *View) Refresh() {
	v.window.Invalidate()
}

func (v *View) SetOnSave(f func()) {
	v.onSave = f
}

func (v *View) SetOnCancel(f func()) {
	v.onCancel = f
}

func (v *View) SetOnLoadDefaults(f func()) {
	v.onLoadDefaults = f
}

func (v *View) SetOnChange(f func(values map[string]any)) {
	v.onChange = f
}

func (v *View) callOnChange(values map[string]any) {
	if v.onChange != nil {
		v.onChange(values)
	}
}

func (v *View) Load(config domain.GlobalConfig) {
	if !v.treeViewSetupDone {
		v.treeView.SetNodes([]*widgets.TreeNode{
			{
				Text:       "General",
				Identifier: "general",
				IsSelected: true,
			},
			{
				Text:       "Scripting",
				Identifier: "scripting",
			},
			{

				Text:       "Editor",
				Identifier: "editor",
			},
			{
				Text:       "Data",
				Identifier: "data",
			},
		})

		v.treeViewSetupDone = true
	}

	// General settings
	generalSettings := widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewHeaderItem("Request"),
		widgets.NewDropDownItem("HTTP request version", "httpVersion", "Select the HTTP version to use for sending the request.", config.Spec.General.HTTPVersion,
			widgets.NewDropDownOption("HTTP/1.1").WithIdentifier("http/1.1").WithValue("http/1.1"),
			widgets.NewDropDownOption("HTTP/2").WithIdentifier("http/2").WithValue("http/2"),
		),
		widgets.NewNumberItem("Request timeout Seconds", "requestTimeoutSec", "Set how long a request need to wait for the response before timeout, zero means never", config.Spec.General.RequestTimeoutSec),
		widgets.NewNumberItem("Response Size MB", "responseSizeMb", "Maximum size of the response to download. zero mean unlimited", config.Spec.General.ResponseSizeMb),
		widgets.NewHeaderItem("Headers"),
		widgets.NewBoolItem("Send no-cache header", "sendNoCacheHeader", "Add and send no-cache header in http requests", config.Spec.General.SendNoCacheHeader),
		widgets.NewBoolItem("Send Chapar agent header", "sendChaparAgentHeader", "Add and send Chapar agent header in http requests", config.Spec.General.SendChaparAgentHeader),
	})
	generalSettings.SetOnChange(v.callOnChange)
	v.settings.Set("general", generalSettings)

	dockerVisibility := func(values map[string]any) bool {
		return values["useDocker"].(bool)
	}

	localEngineVisibility := func(values map[string]any) bool {
		return !values["useDocker"].(bool)
	}

	scriptingSettings := widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewBoolItem("Enable", "enable", "Enable scripting for pre and post request triggers", config.Spec.Scripting.Enabled),
		widgets.NewDropDownItem("Language", "language", "Select the scripting language you would like to have for scripting", config.Spec.Scripting.Language,
			widgets.NewDropDownOption("Python").WithIdentifier("python").WithValue("python"),
			// widgets.NewDropDownOption("Javascript").WithIdentifier("javascript").WithValue("javascript"),
		),
		widgets.NewBoolItem("Use Docker", "useDocker", "Use docker to run the scripting engine (right now docker is only working option)", config.Spec.Scripting.UseDocker),
		widgets.NewTextItem("Docker image", "dockerImage", "The docker image to use for the scripting engine", config.Spec.Scripting.DockerImage).MinWidth(unit.Dp(400)).TextAlignment(text.Start).SetVisibleWhen(dockerVisibility),
		widgets.NewTextItem("Executable path", "executable", "The absolute path to the executable binary", config.Spec.Scripting.ExecutablePath).MinWidth(unit.Dp(400)).TextAlignment(text.Start).SetVisibleWhen(localEngineVisibility),
		widgets.NewTextItem("Server script path", "serverScriptPath", "The absolute path to where Chapar can use to create server script", config.Spec.Scripting.ServerScriptPath).MinWidth(unit.Dp(400)).TextAlignment(text.Start).SetVisibleWhen(localEngineVisibility),
		widgets.NewNumberItem("Port", "port", "Http port that server script is listening to", config.Spec.Scripting.Port),
	})
	scriptingSettings.SetOnChange(v.callOnChange)
	v.settings.Set("scripting", scriptingSettings)

	editorSettings := widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewHeaderItem("Font"),
		widgets.NewTextItem("Font Family", "fontFamily", "The font to use for the editor", config.Spec.Editor.FontFamily).MinWidth(unit.Dp(400)).TextAlignment(text.Start),
		widgets.NewNumberItem("Font size", "fontSize", "The font size to use for the editor", config.Spec.Editor.FontSize),
		widgets.NewDropDownItem("Theme", "theme", "Select editor theme", config.Spec.Editor.Theme,
			getThemes()...,
		),
		widgets.NewHeaderItem("Editing"),
		widgets.NewDropDownItem("Indentation", "indentation", "Select the indentation type to use for the editor", config.Spec.Editor.Indentation,
			widgets.NewDropDownOption("Spaces").WithIdentifier("spaces").WithValue("spaces"),
			widgets.NewDropDownOption("Tabs").WithIdentifier("tabs").WithValue("tabs"),
		),
		widgets.NewNumberItem("Tab width", "tabWidth", "The width of the tab to use for the editor", config.Spec.Editor.TabWidth),
		widgets.NewBoolItem("Auto close brackets", "autoCloseBrackets", "Automatically close brackets in the editor", config.Spec.Editor.AutoCloseBrackets),
		widgets.NewBoolItem("Auto close quotes", "autoCloseQuotes", "Automatically close quotes in the editor", config.Spec.Editor.AutoCloseQuotes),
	})
	editorSettings.SetOnChange(v.callOnChange)
	v.settings.Set("editor", editorSettings)

	dataSettings := widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewTextItem("Workspace path", "workspacePath", "The absolute path to the workspace folder", config.Spec.Data.WorkspacePath).MinWidth(unit.Dp(400)).TextAlignment(text.Start),
	})
	dataSettings.SetOnChange(v.callOnChange)
	v.settings.Set("data", dataSettings)
}

func getThemes() []*widgets.DropDownOption {
	names := styles.Names()
	out := make([]*widgets.DropDownOption, 0, len(names))
	for _, name := range names {
		out = append(out, widgets.NewDropDownOption(name).WithIdentifier(name).WithValue(name))
	}
	return out
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
	bt := widgets.Button(theme.Material(), b.Clickable, b.Icon, widgets.IconPositionStart, b.Text)
	if b.IsDataChanged {
		bt.Color = theme.ButtonTextColor
		bt.Background = theme.ActionButtonBgColor
	} else {
		bt.Color = widgets.Disabled(theme.Palette.ContrastFg)
	}

	return bt.Layout(gtx, theme)
}

func (v *View) layoutActions(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if v.SaveButton.Clicked(gtx) {
		if v.onSave != nil {
			v.onSave()
		}
	}

	if v.onSave != nil {
		keys.OnSaveCommand(gtx, v, func() {
			v.onSave()
		})
	}

	if v.CancelButton.Clicked(gtx) {
		if v.onCancel != nil {
			v.onCancel()
		}
	}

	if v.LoadDefaultButton.Clicked(gtx) {
		if v.onLoadDefaults != nil {
			v.onLoadDefaults()
		}
	}

	return layout.Inset{Bottom: unit.Dp(15), Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := widgets.Button(theme.Material(), &v.LoadDefaultButton, nil, widgets.IconPositionStart, "Load Defaults")
				return btn.Layout(gtx, theme)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := widgets.Button(theme.Material(), &v.CancelButton, nil, widgets.IconPositionStart, "Cancel")
				return btn.Layout(gtx, theme)
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
