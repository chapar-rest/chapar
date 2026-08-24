package uiv2

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

	// ui.Center loses JustifyCenter when this pane sits inside a Splitter — the
	// splitter overwrites the pane root style with flex-grow only. Spacers center
	// vertically; AlignCenter on the inner column handles horizontal alignment.
	return ui.Column(
		ui.Spacer(),
		ui.Column(
			ui.Image("workspace-chapar", assets.ChaparPNG).Width(128),
			ui.Subtitle("Chapar").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
			ui.Column(rows...).
				Gap(th.Spacing.XS).
				Width(emptyWorkspaceWidth).
				MarginTop(th.Spacing.XL),
		).Gap(th.Spacing.M).Align(ui.AlignCenter),
		ui.Spacer(),
	).Grow(1)
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
