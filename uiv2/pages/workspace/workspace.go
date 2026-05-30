package workspace

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"

	"github.com/chapar-rest/chapar/internal/domain"
	appevents "github.com/chapar-rest/chapar/internal/events"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/uiv2/widget"
)

const (
	pagePadTop         = 30
	pagePadSide        = 250
	actionsColumnWidth = 220
	gapAfterTitle      = 15
	gapBeforeList      = 30
)

// Section is the workspaces sidebar section.
type Section struct {
	repo repository.RepositoryV2
}

// New constructs the workspaces section.
func New(repo repository.RepositoryV2) *Section {
	return &Section{repo: repo}
}

// MenuItem returns the side menu entry for workspaces.
func (s *Section) MenuItem() widget.SideMenuItem {
	return widget.SideMenuItem{Tag: "workspaces", Name: "Workspaces", Icon: icons.Workspaces}
}

// BuildListPanel is a no-op; workspaces render as a full page.
func (s *Section) BuildListPanel(_ *core.Frame) {}

// BuildContent renders the workspaces management page in the main content area.
func (s *Section) BuildContent(content *core.Frame) {
	content.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 1)
		st.Padding.Set(units.Dp(pagePadTop), units.Dp(pagePadSide))
	})

	top := core.NewFrame(content)
	top.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Align.Items = styles.Start
		st.Justify.Content = styles.SpaceBetween
		st.Gap.Set(units.Dp(16))
		st.Grow.Set(1, 0)
	})

	leftCol := core.NewFrame(top)
	leftCol.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 0)
		st.Gap.Set(units.Dp(gapAfterTitle))
	})
	core.NewText(leftCol).
		SetType(core.TextTitleLarge).
		SetText("Workspaces")
	core.NewText(leftCol).
		SetType(core.TextBodyMedium).
		SetText("Workspaces are a way to organize your work.\nYou can create multiple workspaces to separate your projects.").
		Styler(func(st *styles.Style) {
			st.Color = colors.Scheme.OnSurfaceVariant
			st.SetTextWrap(true)
		})

	rightCol := core.NewFrame(top)
	rightCol.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Align.Items = styles.End
		st.Gap.Set(units.Dp(gapAfterTitle))
		st.Grow.Set(0, 0)
		st.Min.X.Dp(actionsColumnWidth)
		st.Max.X.Dp(actionsColumnWidth)
	})
	newBtn := core.NewButton(rightCol).
		SetType(core.ButtonTonal).
		SetText("New Workspace").
		SetIcon(icons.Add)
	search := core.NewTextField(rightCol)
	search.SetPlaceholder("Search...")
	search.SetTrailingIcon(icons.Search)
	search.SendChangeOnInput()
	search.Styler(func(st *styles.Style) {
		st.Grow.Set(1, 0)
	})

	list := core.NewFrame(content)
	list.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 1)
		st.Gap.Set(units.Dp(8))
		st.Padding.Set(units.Dp(gapBeforeList), units.Dp(0), units.Dp(0), units.Dp(0))
	})

	var populate func(filter string)
	populate = func(filter string) {
		list.DeleteChildren()

		workspaces, err := s.repo.LoadWorkspaces()
		if err != nil {
			log.Println(err)
			core.NewText(list).
				SetType(core.TextBodyMedium).
				SetText("Failed to load workspaces")
			list.Update()
			return
		}

		sort.Slice(workspaces, func(i, j int) bool {
			return workspaces[i].GetName() < workspaces[j].GetName()
		})

		activeID := ""
		if aw := prefs.GetAppState().Spec.ActiveWorkspace; aw != nil {
			activeID = aw.ID
		}

		q := strings.ToLower(strings.TrimSpace(filter))
		filtered := make([]*domain.Workspace, 0, len(workspaces))
		for _, ws := range workspaces {
			name := ws.GetName()
			if q == "" || strings.Contains(strings.ToLower(name), q) {
				filtered = append(filtered, ws)
			}
		}

		if len(filtered) == 0 {
			core.NewText(list).
				SetType(core.TextBodyMedium).
				SetText("No workspaces")
			list.Update()
			return
		}

		for _, ws := range filtered {
			s.buildRow(list, content, ws, activeID, populate)
		}
		list.Update()
	}

	newBtn.OnClick(func(e events.Event) {
		s.onNew(content, populate)
	})

	search.OnChange(func(e events.Event) {
		populate(search.Text())
	})

	populate("")
}

