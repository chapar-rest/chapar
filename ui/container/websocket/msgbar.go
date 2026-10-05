package websocket

import (
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/container"
)

// The Message tab's toolbar gives up room in steps as the pane narrows,
// instead of pushing Send past the pane's edge.
const (
	barFull     = iota // every label, and Send's shortcut chip
	barNoHint          // Send drops its shortcut chip
	barSaveIcon        // Save message is an icon
	barSendIcon        // Send is an icon too
	barNarrow          // the format select is narrower
	barLevels
)

// msgBar keeps how far the toolbar has collapsed, and the width each step
// needed when it was laid out. Layout only knows widths after the views are
// built, so a change shows on the next frame.
type msgBar struct {
	level int
	fit   [barLevels]float32 // content width of each step; 0 until laid out
	size  float32            // body text size the widths were taken at
}

func (c *Container) messageBar(ctx *ui.Ctx, th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	w := c.req.Spec.WebSocket
	b := &c.bar
	if b.size != th.Typography.Body.Size {
		*b = msgBar{size: th.Typography.Body.Size}
	}

	selW := float32(170)
	if b.level >= barNarrow {
		selW = 110
	}
	format := ui.Select("ws-format-"+id, formatOptions).Width(selW).
		Selected(optionIndex(w.MessageFormat, formatOptions)).
		OnChange(func(v string) { c.setFormat(v) })

	// Icon forms take their own ids: widget state is kept per id, and a
	// Button's is not an IconButton's.
	styles := ui.ThemeStyles(th)
	const saveTip = "Keep this message in the Saved tab"
	save := ui.View(ui.Button("ws-keep-"+id, ui.Text("Save message")).IconStart(icons.BookmarkPlus).
		Tooltip(saveTip).OnClick(c.saveDraft))
	if b.level >= barSaveIcon {
		save = ui.IconButton("ws-keep-icon-"+id, icons.BookmarkPlus).Style(styles.ButtonSecondary).
			Tooltip(saveTip).OnClick(c.saveDraft)
	}
	disabled := c.state != stateConnected
	var send ui.View
	switch {
	case b.level >= barSendIcon:
		send = ui.IconButton("ws-send-icon-"+id, icons.SendHorizontal).Style(styles.ButtonPrimary).
			Tooltip("Send (⌘↵)").Disabled(disabled).OnClick(c.sendDraft)
	case b.level >= barNoHint:
		send = ui.Button("ws-send-"+id, ui.Text("Send")).Primary().IconStart(icons.SendHorizontal).
			Tooltip("⌘↵").Disabled(disabled).OnClick(c.sendDraft)
	default:
		send = ui.Button("ws-send-"+id, ui.Text("Send")).Primary().IconStart(icons.SendHorizontal).Hint("⌘↵").
			Disabled(disabled).OnClick(c.sendDraft)
	}

	gap := th.Spacing.S
	row := ui.Row(
		format,
		ui.Spacer(),
		ui.Row(save, send).Gap(gap).Align(ui.AlignCenter),
	).Gap(gap).Align(ui.AlignCenter).Layout(ctx)
	c.measureBar(ctx, row, b.level, gap)
	return ui.Raw(row)
}

// measureBar picks the toolbar's step for the width it was given: the
// fullest step whose content fits, stepping down one at a time through
// steps not laid out yet.
func (c *Container) measureBar(ctx *ui.Ctx, el *layout.Element, built int, gap float32) {
	prev := el.AfterLayout
	el.AfterLayout = func(e *layout.Element) bool {
		b := &c.bar
		avail, _ := e.LayoutSize()
		if len(e.Children) == 3 && avail > 0 {
			// The buttons' row shrinks with the pane; its buttons do not.
			need, _ := e.Children[0].LayoutSize()
			need += 2 * gap // either side of the spacer
			for i, btn := range e.Children[2].Children {
				bw, _ := btn.LayoutSize()
				if i > 0 {
					need += gap
				}
				need += bw
			}
			b.fit[built] = need
			next := built
			for next > 0 && b.fit[next-1] > 0 && b.fit[next-1] <= avail {
				next--
			}
			if next == built && need > avail && built < barLevels-1 {
				next++
				for next < barLevels-1 && b.fit[next] > avail {
					next++
				}
			}
			if next != b.level {
				b.level = next
				// Not Invalidate: a repaint asked for during layout is dropped
				// once this frame presents. Animate wakes the next frame.
				ctx.Animate(0)
			}
		}
		if prev != nil {
			return prev(e)
		}
		return false
	}
}

func (c *Container) messagePane(ctx *ui.Ctx, th *theme.Theme) []ui.View {
	id := c.req.MetaData.ID
	editor := ui.View(ui.ViewOf(c.msgEd).Grow(1))
	if c.req.Spec.WebSocket.MessageFormat == domain.WebSocketFormatJSON {
		editor = container.JSONBodyEditor("ws-msg-"+id, c.msgEd, c.deps, nil)
	}
	return []ui.View{c.messageBar(ctx, th), editor}
}
