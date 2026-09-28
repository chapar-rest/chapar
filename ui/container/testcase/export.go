package testcase

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mirzakhany/yoga/ui"
	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/testrun"
	"github.com/chapar-rest/chapar/ui/container"
)

// Export writes the case, the requests it sends and optionally an
// environment to one file that `chapar test` runs without the workspace.
func (c *Container) Export() {
	if c.deps.Catalog == nil || c.deps.Dialogs == nil {
		return
	}
	tc, _, ok := c.current()
	if !ok {
		c.deps.ShowError(fmt.Errorf("fix the YAML before exporting: %s", c.yamlErr))
		return
	}
	host := c.deps.Dialogs()
	if host == nil {
		return
	}

	envOpts := []ui.SelectOption{{Label: "No environment", Value: ""}}
	for _, e := range c.deps.Catalog.AllEnvironments() {
		envOpts = append(envOpts, ui.SelectOption{Label: e.GetName(), Value: e.ID()})
	}
	envID, secrets := "", false
	if c.deps.ActiveEnv != nil {
		if env := c.deps.ActiveEnv(); env != nil {
			envID = env.ID()
		}
	}

	host.Show(ui.DialogOpts{
		Title:  "Export test case",
		Width:  560,
		Height: 340,
		Body: func(ctx *ui.Ctx) ui.View {
			th := ctx.Theme()
			rows := []ui.View{
				ui.Paragraph("One file with the test case and the requests it sends. Run it anywhere, such as in CI, with chapar test <file>."),
				ui.Form("tc-export-form",
					ui.FormSelect("tc-export-env", "Environment", "Values to run with. The command line can set or replace them with --env-file, --os-env and --var.",
						envOpts, optionIndex(envID, envOpts), func(v string) { envID = v }),
					ui.FormSwitch("tc-export-secrets", "Include secret values", "Off: secret values are left out and have to be given when the file runs.",
						secrets, func(v bool) { secrets = v }),
				),
			}
			if secrets && envID != "" {
				rows = append(rows, ui.Alert("Secret values will be written to the file in plain text.", ui.AlertWarning))
			}
			return ui.Column(rows...).Gap(th.Spacing.M).PaddingXY(th.Spacing.M, th.Spacing.S)
		},
		Actions: []ui.DialogAction{
			{Label: "Cancel"},
			{Label: "Export", Primary: true, OnClick: func() { c.saveBundle(tc, envID, secrets) }},
		},
	})
}

func (c *Container) saveBundle(tc *domain.TestCase, envID string, secrets bool) {
	opts := testrun.BundleOptions{IncludeSecrets: secrets}
	if envID != "" {
		if env := c.deps.Catalog.EnvironmentByID(envID); env != nil {
			opts.Env = container.CopyEnv(env)
		}
	}
	b, err := testrun.Bundle(tc.GetName(), []*domain.TestCase{tc}, c.deps.Catalog, c.deps.Catalog.CollectionByID, opts)
	if err != nil {
		c.deps.ShowError(err)
		return
	}
	data, err := yaml.Marshal(b)
	if err != nil {
		c.deps.ShowError(err)
		return
	}
	if c.deps.Files == nil {
		return
	}
	fd := c.deps.Files()
	if fd == nil {
		return
	}
	fd.Show(ui.FileDialogOpts{
		Title:             "Export test case",
		Mode:              ui.FileDialogSaveFile,
		Filters:           []ui.FileFilter{{Label: "YAML files", Exts: []string{".yaml", ".yml"}}},
		ShowSaveFilter:    true,
		AllowCreateFolder: true,
		OnConfirm: func(paths []string) {
			if len(paths) == 0 {
				return
			}
			path := paths[0]
			if ext := filepath.Ext(path); ext != ".yaml" && ext != ".yml" {
				path += ".yaml"
			}
			// Keep a file with secrets readable by its owner only.
			mode := os.FileMode(0o644)
			if secrets && opts.Env != nil {
				mode = 0o600
			}
			if err := os.WriteFile(path, data, mode); err != nil {
				c.deps.ShowError(err)
				return
			}
			msg := "Exported. Run it with: chapar test " + filepath.Base(path)
			if n := len(b.Spec.SecretsLeftOut); n > 0 {
				msg += fmt.Sprintf(" (%d secret value(s) left out)", n)
			}
			c.deps.Toast(msg)
		},
	})
}
