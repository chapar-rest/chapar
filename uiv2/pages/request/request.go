package request

import (
	"log"

	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/uiv2/pages/internal/listui"
	"github.com/chapar-rest/chapar/uiv2/widget"
)

// Section is the requests sidebar section.
type Section struct {
	repo    repository.RepositoryV2
	tabView *widget.TabView
}

// New constructs the requests section.
func New(repo repository.RepositoryV2, tabView *widget.TabView) *Section {
	return &Section{repo: repo, tabView: tabView}
}

// MenuItem returns the side menu entry for requests.
func (s *Section) MenuItem() widget.SideMenuItem {
	return widget.SideMenuItem{Tag: "requests", Name: "Requests", Icon: icons.SwapHoriz}
}

// BuildListPanel renders the requests list in the left rail.
func (s *Section) BuildListPanel(panel *core.Frame) {
	listui.SectionHeader(panel, "Requests")

	requests, err := s.repo.LoadRequests()
	if err != nil {
		log.Println(err)
		core.NewText(panel).
			SetType(core.TextBodyMedium).
			SetText("Failed to load requests")
		return
	}

	if len(requests) == 0 {
		// Placeholder rows when the workspace has no requests yet.
		listui.Item(panel, "GET /users", "demo-get-users", func() {
			s.openTab("request:demo-get-users", "GET /users")
		})
		listui.Item(panel, "POST /login", "demo-post-login", func() {
			s.openTab("request:demo-post-login", "POST /login")
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
		listui.Item(panel, label, req.MetaData.ID, func() {
			s.openTab(key, label)
		})
	}
}

func (s *Section) openTab(key, label string) {
	s.tabView.Open(key, label, s.page(label))
}

func (s *Section) page(name string) func(content *core.Frame) {
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
