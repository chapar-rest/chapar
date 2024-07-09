package component

import (
	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Auth struct {
	DropDown *widgets.DropDown

	auth domain.Auth

	TokenForm  *Form
	BasicForm  *Form
	APIKeyForm *Form

	onChange func(auth domain.Auth)
}

func NewAuth(auth domain.Auth, theme *chapartheme.Theme) *Auth {
	a := &Auth{
		auth: auth,
		DropDown: widgets.NewDropDown(
			theme,
			widgets.NewDropDownOption("None").WithValue(domain.AuthTypeNone),
			widgets.NewDropDownOption("Basic").WithValue(domain.AuthTypeBasic),
			widgets.NewDropDownOption("Token").WithValue(domain.AuthTypeToken),
			widgets.NewDropDownOption("API Key").WithValue(domain.AuthTypeAPIKey),
		),

		TokenForm: NewForm([]*Field{
			{Label: "Token", Value: ""},
		}),
		BasicForm: NewForm([]*Field{
			{Label: "Username", Value: ""},
			{Label: "Password", Value: ""},
		}),
		APIKeyForm: NewForm([]*Field{
			{Label: "Key", Value: ""},
			{Label: "Value", Value: ""},
		}),
	}

	a.DropDown.SetSelectedByValue(auth.Type)
	a.DropDown.MaxWidth = unit.Dp(150)

	if auth.BasicAuth != nil {
		a.BasicForm.SetValues(map[string]string{
			"Username": auth.BasicAuth.Username,
			"Password": auth.BasicAuth.Password,
		})
	}

	if auth.TokenAuth != nil {
		a.TokenForm.SetValues(map[string]string{
			"Token": auth.TokenAuth.Token,
		})
	}

	if auth.APIKeyAuth != nil {
		a.APIKeyForm.SetValues(map[string]string{
			"Key":   auth.APIKeyAuth.Key,
			"Value": auth.APIKeyAuth.Value,
		})
	}

	return a
}

func (a *Auth) SetOnChange(f func(auth domain.Auth)) {
	a.onChange = f

	a.DropDown.SetOnChanged(func(selected string) {
		a.auth.Type = selected
		a.onChange(a.auth)
	})

	a.TokenForm.SetOnChange(func(values map[string]string) {
		if a.auth.TokenAuth == nil {
			a.auth.TokenAuth = &domain.TokenAuth{}
		}

		a.auth.TokenAuth.Token = values["Token"]
		a.onChange(a.auth)
	})

	a.BasicForm.SetOnChange(func(values map[string]string) {
		if a.auth.BasicAuth == nil {
			a.auth.BasicAuth = &domain.BasicAuth{}
		}

		a.auth.BasicAuth.Username = values["Username"]
		a.auth.BasicAuth.Password = values["Password"]
		a.onChange(a.auth)
	})

	a.APIKeyForm.SetOnChange(func(values map[string]string) {
		if a.auth.APIKeyAuth == nil {
			a.auth.APIKeyAuth = &domain.APIKeyAuth{}
		}

		a.auth.APIKeyAuth.Key = values["Key"]
		a.auth.APIKeyAuth.Value = values["Value"]
		a.onChange(a.auth)
	})
}

func (a *Auth) SetAuth(auth domain.Auth) {
	a.auth = auth
	a.DropDown.SetSelectedByValue(auth.Type)

	if auth.BasicAuth != nil {
		a.BasicForm.SetValues(map[string]string{
			"Username": auth.BasicAuth.Username,
			"Password": auth.BasicAuth.Password,
		})
	}

	if auth.TokenAuth != nil {
		a.TokenForm.SetValues(map[string]string{
			"Token": auth.TokenAuth.Token,
		})
	}

	if auth.APIKeyAuth != nil {
		a.APIKeyForm.SetValues(map[string]string{
			"Key":   auth.APIKeyAuth.Key,
			"Value": auth.APIKeyAuth.Value,
		})
	}
}

func (a *Auth) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	inset := layout.Inset{Top: unit.Dp(15), Right: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Start,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return a.DropDown.Layout(gtx, theme)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				switch a.DropDown.GetSelected().Text {
				case "Token":
					return a.TokenForm.Layout(gtx, theme)
				case "Basic":
					return a.BasicForm.Layout(gtx, theme)
				case "API Key":
					return a.APIKeyForm.Layout(gtx, theme)
				default:
					return layout.Dimensions{}
				}
			}),
		)
	})
}
