package uiv2

import (
	"log"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/internal/domain"
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

const defaultListSplit = 0.18

func clampListSplit(v float32) float32 {
	return math32.Clamp(v, 0.05, 0.95)
}

func seedListSplit(appState domain.AppState) float32 {
	if appState.Spec.Layout != nil && appState.Spec.Layout.MainListSplit > 0 {
		return clampListSplit(appState.Spec.Layout.MainListSplit)
	}
	return defaultListSplit
}

func isFullPageSection(s pages.Section) bool {
	_, ok := s.(pages.FullPageSection)
	return ok
}

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

	listSplit := seedListSplit(appState)

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
		s.Overflow.X = styles.OverflowHidden
		s.Min.X.Zero()
	})

	welcome.New(content)
	tabView := widget.NewTabView(content)

	fullPage := core.NewFrame(content)
	fullPage.SetName("full-page")
	fullPage.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Overflow.Y = styles.OverflowAuto
		s.Overflow.X = styles.OverflowHidden
		s.Min.X.Zero()
	})

	sections := []pages.Section{
		request.New(repo, tabView),
		environment.New(repo, tabView),
		protofile.New(repo),
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
		if isFullPageSection(currentSection) {
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
		isFullPage := isFullPageSection(currentSection)
		listPanel.SetState(!sectionActive || isFullPage, states.Invisible)
		listPanel.Restyle()
		if sectionActive && !isFullPage {
			mainSplit.SetSplits(listSplit, 1-listSplit)
		} else {
			mainSplit.SetSplits(0, 1)
		}
		mainSplit.SetHandlesVisible(sectionActive && !isFullPage)
		mainSplit.NeedsLayout()
		mainSplit.Update()
	}

	mainSplit.OnResize(func(sp []float32) {
		if !sectionActive || isFullPageSection(currentSection) {
			return
		}
		if len(sp) < 2 {
			return
		}
		v := sp[0]
		if v < 0.05 || v > 0.95 || math32.Abs(v-listSplit) < 0.005 {
			return
		}
		listSplit = v
		state := prefs.GetAppState()
		if state.Spec.Layout == nil {
			state.Spec.Layout = &domain.LayoutState{}
		}
		state.Spec.Layout.MainListSplit = v
		_ = prefs.UpdateAppState(state)
	})

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
