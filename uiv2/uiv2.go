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
	"github.com/chapar-rest/chapar/uiv2/pages"
	"github.com/chapar-rest/chapar/uiv2/settings"
	"github.com/chapar-rest/chapar/uiv2/theme"
	"github.com/chapar-rest/chapar/uiv2/widget"
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

	menu := widget.NewSideMenu(b)
	menu.AddItem(widget.SideMenuItem{Tag: "requests", Name: "Requests", Icon: icons.SwapHoriz})
	menu.AddItem(widget.SideMenuItem{Tag: "environments", Name: "Envs", Icon: icons.Menu})
	menu.AddItem(widget.SideMenuItem{Tag: "protofiles", Name: "Proto", Icon: icons.Folder})
	menu.AddItem(widget.SideMenuItem{Tag: "workspaces", Name: "Workspaces", Icon: icons.Workspaces})
	menu.AddBottomAction(icons.Settings, "Settings", func(ctx core.Widget) {
		settings.OpenDialog(ctx)
	})

	mainSplit := widget.NewSplits(b)
	mainSplit.SetName("main-split")
	mainSplit.SetHandlesVisible(false)

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

	pages.NewWelcome(content)
	tabView := widget.NewTabView(content)

	sectionActive := false

	updateContent := func() {
		if tabView.IsEmpty() {
			content.StackTop = 0 // welcome
		} else {
			content.StackTop = 1 // tabView
		}
		content.UpdateStackedVisibility()
		content.NeedsLayout()
		content.Update()
	}

	updateChrome := func() {
		listPanel.SetState(!sectionActive, states.Invisible)
		listPanel.Restyle()
		if sectionActive {
			mainSplit.SetSplits(defaultListSplit, 1-defaultListSplit)
		} else {
			mainSplit.SetSplits(0, 1)
		}
		mainSplit.SetHandlesVisible(sectionActive)
		mainSplit.NeedsLayout()
		mainSplit.Update()
	}

	tabView.OnChange(func(bool) { updateContent() })

	updateChrome()
	updateContent()

	menu.OnSelect(func(item widget.SideMenuItem) {
		sectionActive = true
		updateChrome()
		buildListPanel(item.Tag, listPanel, tabView, repo)
		updateContent()
		listPanel.Update()
	})

	b.NewWindow().SetDisplayTitle(false).RunMain()
}
