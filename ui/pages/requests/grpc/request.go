package grpc

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/ui/converter"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/pages/requests/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Request struct {
	Tabs   *widgets.Tabs
	Prompt *widgets.Prompt

	ServerInfo *ServerInfo
	Body       *widgets.CodeEditor
	Metadata   *widgets.KeyValue
	Auth       *component.Auth
	Settings   *widgets.Settings
}

func NewRequest(req *domain.Request, theme *chapartheme.Theme, explorer *explorer.Explorer) *Request {
	visibilityFunc := func(values map[string]any) bool {
		return !values["insecure"].(bool)
	}

	certExt := []string{"pem", "crt"}

	r := &Request{
		Prompt: widgets.NewPrompt("Failed", "foo bar", widgets.ModalTypeErr),
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Server Info"},
			{Title: "Body"},
			{Title: "Auth"},
			{Title: "Meta Data"},
			{Title: "Settings"},
		}, nil),
		ServerInfo: NewServerInfo(req.Spec.GRPC.ServerInfo),
		Body:       widgets.NewCodeEditor(req.Spec.GRPC.Body, widgets.CodeLanguageJSON, theme),
		Metadata: widgets.NewKeyValue(
			converter.WidgetItemsFromKeyValue(req.Spec.GRPC.Metadata)...,
		),
		Auth: component.NewAuth(req.Spec.GRPC.Auth, theme),
		Settings: widgets.NewSettings([]*widgets.SettingItem{
			widgets.NewBoolItem("Plain Text", "insecure", "Insecure connection", req.Spec.GRPC.Settings.Insecure),
			widgets.NewFileItem(explorer, "Trusted Root certificate", "root_cert", "x509 pem trusted root certificate", req.Spec.GRPC.Settings.RootCertFile, certExt...).SetVisibleWhen(visibilityFunc),
			widgets.NewFileItem(explorer, "Client certificate", "client_public_key", "Public key", req.Spec.GRPC.Settings.ClientCertFile, certExt...).SetVisibleWhen(visibilityFunc),
			widgets.NewFileItem(explorer, "Client key", "client_private_key", "Private key", req.Spec.GRPC.Settings.ClientKeyFile, certExt...).SetVisibleWhen(visibilityFunc),
			widgets.NewTextItem("Overwrite server name for certificate verification", "nameOverride", "The value used to validate the common name in the server certificate.", req.Spec.GRPC.Settings.NameOverride).SetVisibleWhen(visibilityFunc),
			widgets.NewNumberItem("Timeout", "timeoutMilliseconds", "Timeout for the request in milliseconds", req.Spec.GRPC.Settings.TimeoutMilliseconds),
		}),
	}

	return r
}

func (r *Request) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Tabs.Layout(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return r.Prompt.Layout(gtx, theme)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				switch r.Tabs.SelectedTab().Title {
				case "Server Info":
					return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.ServerInfo.Layout(gtx, theme)
					})
				case "Body":
					return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.Body.Layout(gtx, theme, "JSON")
					})
				case "Meta Data":
					return layout.Inset{Top: unit.Dp(5), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.Metadata.WithAddLayout(gtx, "", "", theme)
					})
				case "Auth":
					return r.Auth.Layout(gtx, theme)
				case "Settings":
					return r.Settings.Layout(gtx, theme)
				default:
					return layout.Dimensions{}
				}
			}),
		)
	})
}
