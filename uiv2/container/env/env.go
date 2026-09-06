package env

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/container"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

type Container struct {
	env    *domain.Environment
	deps   container.Deps
	table  *ui.Table
	dirty  bool
	closed bool
}

func Open(env *domain.Environment, deps container.Deps) *Container {
	c := &Container{
		env:  container.CopyEnv(env),
		deps: deps,
	}
	c.table = container.NewKVTable("env-"+c.env.MetaData.ID, c.markDirty)
	container.LoadKV(c.table, c.env.Spec.Values)
	return c
}

func (c *Container) ID() string           { return c.env.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindEnv }
func (c *Container) Title() string        { return c.env.MetaData.Name }
func (c *Container) Dirty() bool          { return c.dirty }
func (c *Container) Send()                {}
func (c *Container) Close()               { c.closed = true }

func (c *Container) markDirty() {
	c.dirty = true
	c.deps.ReportDirty(true)
}

func (c *Container) Save() error {
	c.env.Spec.Values = container.DumpKV(c.table)
	if err := c.deps.Repo.UpdateEnvironment(c.env); err != nil {
		return err
	}
	if c.deps.Catalog != nil {
		c.deps.Catalog.ReplaceEnvironment(c.env)
	}
	c.dirty = false
	c.deps.ReportDirty(false)
	if c.deps.Report.Saved != nil {
		c.deps.Report.Saved()
	}
	c.deps.Toast("Environment saved")
	return nil
}

func (c *Container) Layout(ctx *ui.Ctx) ui.View {
	th := ctx.Theme()
	id := c.env.MetaData.ID
	return ui.Column(
		container.SimpleTitleRow(th, id, c.env.MetaData.Name,
			func(s string) { container.RenameEnvironment(c.deps, c.env, s) },
			ui.Button("env-add-"+id, ui.Text("Add")).IconStart(icons.Plus).OnClick(func() {
				container.AddKVRow(c.table, c.markDirty)
			}),
			ui.Button("env-save-"+id, ui.Text("Save")).Primary().IconStart(icons.Save).Hint("⌘S").
				Disabled(!c.dirty).
				OnClick(func() {
					if err := c.Save(); err != nil {
						c.deps.ShowError(err)
					}
				}),
		),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(c.table).Grow(1),
	).Grow(1)
}
