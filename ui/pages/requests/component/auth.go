package component

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"

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

	collectionAuth *domain.Auth // Auth from collection for inheritance

	onChange func(auth domain.Auth)
}

func NewAuth(auth domain.Auth, theme *chapartheme.Theme) *Auth {
	a := &Auth{
		auth: auth,
		DropDown: widgets.NewDropDown(
			widgets.NewDropDownOption("None").WithValue(domain.AuthTypeNone),
			widgets.NewDropDownOption("Basic").WithValue(domain.AuthTypeBasic),
			widgets.NewDropDownOption("Token").WithValue(domain.AuthTypeToken),
			widgets.NewDropDownOption("API Key").WithValue(domain.AuthTypeAPIKey),
			widgets.NewDropDownOption("Inherit from Collection").WithValue(domain.AuthTypeInherit),
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

// SetCollectionAuth sets the auth configuration from the collection for inheritance
func (a *Auth) SetCollectionAuth(collectionAuth *domain.Auth) {
	a.collectionAuth = collectionAuth
}

func (a *Auth) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if a.DropDown.Changed() {
		a.auth.Type = a.DropDown.GetSelected().Value
		if a.onChange != nil {
			a.onChange(a.auth)
		}
	}

	inset := layout.Inset{Top: unit.Dp(15), Right: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		selectedAuthType := a.DropDown.GetSelected().Value
		isInherited := selectedAuthType == domain.AuthTypeInherit

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
				if isInherited {
					// Show inherited auth info
					if a.collectionAuth != nil && a.collectionAuth.Type != "" && a.collectionAuth.Type != domain.AuthTypeNone {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								inheritedType := a.getAuthTypeDisplay(a.collectionAuth.Type)
								label := material.Label(theme.Material(), unit.Sp(14), fmt.Sprintf("Inherited: %s", inheritedType))
								label.Color = theme.TextColor
								return label.Layout(gtx)
							}),
						)
					} else {
						label := material.Label(theme.Material(), unit.Sp(14), "Inherited: None (no auth configured in collection)")
						label.Color = theme.TextColor
						return label.Layout(gtx)
					}
				}

				// Show auth forms for non-inherited types
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

func (a *Auth) getAuthTypeDisplay(authType string) string {
	switch authType {
	case domain.AuthTypeBasic:
		return "Basic Auth"
	case domain.AuthTypeToken:
		return "Token Auth"
	case domain.AuthTypeAPIKey:
		return "API Key Auth"
	case domain.AuthTypeNone:
		return "None"
	default:
		return authType
	}
}
