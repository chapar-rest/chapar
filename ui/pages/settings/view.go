package settings

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	giox "gioui.org/x/component"

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

	u.load()

	u.treeView.OnNodeClick(func(tr *widgets.TreeNode) {
		u.selectedSettingIdentifier = tr.Identifier
		u.selectedSettingTitle = tr.Text
	})

	u.treeView.SetSelectedOnClick(true)

	return u
}

func (v *View) load() {
	// General settings
	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "General",
		Identifier: "general",
		IsSelected: true,
	})

	v.settings.Set("general", widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewHeaderItem("Request"),
		widgets.NewDropDownItem("HTTP request version", "httpRequestVersion", "Select the HTTP version to use for sending the request.", "http/1.1",
			widgets.NewDropDownOption("HTTP/1.1").WithIdentifier("http/1.1").WithValue("http/1.1"),
			widgets.NewDropDownOption("HTTP/2").WithIdentifier("http/2").WithValue("http/2"),
		),
		widgets.NewNumberItem("Request timeout", "requestTimeout", "Set how long a request need to wait for the response before timeout, zero means never", 0),
		widgets.NewNumberItem("Response Size", "responseSize", "Maximum size of the response to download. zero mean unlimited", 0),
		widgets.NewHeaderItem("Headers"),
		widgets.NewBoolItem("Send no-cache header", "SendNoCacheHeader", "Add and send no-cache header in http requests", false),
		widgets.NewBoolItem("Send Chapar agent header", "SendChaparAgentHeader", "Add and send Chapar agent header in http requests", true),
	}))

	// Scripting settings
	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "Scripting",
		Identifier: "scripting",
	})

	v.settings.Set("scripting", widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewBoolItem("Enable", "enable", "Enable scripting for pre and post request triggers", true),
		widgets.NewDropDownItem("Language", "language", "Select the scripting language you would like to have for scripting", "python",
			widgets.NewDropDownOption("Python").WithIdentifier("python").WithValue("python"),
			widgets.NewDropDownOption("Javascript").WithIdentifier("javascript").WithValue("javascript"),
		),
		widgets.NewTextItem("Executable path", "executable", "The absolute path to the executable binary", "").MinWidth(unit.Dp(400)).TextAlignment(text.Start),
		widgets.NewTextItem("Server script path", "serverScriptPath", "The absolute path to where Chapar can use to create server script", "").MinWidth(unit.Dp(400)).TextAlignment(text.Start),
		widgets.NewNumberItem("Port", "port", "Http port that server script is listening to", 2397),
	}))

	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "Editor",
		Identifier: "editor",
	})

	v.settings.Set("editor", widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewHeaderItem("Font"),
		widgets.NewTextItem("Font", "font", "The font to use for the editor", "JetBrains Mono").MinWidth(unit.Dp(400)).TextAlignment(text.Start),
		widgets.NewNumberItem("Font size", "fontSize", "The font size to use for the editor", 14),
		widgets.NewHeaderItem("Editing"),
		widgets.NewBoolItem("Show line numbers", "showLineNumbers", "Show line numbers in the editor", true),
		widgets.NewDropDownItem("Indentation", "indentation", "Select the indentation type to use for the editor", "spaces",
			widgets.NewDropDownOption("Spaces").WithIdentifier("spaces").WithValue("spaces"),
			widgets.NewDropDownOption("Tabs").WithIdentifier("tabs").WithValue("tabs"),
		),
		widgets.NewNumberItem("Tab size", "tabSize", "The size of the tab to use for the editor", 4),
		widgets.NewBoolItem("Auto close brackets", "autoCloseBrackets", "Automatically close brackets in the editor", true),
		widgets.NewBoolItem("Auto close quotes", "autoCloseQuotes", "Automatically close quotes in the editor", true),
	}))
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

	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.H5(theme.Material(), v.selectedSettingTitle).Layout(gtx)
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
