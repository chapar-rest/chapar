package settings

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
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
	})

	return u
}

func (v *View) load() {
	// General settings
	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "General",
		Identifier: "general",
	})

	v.settings.Set("general", widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewBoolItem("Plain Text", "insecure", "Insecure connection", true),
		widgets.NewTextItem("Overwrite server name for certificate verification", "nameOverride", "The value used to validate the common name in the server certificate.", ""),
		widgets.NewNumberItem("Timeout", "timeoutMilliseconds", "Timeout for the request in milliseconds", 10),
	}))

	// Scripting settings
	v.treeView.AddNode(&widgets.TreeNode{
		Text:       "Scripting",
		Identifier: "scripting",
	})

	v.settings.Set("scripting", widgets.NewSettings([]*widgets.SettingItem{
		widgets.NewBoolItem("Enable", "enable", "Enable scripting for pre and post request triggers", true),
		widgets.NewTextItem("Executable path", "executable", "The absolute path to the executable binary", "").MinWidth(unit.Dp(400)).TextAlignment(text.Start),
		widgets.NewTextItem("Server script path", "serverScriptPath", "The absolute path to where Chapar can use to create server script", "").MinWidth(unit.Dp(400)).TextAlignment(text.Start),
		widgets.NewNumberItem("Port", "port", "Http port that server script is listening to", 2397),
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
	}

	setting, ok := v.settings.Get(v.selectedSettingIdentifier)
	if !ok {
		return layout.Dimensions{}
	}

	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return setting.Layout(gtx, theme)
	})
}
