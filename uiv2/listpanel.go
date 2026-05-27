package uiv2

import (
	"fmt"
	"log"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

func buildListPanel(tag any, panel *core.Frame, tabView *TabView, repo repository.RepositoryV2) {
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

func buildSectionHeader(panel *core.Frame, title string) {
	core.NewText(panel).
		SetType(core.TextTitleMedium).
		SetText(title).
		Styler(func(s *styles.Style) {
			s.Padding.Set(units.Dp(8), units.Dp(4))
			s.SetTextWrap(false)
		})
}

func buildRequestsList(panel *core.Frame, tabView *TabView, repo repository.RepositoryV2) {
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
			tabView.Open("request:demo-get-users", "GET /users", stubRequestPage("GET /users"))
		})
		addListItem(panel, "POST /login", "demo-post-login", func() {
			tabView.Open("request:demo-post-login", "POST /login", stubRequestPage("POST /login"))
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
			tabView.Open(key, label, stubRequestPage(label))
		})
	}
}

func buildEnvironmentsList(panel *core.Frame, tabView *TabView, repo repository.RepositoryV2) {
	buildSectionHeader(panel, "Environments")

	environments, err := repo.LoadEnvironments()
	if err != nil {
		log.Println(err)
		core.NewText(panel).
			SetType(core.TextBodyMedium).
			SetText("Failed to load environments")
		return
	}

	if len(environments) == 0 {
		core.NewText(panel).
			SetType(core.TextBodyMedium).
			SetText("No environments")
		return
	}

	for _, env := range environments {
		env := env
		label := env.GetName()
		if label == "" {
			label = "Untitled environment"
		}
		key := "env:" + env.MetaData.ID
		addListItem(panel, label, env.MetaData.ID, func() {
			tabView.Open(key, label, stubEnvPage(env))
		})
	}
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

func stubRequestPage(name string) func(content *core.Frame) {
	return func(content *core.Frame) {
		content.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Padding.Set(units.Dp(16))
			s.Gap.Set(units.Dp(8))
		})
		core.NewText(content).
			SetType(core.TextHeadlineSmall).
			SetText(name)
		core.NewText(content).
			SetType(core.TextBodyMedium).
			SetText("Request page placeholder")
	}
}

func stubEnvPage(env *domain.Environment) func(content *core.Frame) {
	name := env.GetName()
	return func(content *core.Frame) {
		content.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Padding.Set(units.Dp(16))
			s.Gap.Set(units.Dp(8))
		})
		core.NewText(content).
			SetType(core.TextHeadlineSmall).
			SetText(name)
		core.NewText(content).
			SetType(core.TextBodyMedium).
			SetText(fmt.Sprintf("Environment page placeholder (%d variables)", len(env.Spec.Values)))
	}
}
