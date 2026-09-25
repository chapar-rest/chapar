package pages

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

// workspacesMaxWidth caps the page's header and card column on wide windows.
const workspacesMaxWidth = 760

// WorkspacesDeps wires the workspaces page to the app.
type WorkspacesDeps struct {
	Repo repository.RepositoryV2
	// List returns the loaded workspaces.
	List func() []*domain.Workspace
	// ActiveID returns the ID of the open workspace.
	ActiveID func() string
	// Summary describes the open workspace's contents, e.g. "3 collections".
	Summary func() string
	// Load reloads the catalog from disk.
	Load  func() error
	Error func(error)
	// Use switches to a workspace.
	Use func(*domain.Workspace)
	// Renamed runs after the open workspace was renamed on disk.
	Renamed func(*domain.Workspace)
}

// Workspaces lists workspaces as cards with search, create, switch, rename
// and delete.
type Workspaces struct {
	deps  WorkspacesDeps
	query string
}

func NewWorkspacesPage(deps WorkspacesDeps) *Workspaces {
	return &Workspaces{deps: deps}
}

// isDefault reports whether ws is the built-in workspace, which can't be
// renamed or deleted.
func isDefault(ws *domain.Workspace) bool {
	return ws.MetaData.ID == "default" || ws.MetaData.Name == domain.DefaultWorkspaceName
}

// visible returns the workspaces matching the search, sorted by name.
func (p *Workspaces) visible() []*domain.Workspace {
	q := strings.ToLower(strings.TrimSpace(p.query))
	var out []*domain.Workspace
	for _, w := range p.deps.List() {
		if q == "" || strings.Contains(strings.ToLower(w.MetaData.Name), q) {
			out = append(out, w)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].MetaData.Name) < strings.ToLower(out[j].MetaData.Name)
	})
	return out
}

func (p *Workspaces) reload() {
	if err := p.deps.Load(); err != nil {
		p.deps.Error(err)
	}
}

func (p *Workspaces) create(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "New Space"
	}
	if err := p.deps.Repo.CreateWorkspace(domain.NewWorkspace(name)); err != nil {
		p.deps.Error(fmt.Errorf("failed to create space: %w", err))
		return
	}
	p.reload()
}

func (p *Workspaces) rename(ws *domain.Workspace, name string) {
	name = strings.TrimSpace(name)
	if name == "" || name == ws.MetaData.Name {
		return
	}
	old := ws.MetaData.Name
	ws.MetaData.Name = name
	if err := p.deps.Repo.UpdateWorkspace(ws); err != nil {
		ws.MetaData.Name = old
		p.deps.Error(fmt.Errorf("failed to rename space: %w", err))
		return
	}
	if ws.MetaData.ID == p.deps.ActiveID() && p.deps.Renamed != nil {
		p.deps.Renamed(ws)
	}
	p.reload()
}

func (p *Workspaces) delete(ws *domain.Workspace) {
	if isDefault(ws) || ws.MetaData.ID == p.deps.ActiveID() {
		return
	}
	if err := p.deps.Repo.DeleteWorkspace(ws); err != nil {
		p.deps.Error(fmt.Errorf("failed to delete space: %w", err))
		return
	}
	p.reload()
}

func (p *Workspaces) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	dialogs := c.Dialogs()

	header := ui.Column(
		ui.Row(
			ui.Title("Spaces"),
			ui.Spacer(),
			ui.TextField("ws-search", p.query).Placeholder("Search spaces").IconStart(icons.Search).Width(220).
				OnChange(func(s string) { p.query = s }),
			ui.Button("ws-new", ui.Text("New Space")).Primary().IconStart(icons.Plus).OnClick(func() {
				dialogs.ShowInput("New space", "Space name", p.create, nil)
			}),
		).Gap(th.Spacing.S).Align(ui.AlignCenter),
		ui.Muted("Each space keeps its own collections, requests and environments."),
	).Gap(th.Spacing.XS).Grow(1).MaxWidth(workspacesMaxWidth)

	items := p.visible()
	var body ui.View
	if len(items) == 0 {
		empty := ui.EmptyState("No spaces", "Create a space to get started.").EmptyIcon(icons.Boxes)
		if p.query != "" {
			empty = ui.EmptyState("No matches", fmt.Sprintf("No space matches %q.", p.query)).EmptyIcon(icons.Search)
		}
		body = empty.Grow(1)
	} else {
		cards := make([]ui.View, 0, len(items))
		for _, ws := range items {
			cards = append(cards, p.card(c, ws))
		}
		body = ui.Scroll("ws-scroll",
			ui.Row(
				ui.Column(cards...).Gap(th.Spacing.M).Grow(1).MaxWidth(workspacesMaxWidth),
			).Justify(ui.JustifyCenter).Padding(th.Spacing.L),
		).Grow(1)
	}

	return ui.Column(
		ui.Row(header).Justify(ui.JustifyCenter).PaddingXY(th.Spacing.L, th.Spacing.L),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(body).Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}

func (p *Workspaces) card(c *ui.Ctx, ws *domain.Workspace) ui.View {
	th := c.Theme()
	dialogs := c.Dialogs()
	id := ws.MetaData.ID
	active := id == p.deps.ActiveID()
	locked := isDefault(ws)

	iconColor := th.ForegroundMuted
	if active {
		iconColor = th.Accent
	}

	title := []ui.View{ui.Strong(ws.MetaData.Name)}
	if active {
		title = append(title, ui.Badge("Active").Tone(ui.BadgeSuccess))
	}
	if locked {
		title = append(title, ui.Badge("Default"))
	}

	detail := "Switch to this space to work with its requests."
	if active {
		detail = "Open now."
		if p.deps.Summary != nil {
			if s := p.deps.Summary(); s != "" {
				detail = s
			}
		}
	}

	actions := []ui.View{}
	if !active {
		actions = append(actions, ui.Button("ws-use-"+id, ui.Text("Switch")).IconStart(icons.ArrowRightLeft).
			OnClick(func() { p.deps.Use(ws) }))
	}

	renameTip := "Rename"
	if locked {
		renameTip = "The default space can't be renamed"
	}
	actions = append(actions, ui.IconButton("ws-edit-"+id, icons.Pencil).Disabled(locked).Tooltip(renameTip).
		OnClick(func() {
			dialogs.ShowInputValue("Rename space", "Space name", ws.MetaData.Name,
				func(name string) { p.rename(ws, name) }, nil)
		}))

	deleteTip := "Delete"
	switch {
	case locked:
		deleteTip = "The default space can't be deleted"
	case active:
		deleteTip = "Switch to another space to delete this one"
	}
	actions = append(actions, ui.IconButton("ws-del-"+id, icons.Trash2).Disabled(locked || active).Tooltip(deleteTip).
		OnClick(func() {
			dialogs.ShowAction("Delete space?",
				fmt.Sprintf("%q and all its collections, requests and environments will be deleted. This can't be undone.", ws.MetaData.Name),
				func() { p.delete(ws) }, nil)
		}))

	return ui.Card("", "", ui.Row(
		ui.Icon(icons.Boxes, th.Metrics.IconSizeMD, iconColor),
		ui.Column(
			ui.Row(title...).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.Muted(detail),
		).Gap(th.Spacing.XS).Grow(1).Shrink(1),
		ui.Row(actions...).Gap(th.Spacing.XS).Align(ui.AlignCenter),
	).Gap(th.Spacing.M).Align(ui.AlignCenter))
}