func (s *Section) buildRow(list, content *core.Frame, ws *domain.Workspace, activeID string, refresh func(string)) {
	name := ws.GetName()
	readOnly := name == domain.DefaultWorkspaceName
	isActive := ws.ID() == activeID
	canDelete := !readOnly && !isActive

	sheet := core.NewFrame(list)
	sheet.Styler(func(st *styles.Style) {
		st.Direction = styles.Column
		st.Grow.Set(1, 0)
		st.Background = colors.Scheme.SurfaceContainerLow
		st.Border.Style.Set(styles.BorderSolid)
		st.Border.Width.Set(units.Dp(1))
		st.Border.Color.Set(colors.Scheme.OutlineVariant)
		st.Border.Radius = styles.BorderRadiusMedium
	})

	row := core.NewFrame(sheet)
	row.Styler(func(st *styles.Style) {
		st.Direction = styles.Row
		st.Align.Items = styles.Center
		st.Grow.Set(1, 0)
		st.Padding.Set(units.Dp(10), units.Dp(12))
	})

	nameWrap := core.NewFrame(row)
	nameWrap.Styler(func(st *styles.Style) {
		st.Grow.Set(0, 0)
	})

	if readOnly {
		core.NewText(nameWrap).
			SetType(core.TextBodyMedium).
			SetText(name).
			Styler(func(st *styles.Style) {
				st.Color = colors.Scheme.OnSurfaceVariant
				st.SetTextWrap(false)
			})
	} else {
		label := widget.NewEditableLabel(nameWrap, name)
		label.SetLabelType(core.TextBodyMedium)
		label.OnCommit(func(next string) {
			s.onRename(content, ws, next, refresh)
		})
	}

	core.NewStretch(row)

	if canDelete {
		delBtn := core.NewButton(row).
			SetType(core.ButtonAction).
			SetIcon(icons.Delete)
		delBtn.Styler(func(st *styles.Style) {
			st.IconSize.Set(units.Dp(20))
			st.Min.Set(units.Dp(20))
			st.Max.Set(units.Dp(20))
			st.Padding.Zero()
			st.Grow.Set(0, 0)
		})
		delBtn.OnClick(func(e events.Event) {
			s.confirmDelete(content, ws, refresh)
		})
	}
}

func (s *Section) onNew(content *core.Frame, refresh func(string)) {
	ws := domain.NewWorkspace("New Workspace")
	if err := s.repo.CreateWorkspace(ws); err != nil {
		log.Println(err)
		core.MessageDialog(content, err.Error(), "Create failed")
		return
	}
	appevents.WorkspaceChangeTopic.Publish(ws)
	refresh("")
}

func (s *Section) onRename(content *core.Frame, ws *domain.Workspace, name string, refresh func(string)) {
	ws.SetName(name)
	if err := s.repo.UpdateWorkspace(ws); err != nil {
		log.Println(err)
		core.MessageDialog(content, err.Error(), "Update failed")
		return
	}
	appevents.WorkspaceChangeTopic.Publish(ws)
	refresh("")
}

func (s *Section) confirmDelete(content *core.Frame, ws *domain.Workspace, refresh func(string)) {
	d := core.NewBody("Delete workspace")
	core.NewText(d).
		SetType(core.TextSupporting).
		SetText(fmt.Sprintf("Are you sure you want to delete %q?", ws.GetName()))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			s.onDelete(content, ws, refresh)
			d.Close()
		})
	})
	d.RunDialog(content)
}

func (s *Section) onDelete(content *core.Frame, ws *domain.Workspace, refresh func(string)) {
	if err := s.repo.DeleteWorkspace(ws); err != nil {
		log.Println(err)
		core.MessageDialog(content, err.Error(), "Delete failed")
		return
	}
	appevents.WorkspaceChangeTopic.Publish(nil)
	refresh("")
}
