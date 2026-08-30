package container

import (
	"fmt"
	"strconv"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/google/uuid"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

// AuthState holds auth form field values for a container.
type AuthState struct {
	Type            string
	User, Pass      string
	Token           string
	Key, Val        string
	CollectionID    string
	AllowInherit    bool
}

// AuthForm builds the auth type selector and sub-forms.
func AuthForm(th *theme.Theme, id string, auth *domain.Auth, st *AuthState, catalog Catalog, markDirty func()) ui.View {
	opts := []ui.SelectOption{
		{Label: "None", Value: domain.AuthTypeNone},
		{Label: "Bearer", Value: domain.AuthTypeToken},
		{Label: "Basic", Value: domain.AuthTypeBasic},
		{Label: "API Key", Value: domain.AuthTypeAPIKey},
	}
	if st.AllowInherit {
		opts = append([]ui.SelectOption{{Label: "Inherit", Value: domain.AuthTypeInherit}}, opts...)
	}
	if auth.Type == "" {
		auth.Type = domain.AuthTypeNone
	}
	st.Type = auth.Type
	rows := []ui.View{
		ui.Select("auth-type-"+id, opts).Width(180).
			Selected(optionIndex(auth.Type, opts)).
			OnChange(func(v string) { auth.Type = v; st.Type = v; markDirty() }),
	}
	switch auth.Type {
	case domain.AuthTypeInherit:
		label := "Inherited: None (no auth configured in collection)"
		if st.CollectionID != "" && catalog != nil {
			if col := catalog.CollectionByID(st.CollectionID); col != nil {
				label = fmt.Sprintf("Inherited: %s", authTypeLabel(col.Spec.Auth.Type))
			}
		}
		rows = append(rows, ui.Text(label).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
	case domain.AuthTypeToken:
		rows = append(rows, ui.TextField("auth-token-"+id, st.Token).Placeholder("Token").
			OnChange(func(s string) { st.Token = s; markDirty() }).Grow(1))
	case domain.AuthTypeBasic:
		rows = append(rows,
			ui.TextField("auth-user-"+id, st.User).Placeholder("Username").
				OnChange(func(s string) { st.User = s; markDirty() }).Grow(1),
			ui.TextField("auth-pass-"+id, st.Pass).Placeholder("Password").Password(true).
				OnChange(func(s string) { st.Pass = s; markDirty() }).Grow(1),
		)
	case domain.AuthTypeAPIKey:
		rows = append(rows,
			ui.TextField("auth-key-"+id, st.Key).Placeholder("Header").
				OnChange(func(s string) { st.Key = s; markDirty() }).Grow(1),
			ui.TextField("auth-val-"+id, st.Val).Placeholder("Value").
				OnChange(func(s string) { st.Val = s; markDirty() }).Grow(1),
		)
	}
	return ui.Column(rows...).Gap(th.Spacing.S).Grow(1)
}

// FlushAuth writes auth state into the domain auth object.
func FlushAuth(auth *domain.Auth, st AuthState) {
	auth.Type = st.Type
	auth.BasicAuth = nil
	auth.TokenAuth = nil
	auth.APIKeyAuth = nil
	switch st.Type {
	case domain.AuthTypeBasic:
		auth.BasicAuth = &domain.BasicAuth{Username: st.User, Password: st.Pass}
	case domain.AuthTypeToken:
		auth.TokenAuth = &domain.TokenAuth{Token: st.Token}
	case domain.AuthTypeAPIKey:
		auth.APIKeyAuth = &domain.APIKeyAuth{Key: st.Key, Value: st.Val}
	}
}

func authTypeLabel(t string) string {
	switch t {
	case domain.AuthTypeBasic:
		return "Basic"
	case domain.AuthTypeToken:
		return "Bearer"
	case domain.AuthTypeAPIKey:
		return "API Key"
	case domain.AuthTypeNone, "":
		return "None"
	default:
		return t
	}
}

// LoadAuthState reads domain auth into AuthState.
func LoadAuthState(auth domain.Auth) AuthState {
	st := AuthState{Type: auth.Type}
	if auth.BasicAuth != nil {
		st.User = auth.BasicAuth.Username
		st.Pass = auth.BasicAuth.Password
	}
	if auth.TokenAuth != nil {
		st.Token = auth.TokenAuth.Token
	}
	if auth.APIKeyAuth != nil {
		st.Key = auth.APIKeyAuth.Key
		st.Val = auth.APIKeyAuth.Value
	}
	return st
}

// NewVariablesTable creates a variables extraction table.
func NewVariablesTable(id string, fromOptions []ui.SelectOption, markDirty func()) *ui.Table {
	t := ui.NewTable([]ui.TableColumn{
		{ID: "en", Label: "", Kind: ui.TableColCheckbox, Width: 36},
		{ID: "target", Label: "Target", Kind: ui.TableColEditable, Width: 0},
		{ID: "from", Label: "From", Kind: ui.TableColEditable, Width: 80},
		{ID: "status", Label: "Status", Kind: ui.TableColEditable, Width: 60},
		{ID: "path", Label: "Path/Key", Kind: ui.TableColEditable, Width: 0},
		{ID: "preview", Label: "Preview", Kind: ui.TableColText, Width: 0},
		{ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: icons.Trash2, Tooltip: "Delete"}})
	t.Actions[0].OnClick = func(rowID string) {
		t.RemoveRow(rowID)
		if markDirty != nil {
			markDirty()
		}
	}
	t.OnCellChange = func(_, _, _ string) {
		if markDirty != nil {
			markDirty()
		}
	}
	_ = id
	_ = fromOptions
	return t
}

func LoadVariables(t *ui.Table, items []domain.Variable) {
	rows := make([]ui.TableRow, 0, len(items))
	for _, v := range items {
		id := v.ID
		if id == "" {
			id = uuid.NewString()
		}
		status := ""
		if v.OnStatusCode != 0 {
			status = strconv.Itoa(v.OnStatusCode)
		}
		path := v.JsonPath
		if v.From != domain.VariableFromBody && v.SourceKey != "" {
			path = v.SourceKey
		}
		rows = append(rows, ui.TableRow{
			ID:       id,
			Selected: v.Enable,
			Cells: map[string]string{
				"target":  v.TargetEnvVariable,
				"from":    string(v.From),
				"status":  status,
				"path":    path,
				"preview": "",
			},
		})
	}
	t.SetRows(rows)
}

func DumpVariables(t *ui.Table) []domain.Variable {
	out := make([]domain.Variable, 0, len(t.Rows))
	for _, row := range t.Rows {
		status, _ := strconv.Atoi(row.Cells["status"])
		from := domain.VariableFrom(row.Cells["from"])
		if from == "" {
			from = domain.VariableFromBody
		}
		v := domain.Variable{
			ID:                row.ID,
			TargetEnvVariable: row.Cells["target"],
			From:              from,
			OnStatusCode:      status,
			Enable:            row.Selected,
		}
		if from == domain.VariableFromBody {
			v.JsonPath = row.Cells["path"]
		} else {
			v.SourceKey = row.Cells["path"]
		}
		out = append(out, v)
	}
	return out
}

func AddVariableRow(t *ui.Table, defaultStatus int, markDirty func()) {
	status := ""
	if defaultStatus != 0 {
		status = strconv.Itoa(defaultStatus)
	}
	t.AddRow(ui.TableRow{
		ID:       uuid.NewString(),
		Selected: true,
		Cells: map[string]string{
			"target": "",
			"from":   string(domain.VariableFromBody),
			"status": status,
			"path":   "",
		},
	})
	if markDirty != nil {
		markDirty()
	}
}

// UpdateVariablePreviews fills preview cells from the last response.
func UpdateVariablePreviews(t *ui.Table, vars []domain.Variable, res *egress.Response) {
	if t == nil || res == nil {
		return
	}
	code := res.StatusCode
	if code == 0 {
		code = res.StatueCode
	}
	for i, row := range t.Rows {
		preview := ""
		for _, v := range vars {
			if v.ID != row.ID {
				continue
			}
			if v.OnStatusCode != 0 && v.OnStatusCode != code {
				break
			}
			preview = extractVariablePreview(v, res)
		}
		t.Rows[i].Cells["preview"] = preview
	}
}

func extractVariablePreview(v domain.Variable, res *egress.Response) string {
	switch v.From {
	case domain.VariableFromBody:
		if res.JSON != "" {
			data, err := jsonpath.Get(res.JSON, v.JsonPath)
			if err == nil {
				if s, ok := data.(string); ok {
					return s
				}
				return fmt.Sprintf("%v", data)
			}
		}
	case domain.VariableFromHeader:
		if s, ok := res.ResponseHeaders[v.SourceKey]; ok {
			return s
		}
	case domain.VariableFromCookies:
		for _, c := range res.Cookies {
			if c.Name == v.SourceKey {
				return c.Value
			}
		}
	case domain.VariableFromMetaData:
		for _, item := range res.ResponseMetadata {
			if item.Key == v.SourceKey {
				return item.Value
			}
		}
	case domain.VariableFromTrailers:
		for _, item := range res.Trailers {
			if item.Key == v.SourceKey {
				return item.Value
			}
		}
	}
	return ""
}

func optionIndex(v string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}
