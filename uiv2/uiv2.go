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
	"github.com/chapar-rest/chapar/uiv2/pages/environment"
	"github.com/chapar-rest/chapar/uiv2/pages/protofile"
	"github.com/chapar-rest/chapar/uiv2/pages/request"
	"github.com/chapar-rest/chapar/uiv2/pages/welcome"
	"github.com/chapar-rest/chapar/uiv2/pages/workspace"
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

	welcome.New(content)
	tabView := widget.NewTabView(content)

	fullPage := core.NewFrame(content)
	fullPage.SetName("full-page")
	fullPage.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Overflow.Y = styles.OverflowAuto
	})

	sections := []pages.Section{
		request.New(repo, tabView),
		environment.New(repo, tabView),
		protofile.New(),
		workspace.New(repo),
	}
	byTag := map[any]pages.Section{}
	for _, s := range sections {
		mi := s.MenuItem()
		menu.AddItem(mi)
		byTag[mi.Tag] = s
	}
	menu.AddBottomAction(icons.Settings, "Settings", func(ctx core.Widget) {
		settings.OpenDialog(ctx)
	})

	sectionActive := false
	var currentSection pages.Section

	updateContent := func() {
		if _, ok := currentSection.(pages.FullPageSection); ok {
			content.StackTop = 2 // fullPage
		} else if tabView.IsEmpty() {
			content.StackTop = 0 // welcome
		} else {
			content.StackTop = 1 // tabView
		}
		content.UpdateStackedVisibility()
		content.NeedsLayout()
		content.Update()
	}

	updateChrome := func() {
		isFullPage := false
		if _, ok := currentSection.(pages.FullPageSection); ok {
			isFullPage = true
		}
		listPanel.SetState(!sectionActive || isFullPage, states.Invisible)
		listPanel.Restyle()
		if sectionActive && !isFullPage {
			mainSplit.SetSplits(defaultListSplit, 1-defaultListSplit)
		} else {
			mainSplit.SetSplits(0, 1)
		}
		mainSplit.SetHandlesVisible(sectionActive && !isFullPage)
		mainSplit.NeedsLayout()
		mainSplit.Update()
	}

	tabView.OnChange(func(bool) { updateContent() })

	updateChrome()
	updateContent()

	menu.OnSelect(func(item widget.SideMenuItem) {
		sectionActive = true
		if s, ok := byTag[item.Tag]; ok {
			currentSection = s
			if fp, ok := s.(pages.FullPageSection); ok {
				fullPage.DeleteChildren()
				fp.BuildContent(fullPage)
				fullPage.Update()
			} else {
				listPanel.DeleteChildren()
				s.BuildListPanel(listPanel)
				listPanel.Update()
			}
		}
		updateChrome()
		updateContent()
	})

	b.NewWindow().SetDisplayTitle(false).RunMain()
}
