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
	Variables  *component.Variables

	PreRequest  *component.PrePostRequest
	PostRequest *component.PrePostRequest

	currentTab  string
	OnTabChange func(title string)
}

func NewRequest(req *domain.Request, theme *chapartheme.Theme, explorer *explorer.Explorer) *Request {
	visibilityFunc := func(values map[string]any) bool {
		return !values["insecure"].(bool)
	}

	certExt := []string{"pem", "crt"}

	postRequestDropDown := widgets.NewDropDown(
		theme,
		widgets.NewDropDownOption("From Response").WithValue(domain.PostRequestSetFromResponseBody),
		widgets.NewDropDownOption("From Metadata").WithValue(domain.PostRequestSetFromResponseMetaData),
		widgets.NewDropDownOption("From Trailers").WithValue(domain.PostRequestSetFromResponseTrailers),
	)

	r := &Request{
		Prompt: widgets.NewPrompt("Failed", "foo bar", widgets.ModalTypeErr),
		Tabs: widgets.NewTabs([]*widgets.Tab{
			{Title: "Server Info"},
			{Title: "Body"},
			{Title: "Auth"},
			{Title: "Meta Data"},
			{Title: "Variables"},
			{Title: "Settings"},
			{Title: "Pre Request"},
			{Title: "Post Request"},
		}, nil),
		ServerInfo: NewServerInfo(explorer, req.Spec.GRPC.ServerInfo),
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
		PreRequest: component.NewPrePostRequest([]component.Option{
			{Title: "None", Value: domain.PrePostTypeNone},
			{Title: "Trigger request", Value: domain.PrePostTypeTriggerRequest, Type: component.TypeTriggerRequest, Hint: "Trigger another request"},
			//	{Title: "Python", Value: domain.PostRequestTypePythonScript, Type: component.TypeScript, Hint: "Write your pre request python script here"},
			//	{Title: "Shell Script", Value: domain.PostRequestTypeSSHTunnel, Type: component.TypeScript, Hint: "Write your pre request shell script here"},
			//	{Title: "Kubectl tunnel", Value: domain.PostRequestTypeK8sTunnel, Type: component.TypeScript, Hint: "Run kubectl port-forward command"},
			//	{Title: "SSH tunnel", Value: domain.PostRequestTypeSSHTunnel, Type: component.TypeScript, Hint: "Run ssh command"},
		}, nil, theme),
		PostRequest: component.NewPrePostRequest([]component.Option{
			{Title: "None", Value: domain.PrePostTypeNone},
			{Title: "Set Environment Variable", Value: domain.PrePostTypeSetEnv, Type: component.TypeSetEnv, Hint: "Set environment variable"},
			//	{Title: "Python", Value: domain.PostRequestTypePythonScript, Type: component.TypeScript, Hint: "Write your post request python script here"},
			//	{Title: "Shell Script", Value: domain.PostRequestTypeShellScript, Type: component.TypeScript, Hint: "Write your post request shell script here"},
		}, postRequestDropDown, theme),
		Variables: component.NewVariables(theme, domain.RequestTypeGRPC),
	}

	if req.Spec.GRPC.PreRequest != (domain.PreRequest{}) {
		r.PreRequest.SetSelectedDropDown(req.Spec.GRPC.PreRequest.Type)
	}

	if req.Spec.GRPC.PostRequest != (domain.PostRequest{}) {
		r.PostRequest.SetSelectedDropDown(req.Spec.GRPC.PostRequest.Type)

		if req.Spec.GRPC.PostRequest.PostRequestSet != (domain.PostRequestSet{}) {
			r.PostRequest.SetPostRequestSetValues(req.Spec.GRPC.PostRequest.PostRequestSet)
		}
	}

	if req.Spec.GRPC.Variables != nil {
		r.Variables.SetValues(req.Spec.GRPC.Variables)
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
				if r.Tabs.SelectedTab().Title != r.currentTab {
					r.currentTab = r.Tabs.SelectedTab().Title
					if r.OnTabChange != nil {
						r.OnTabChange(r.currentTab)
					}
				}
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
				case "Pre Request":
					return r.PreRequest.Layout(gtx, theme)
				case "Post Request":
					return r.PostRequest.Layout(gtx, theme)
				case "Variables":
					return r.Variables.Layout(gtx, "Variables", "", theme)
				default:
					return layout.Dimensions{}
				}
			}),
		)
	})
}
