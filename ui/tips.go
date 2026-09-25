package ui

import (
	"github.com/chapar-rest/chapar/assets"
	"github.com/mirzakhany/yoga/ui"
)

const emptyWorkspaceWidth = float32(380)

// emptyWorkspace is the VS Code-style watermark shown when no tab is open.
func (w *Workspace) emptyWorkspace(c *ui.Ctx) ui.View {
	th := c.Theme()

	rows := []ui.View{
		w.tipRow("tip-commands", "Show All Commands", c.Commands().ToggleLabel(), func() {
			c.Commands().Show()
		}),
		w.tipRow("tip-save", "Save", "⌘S", nil),
		w.tipRow("tip-send", "Send / Invoke", "⌘Enter", nil),
		w.tipRow("tip-settings", "Open Settings", "⌘,", w.onSettings),
		w.tipRow("tip-open", "Open a request or environment", "Double-click", nil),
	}

	return ui.Center(
		ui.Column(
			ui.Image("workspace-chapar", assets.ChaparPNG).Width(128),
			ui.Subtitle("Chapar").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
			ui.Column(rows...).
				Gap(th.Spacing.XS).
				Width(emptyWorkspaceWidth).
				MarginTop(th.Spacing.XL),
		).Gap(th.Spacing.M).Align(ui.AlignCenter),
	).Background(ui.TokenSurface)
}

func (w *Workspace) tipRow(id, label, hint string, onClick func()) ui.View {
	btn := ui.Button(id, ui.Text(label)).
		Ghost().
		HoverFill().
		Hint(hint).
		Width(emptyWorkspaceWidth)
	if onClick != nil {
		btn.OnClick(onClick)
	}
	return btn
}
