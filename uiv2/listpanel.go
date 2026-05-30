package uiv2

import (
	"fmt"
	"log"
	"os"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"

	appevents "github.com/chapar-rest/chapar/internal/events"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/uiv2/pages"
	"github.com/chapar-rest/chapar/uiv2/widget"
)

func buildListPanel(tag any, panel *core.Frame, tabView *widget.TabView, repo repository.RepositoryV2) {
	panel.DeleteChildren()

	switch tag {
	case "requests":
		buildRequestsList(panel, tabView, repo)
	case "environments":
		buildEnvironmentsList(panel, tabView, repo)
	case "protofiles":
		buildSectionHeader(panel, "Proto Files")
		core.NewText(panel).
			SetType(core.TextBodyMedium).
			SetText("Coming soon")
	case "workspaces":
		buildSectionHeader(panel, "Workspaces")
		core.NewText(panel).
			SetType(core.TextBodyMedium).
			SetText("Coming soon")
	default:
		buildSectionHeader(panel, fmt.Sprint(tag))
	}
}

func refreshEnvironmentsList(panel *core.Frame, tabView *widget.TabView, repo repository.RepositoryV2) {
	buildEnvironmentsList(panel, tabView, repo)
	panel.Update()
}

func buildSectionHeader(panel *core.Frame, title string) {
	core.NewText(panel).
		SetType(core.TextTitleSmall).
		SetText(title).
		Styler(func(s *styles.Style) {
			s.Padding.Set(units.Dp(4), units.Dp(2))
			s.SetTextWrap(false)
		})
}

func buildRequestsList(panel *core.Frame, tabView *widget.TabView, repo repository.RepositoryV2) {
	buildSectionHeader(panel, "Requests")

	requests, err := repo.LoadRequests()
	if err != nil {
		log.Println(err)
		core.NewText(panel).
			SetType(core.TextBodyMedium).
			SetText("Failed to load requests")
		return
	}

	if len(requests) == 0 {
		// Placeholder rows when the workspace has no requests yet.
		addListItem(panel, "GET /users", "demo-get-users", func() {
			tabView.Open("request:demo-get-users", "GET /users", pages.RequestPage("GET /users"))
		})
		addListItem(panel, "POST /login", "demo-post-login", func() {
			tabView.Open("request:demo-post-login", "POST /login", pages.RequestPage("POST /login"))
		})
		return
	}

	for _, req := range requests {
		req := req
		label := req.GetName()
		if label == "" {
			label = "Untitled request"
		}
		key := "request:" + req.MetaData.ID
		addListItem(panel, label, req.MetaData.ID, func() {
			tabView.Open(key, label, pages.RequestPage(label))
		})
	}
}

func buildEnvironmentsList(panel *core.Frame, tabView *widget.TabView, repo repository.RepositoryV2) {
	buildSectionHeader(panel, "Environments")

	actions := core.NewFrame(panel)
	actions.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Gap.Set(units.Dp(4))
		s.Padding.Set(units.Dp(2), units.Dp(0))
	})

	core.NewButton(actions).
		SetType(core.ButtonOutlined).
		SetText("Import").
		SetIcon(icons.FileOpen).
		OnClick(func(e events.Event) {
			openImportEnvironmentDialog(panel, tabView, repo)
		})

	core.NewButton(actions).
		SetType(core.ButtonOutlined).
		SetText("New").
		SetIcon(icons.Add).
		OnClick(func(e events.Event) {
			env := domain.NewEnvironment("New Environment")
			if err := repo.CreateEnvironment(env); err != nil {
				log.Println(err)
				core.MessageDialog(panel, err.Error(), "Create failed")
				return
			}
			appevents.EnvironmentChangeTopic.Publish(env)
			refreshEnvironmentsList(panel, tabView, repo)
			openEnvironmentTab(tabView, repo, env)
	})

	search := core.NewTextField(panel)
	search.SetPlaceholder("Search...")
	search.SetTrailingIcon(icons.Search)

	list := core.NewFrame(panel)
	list.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	populate := func(filter string) {
		list.DeleteChildren()
		environments, err := repo.LoadEnvironments()
		if err != nil {
			log.Println(err)
			core.NewText(list).
				SetType(core.TextBodyMedium).
				SetText("Failed to load environments")
			list.Update()
			return
		}
		if len(environments) == 0 {
			core.NewText(list).
				SetType(core.TextBodyMedium).
				SetText("No environments")
			list.Update()
			return
		}
		q := filter
		for _, env := range environments {
			env := env
			label := env.GetName()
			if label == "" {
				label = "Untitled environment"
			}
			if q != "" && !strings.Contains(strings.ToLower(label), strings.ToLower(q)) {
				continue
			}
			addListItem(list, label, env.MetaData.ID, func() {
				openEnvironmentTab(tabView, repo, env)
			})
		}
		list.Update()
	}

	search.OnChange(func(e events.Event) {
		populate(search.Text())
	})

	populate("")
}

func openEnvironmentTab(tabView *widget.TabView, repo repository.RepositoryV2, env *domain.Environment) {
	label := env.GetName()
	if label == "" {
		label = "Untitled environment"
	}
	key := "env:" + env.MetaData.ID
	tabView.Open(key, label, pages.EnvironmentPage(pages.EnvironmentPageDeps{
		Env:     env,
		Repo:    repo,
		TabView: tabView,
		TabKey:  key,
	}))
}

func openImportEnvironmentDialog(panel *core.Frame, tabView *widget.TabView, repo repository.RepositoryV2) {
	d := core.NewBody("Import environment")
	fp := core.NewFilePicker(d).SetExtensions(".json")
	fp.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Min.Set(units.Dp(480), units.Dp(320))
	})

	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			path := fp.SelectedFile()
			if path == "" {
				return
			}
			data, err := os.ReadFile(path)
			if err != nil {
				core.MessageDialog(d, err.Error(), "Import failed")
				return
			}
			if err := importer.ImportPostmanEnvironment(data, repo); err != nil {
				core.MessageDialog(d, err.Error(), "Import failed")
				return
			}
			appevents.EnvironmentChangeTopic.Publish(nil)
			refreshEnvironmentsList(panel, tabView, repo)
			d.Close()
		})
	})

	d.RunDialog(panel)
}

func addListItem(panel *core.Frame, label, _ string, onClick func()) {
	row := core.NewFrame(panel)
	row.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Clickable, abilities.Hoverable, abilities.Focusable)
		s.Padding.Set(units.Dp(8), units.Dp(12))
		s.Border.Radius = styles.BorderRadiusSmall
		s.Cursor = cursors.Pointer
		if s.Is(states.Hovered) {
			s.Background = colors.Scheme.SurfaceContainerHighest
		}
	})
	row.OnClick(func(e events.Event) {
		if onClick != nil {
			onClick()
		}
	})
	lbl := core.NewText(row).
		SetType(core.TextBodyMedium).
		SetText(label)
	lbl.Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
		s.Grow.Set(1, 0)
	})
}
