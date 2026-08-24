package collection

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/container"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type Container struct {
	col                         *domain.Collection
	deps                        container.Deps
	dirty                       bool
	notes                       *ui.Editor
	headers                     *ui.Table
	tabs                        []ui.TabModel
	active                      int
	authType, token, user, pass string
}

func Open(col *domain.Collection, deps container.Deps) *Container {
	c := &Container{
		col:  container.CopyCollection(col),
		deps: deps,
		tabs: []ui.TabModel{{Title: "Notes"}, {Title: "Headers"}, {Title: "Auth"}},
	}
	c.notes = ui.NewEditor([]byte(c.col.Spec.Notes), highlight.Noop{})
	c.headers = container.NewKVTable("col-hdr-"+c.col.MetaData.ID, c.markDirty)
	container.LoadKV(c.headers, c.col.Spec.Headers)
	c.authType = c.col.Spec.Auth.Type
	if c.col.Spec.Auth.TokenAuth != nil {
		c.token = c.col.Spec.Auth.TokenAuth.Token
	}
	if c.col.Spec.Auth.BasicAuth != nil {
		c.user = c.col.Spec.Auth.BasicAuth.Username
		c.pass = c.col.Spec.Auth.BasicAuth.Password
	}
	if c.authType == "" {
		c.authType = domain.AuthTypeNone
	}
	return c
}

func (c *Container) ID() string           { return c.col.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindCollection }
func (c *Container) Title() string        { return c.col.MetaData.Name }
func (c *Container) Dirty() bool          { return c.dirty || c.notes.Modified() }
func (c *Container) Send()                {}
func (c *Container) Close()               { c.notes.Close() }
func (c *Container) markDirty()           { c.dirty = true; c.deps.ReportDirty(true) }

func (c *Container) Save() error {
	c.col.Spec.Notes = string(c.notes.Bytes())
	c.col.Spec.Headers = container.DumpKV(c.headers)
	c.col.Spec.Auth.Type = c.authType
	switch c.authType {
	case domain.AuthTypeToken:
		c.col.Spec.Auth.TokenAuth = &domain.TokenAuth{Token: c.token}
	case domain.AuthTypeBasic:
		c.col.Spec.Auth.BasicAuth = &domain.BasicAuth{Username: c.user, Password: c.pass}
	}
	if err := c.deps.Repo.UpdateCollection(c.col); err != nil {
		return err
	}
	c.dirty = false
	c.notes.MarkSaved()
	c.deps.ReportDirty(false)
	if c.deps.Report.Saved != nil {
		c.deps.Report.Saved()
	}
	c.deps.Toast("Collection saved")
	return nil
}

func (c *Container) Layout(ctx *ui.Ctx) ui.View {
	th := ctx.Theme()
	id := c.col.MetaData.ID
	return ui.Column(
		ui.Row(
			ui.TextField("col-title-"+id, c.col.MetaData.Name).OnChange(func(s string) {
				c.col.MetaData.Name = s
				c.markDirty()
				c.deps.ReportTitle(s)
			}).Grow(1),
			ui.Button("col-save-"+id, ui.Text("Save")).Primary().IconStart(icons.Save).Disabled(!c.Dirty()).OnClick(func() {
				if err := c.Save(); err != nil {
					c.deps.ShowError(err)
				}
			}),
		).Gap(th.Spacing.S).Padding(th.Spacing.M),
		ui.Tabs("col-tabs-"+id, c.tabs).Selected(c.active).OnSelectItem(func(i int, _ string) { c.active = i }),
		ui.HLine(th.Stroke.Thin, th.Border),
		c.body(th),
	).Grow(1)
}

func (c *Container) body(th *theme.Theme) ui.View {
	id := c.col.MetaData.ID
	switch c.active {
	case 1:
		return ui.Column(
			ui.Button("col-hdr-add-"+id, ui.Text("Add")).OnClick(func() { container.AddKVRow(c.headers, c.markDirty) }),
			ui.ViewOf(c.headers).Grow(1),
		).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1)
	case 2:
		opts := []ui.SelectOption{
			{Label: "None", Value: domain.AuthTypeNone},
			{Label: "Bearer", Value: domain.AuthTypeToken},
			{Label: "Basic", Value: domain.AuthTypeBasic},
		}
		rows := []ui.View{
			ui.Select("col-auth-"+id, opts).Width(180).Selected(optionIndex(c.authType, opts)).
				OnChange(func(v string) { c.authType = v; c.markDirty() }),
		}
		if c.authType == domain.AuthTypeToken {
			rows = append(rows, ui.TextField("col-token-"+id, c.token).Placeholder("Token").
				OnChange(func(s string) { c.token = s; c.markDirty() }).Grow(1))
		}
		if c.authType == domain.AuthTypeBasic {
			rows = append(rows,
				ui.TextField("col-user-"+id, c.user).Placeholder("Username").OnChange(func(s string) { c.user = s; c.markDirty() }),
				ui.TextField("col-pass-"+id, c.pass).Placeholder("Password").Password(true).OnChange(func(s string) { c.pass = s; c.markDirty() }),
			)
		}
		return ui.Column(rows...).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1)
	default:
		return ui.ViewOf(c.notes).Grow(1)
	}
}

func optionIndex(v string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}
