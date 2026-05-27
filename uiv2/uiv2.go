package uiv2

import (
	"log"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/uiv2/settings"
	"github.com/chapar-rest/chapar/uiv2/theme"
)

const defaultListSplit = 0.28

func Run() {
	theme.Init()
	settings.Init()
	settings.LoadOrLog()

	b := core.NewBody("Chapar")

	appState := prefs.GetAppState()
	workspacePath := prefs.GetWorkspacePath()

	repo, err := repository.NewFilesystemV2(workspacePath, appState.Spec.ActiveWorkspace.Name)
	if err != nil {
		log.Fatal(err)
	}

	if err := NewAppBar(b, repo); err != nil {
		log.Fatal(err)
	}

	// Lay out the body as a row: side menu | resizable (list panel | content).
	b.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 1)
	})

	menu := NewSideMenu(b)
	menu.AddItem(SideMenuItem{Tag: "requests", Name: "Requests", Icon: icons.SwapHoriz})
	menu.AddItem(SideMenuItem{Tag: "environments", Name: "Envs", Icon: icons.Menu})
	menu.AddItem(SideMenuItem{Tag: "protofiles", Name: "Proto", Icon: icons.Folder})
	menu.AddItem(SideMenuItem{Tag: "workspaces", Name: "Workspaces", Icon: icons.Workspaces})
	menu.AddBottomAction(icons.Settings, "Settings", func(ctx core.Widget) {
		settings.OpenDialog(ctx)
	})

	mainSplit := NewSplits(b)
	mainSplit.SetName("main-split")

	listPanel := core.NewFrame(mainSplit)
	listPanel.SetName("list-panel")
	listPanel.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.Min.X.Dp(180)
		s.Overflow.Y = styles.OverflowAuto
		s.Padding.Set(units.Dp(8))
		s.Gap.Set(units.Dp(2))
		s.Background = colors.Scheme.SurfaceContainer
	})

	content := core.NewFrame(mainSplit)
	content.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Display = styles.Stacked
	})

	NewWelcome(content)
	tabView := NewTabView(content)

	sectionActive := false
	updateContent := func() {
		showWelcome := tabView.IsEmpty() && !sectionActive
		// Stacked layout only displays the StackTop child; welcome is index 0,
		// tabView is index 1.
		if showWelcome {
			content.StackTop = 0
		} else {
			content.StackTop = 1
		}
		content.UpdateStackedVisibility()
		content.NeedsLayout()
		content.Update()
	}
	tabView.OnChange(func(bool) { updateContent() })

	// Start with list panel collapsed; only the welcome screen is shown.
	mainSplit.SetSplits(0, 1)
	listPanel.SetState(true, states.Invisible)
	mainSplit.SetHandlesVisible(false)
	updateContent()

	menu.OnSelect(func(item SideMenuItem) {
		listPanel.SetState(false, states.Invisible)
		listPanel.Restyle()
		sectionActive = true
		mainSplit.SetHandlesVisible(true)
		mainSplit.SetSplits(defaultListSplit, 1-defaultListSplit)
		buildListPanel(item.Tag, listPanel, tabView, repo)
		updateContent()
		mainSplit.NeedsLayout()
		listPanel.Update()
	})

	b.NewWindow().SetDisplayTitle(false).RunMain()
}
