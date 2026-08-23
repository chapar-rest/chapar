package uiv2

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

func (app *App) topBar(c *ui.Ctx) ui.View {
	th := c.Theme()

	workspacesOptions := make([]ui.SelectOption, 0, len(app.catalog.Workspaces))
	for _, workspace := range app.catalog.Workspaces {
		workspacesOptions = append(workspacesOptions, ui.SelectOption{Label: workspace.MetaData.Name, Value: workspace.MetaData.ID})
	}

	environmentsOptions := make([]ui.SelectOption, 0, len(app.catalog.Environments))
	for _, environment := range app.catalog.Environments {
		environmentsOptions = append(environmentsOptions, ui.SelectOption{Label: environment.MetaData.Name, Value: environment.MetaData.ID})
	}

	return ui.Row(
		ui.Select("active-space", workspacesOptions).
			Width(180).
			Selected(activeWorkspaceIndex(app.catalog.Workspaces, app.activeWorkspaceId)).
			OnChange(func(v string) {
				app.activeWorkspaceId = v
				if err := app.onWorkspaceChanged(app.catalog.WorkspaceByID(v)); err != nil {
					c.Dialogs().ShowError("Failed to change workspace", err.Error(), func() {})
				}
				if err := app.loadEnvTree(); err != nil {
					c.Dialogs().ShowError("Failed to load environment tree", err.Error(), func() {})
				}
				if err := app.loadRequestsTree(); err != nil {
					c.Dialogs().ShowError("Failed to load requests tree", err.Error(), func() {})
				}
			}),
		ui.IconButton("btn-add", icons.Plus).OnClick(func() {}),
		ui.Spacer(),
		ui.Button("cmd-palette", ui.Text("Commands")).Width(300).
			IconStart(icons.Search).
			Hint(c.Commands().ToggleLabel()).
			OnClick(func() { c.Commands().Show() }),
		ui.Spacer(),
		ui.Select("active-environment", environmentsOptions).
			Width(180).
			Selected(activeEnvironmentIndex(app.catalog.Environments, app.activeEnvironmentId)).
			OnChange(func(v string) {
				app.activeEnvironmentId = v
				if err := app.onSelectedEnvChanged(app.catalog.EnvironmentByID(v)); err != nil {
					c.Dialogs().ShowError("Failed to change environment", err.Error(), func() {})
				}
			}),
		ui.IconButton("settings-btn", icons.Settings).OnClick(func() {
			c.Dialogs().Show(ui.DialogOpts{
				Title:  "Settings",
				Width:  800,
				Height: 600,
				Body:   app.settings,
				Actions: []ui.DialogAction{
					{Label: "Cancel", OnClick: func() {}},
					{Label: "Default", OnClick: func() {}},
					{Label: "Save", Primary: true, OnClick: func() {}},
				},
			})
		}),
	).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.S).
		Background(ui.TokenChrome).
		Shrink(0) // chrome must not compress when a page is taller than the window
}

func activeEnvironmentIndex(environments []*domain.Environment, activeEnvironmentId string) int {
	for i, environment := range environments {
		if environment.MetaData.ID == activeEnvironmentId {
			return i
		}
	}
	return -1
}

func activeWorkspaceIndex(workspaces []*domain.Workspace, activeWorkspaceId string) int {
	for i, workspace := range workspaces {
		if workspace.MetaData.ID == activeWorkspaceId {
			return i
		}
	}
	return -1
}

func (app *App) onSelectedEnvChanged(env *domain.Environment) error {
	appState := prefs.GetAppState()
	if env != nil {
		if appState.Spec.SelectedEnvironment == nil {
			appState.Spec.SelectedEnvironment = &domain.SelectedEnvironment{}
		}

		appState.Spec.SelectedEnvironment.ID = env.MetaData.ID
		appState.Spec.SelectedEnvironment.Name = env.MetaData.Name
	} else {
		appState.Spec.SelectedEnvironment = nil
	}

	if err := prefs.UpdateAppState(appState); err != nil {
		return err
	}

	if env != nil {
		app.catalog.SetActiveEnv(env.MetaData.ID)
	} else {
		app.catalog.SetActiveEnv("")
	}

	return nil
}

func (app *App) onWorkspaceChanged(ws *domain.Workspace) error {
	appState := prefs.GetAppState()
	appState.Spec.ActiveWorkspace = &domain.ActiveWorkspace{
		ID:   ws.MetaData.ID,
		Name: ws.MetaData.Name,
	}

	if err := prefs.UpdateAppState(appState); err != nil {
		return err
	}

	app.repo.SetActiveWorkspace(ws.GetName())

	if err := app.catalog.Load(); err != nil {
		return err
	}

	return nil
}
