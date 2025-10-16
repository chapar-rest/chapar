package widgets

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type GitStatusWidget struct {
	theme *chapartheme.Theme
	repo  repository.RepositoryV2

	// UI elements
	clickable widget.Clickable
	icon      material.LabelStyle
	text      material.LabelStyle
}

func NewGitStatusWidget(theme *chapartheme.Theme, repo repository.RepositoryV2) *GitStatusWidget {
	return &GitStatusWidget{
		theme: theme,
		repo:  repo,
		icon:  MaterialIcons("git_branch", theme),
		text:  material.Body2(theme.Material(), ""),
	}
}

func (g *GitStatusWidget) Layout(gtx layout.Context) layout.Dimensions {
	status := g.repo.GetSyncStatus()
	
	// Don't show widget if sync is not enabled
	if !status.IsEnabled {
		return layout.Dimensions{}
	}

	// Update text based on status
	g.updateText(status)

	// Handle click events
	if g.clickable.Clicked(gtx) {
		// Could open git settings or show more details
		// For now, we'll just refresh the status
	}

	// Layout the widget
	return layout.Inset{
		Left:   unit.Dp(8),
		Right:  unit.Dp(8),
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return Clickable(gtx, &g.clickable, unit.Dp(4), func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
				Spacing:   layout.SpaceEnd,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return g.icon.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return g.text.Layout(gtx)
				}),
			)
		})
	})
}

func (g *GitStatusWidget) updateText(status *repository.SyncStatus) {
	// Get backend-specific information
	branch := "main"
	if status.BackendInfo != nil {
		if b, ok := status.BackendInfo["branch"].(string); ok {
			branch = b
		}
	}

	if status.IsSyncing {
		g.text.Text = "Syncing..."
		g.text.Color = g.theme.Palette.ContrastFg
		g.icon = MaterialIcons("sync", g.theme)
	} else if status.UncommittedChanges {
		g.text.Text = branch + " (uncommitted)"
		g.text.Color = g.theme.Palette.ContrastFg
		g.icon = MaterialIcons("git_branch", g.theme)
	} else if status.HasRemote {
		if status.LastSyncTime != nil {
			g.text.Text = branch + " (synced)"
		} else {
			g.text.Text = branch + " (remote)"
		}
		g.text.Color = g.theme.Palette.ContrastFg
		g.icon = MaterialIcons("cloud_done", g.theme)
	} else {
		g.text.Text = branch + " (local)"
		g.text.Color = g.theme.Palette.ContrastFg
		g.icon = MaterialIcons("git_branch", g.theme)
	}
}

// RefreshStatus forces a refresh of the git status
func (g *GitStatusWidget) RefreshStatus() {
	// The status will be refreshed on the next layout call
}
