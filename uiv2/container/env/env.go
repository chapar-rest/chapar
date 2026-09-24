package env

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/container"
	reqicons "github.com/chapar-rest/chapar/uiv2/icons"
	"github.com/chapar-rest/chapar/uiv2/secretui"
	"github.com/chapar-rest/chapar/uiv2/vars"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type Container struct {
	env   *domain.Environment
	deps  container.Deps
	table *ui.Table
	query string
	// revealed holds the rows whose secret value the user asked to see. It is
	// dropped when the container closes.
	revealed map[string]bool
	// locked holds the rows that arrived still encrypted, because the secret
	// key is missing.
	locked map[string]bool
	dirty  bool
	closed bool
}

func Open(env *domain.Environment, deps container.Deps) *Container {
	c := &Container{
		env:      container.CopyEnv(env),
		deps:     deps,
		revealed: map[string]bool{},
		locked:   map[string]bool{},
	}
	for _, kv := range c.env.Spec.Values {
		if kv.Locked {
			c.locked[kv.ID] = true
		}
	}
	c.table = container.NewSecretKVTable("env-"+c.env.MetaData.ID, c.markDirty, container.SecretsUI{
		Revealed:     func(rowID string) bool { return c.revealed[rowID] },
		ToggleReveal: c.toggleReveal,
		OnMarkSecret: c.ensureKey,
		Locked:       func(rowID string) bool { return c.locked[rowID] },
	})
	container.LoadKV(c.table, c.env.Spec.Values)
	// An environment value may stand on another value of the same environment,
	// so the table completes its own keys rather than the active environment's.
	container.AssistKV(c.table, container.VarSource(container.Deps{}, func() []vars.Entry {
		return container.EnvEntries(container.DumpKV(c.table))
	}))
	return c
}

func (c *Container) ID() string           { return c.env.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindEnv }
func (c *Container) Title() string        { return c.env.MetaData.Name }
func (c *Container) Dirty() bool          { return c.dirty }
func (c *Container) Send()                {}

func (c *Container) Close() {
	c.closed = true
	// Revealed values should not stay revealed for the next time this
	// environment is opened.
	clear(c.revealed)
}

func (c *Container) markDirty() {
	c.dirty = true
	c.deps.ReportDirty(true)
}

func (c *Container) toggleReveal(rowID string) {
	if c.locked[rowID] {
		return
	}
	if c.revealed[rowID] {
		delete(c.revealed, rowID)
		return
	}
	c.revealed[rowID] = true
}

// ensureKey runs when a row is marked secret: without a key there is nothing to
// encrypt the value with, so ask for one and undo the mark if the user backs
// out.
func (c *Container) ensureKey(rowID string) {
	secretui.EnsureKey(c.secretDeps(), func(ok bool) {
		if !ok {
			c.table.SetCell(rowID, container.KVColSecret, "")
		}
	})
}

func (c *Container) secretDeps() secretui.Deps {
	return secretui.Deps{
		Manager:   c.deps.Secrets,
		Dialogs:   c.deps.Dialogs,
		Clipboard: c.deps.Clipboard,
		Toast:     c.deps.Toast,
		Error:     c.deps.ShowError,
		Changed:   c.deps.WakeNow,
	}
}

func (c *Container) Save() error {
	values := container.DumpKV(c.table)
	// A row that was never decrypted keeps the ciphertext it was loaded with.
	for i := range values {
		if c.locked[values[i].ID] {
			values[i].Locked = true
		}
	}
	c.env.Spec.Values = values
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
		c.titleRow(th, id),
		ui.HLine(th.Stroke.Thin, th.Border),
		c.lockedBanner(th),
		ui.ViewOf(c.table).Grow(1),
	).Grow(1)
}

// titleRow keeps the name on the left and search, Add and Save together on the
// right.
func (c *Container) titleRow(th *theme.Theme, id string) ui.View {
	return ui.Row(
		ui.EditableLabel("title-"+id, c.env.MetaData.Name).
			Placeholder("Name").
			OnSave(func(s string) { container.RenameEnvironment(c.deps, c.env, s) }).
			Grow(1),
		ui.Row(
			ui.TextField("env-search-"+id, c.query).
				Placeholder("Search...").
				IconStart(icons.Search).
				OnChange(func(s string) { c.query = s; c.table.SetFilter(s) }).
				Width(200),
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
		).Gap(th.Spacing.S).Align(ui.AlignCenter),
	).Gap(th.Spacing.S).
		Justify(ui.JustifyBetween).
		Align(ui.AlignCenter).
		PaddingTop(th.Spacing.XS).
		PaddingBottom(th.Spacing.S).
		PaddingLeft(th.Spacing.M).
		PaddingRight(th.Spacing.M)
}

// lockedBanner explains why some values show as ciphertext, and offers the way
// out.
func (c *Container) lockedBanner(th *theme.Theme) ui.View {
	if len(c.locked) == 0 {
		return ui.Row()
	}
	return ui.Row(
		ui.Muted("Some values are still encrypted: the secret key is not available."),
		ui.Spacer(),
		ui.Button("env-unlock-"+c.env.MetaData.ID, ui.Text("Unlock")).IconStart(icons.Key).
			OnClick(func() {
				secretui.EnsureKey(c.secretDeps(), func(ok bool) {
					if ok {
						c.reloadValues()
					}
				})
			}),
	).Gap(th.Spacing.S).Align(ui.AlignCenter).
		PaddingLeft(th.Spacing.M).
		PaddingRight(th.Spacing.M).
		PaddingTop(th.Spacing.S)
}

// reloadValues re-reads the environment now that the key is available, so the
// values that were locked show up decrypted.
func (c *Container) reloadValues() {
	if c.deps.Catalog == nil {
		return
	}
	if err := c.deps.Catalog.Load(); err != nil {
		c.deps.ShowError(err)
		return
	}
	fresh := c.deps.Catalog.EnvironmentByID(c.env.MetaData.ID)
	if fresh == nil {
		return
	}
	c.env = container.CopyEnv(fresh)
	clear(c.locked)
	for _, kv := range c.env.Spec.Values {
		if kv.Locked {
			c.locked[kv.ID] = true
		}
	}
	container.LoadKV(c.table, c.env.Spec.Values)
	c.table.SetFilter(c.query)
}

// TabIcon marks the tab as an environment's.
func (c *Container) TabIcon(th *theme.Theme) (icons.Icon, render.Color) {
	return reqicons.Env(th)
}
