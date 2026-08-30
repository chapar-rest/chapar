package uiv2

import (
	"fmt"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

// ConsolePanel shows application logs in a bottom push drawer.
type ConsolePanel struct {
	filter  string
	visible bool
}

func (p *ConsolePanel) Toggle() { p.visible = !p.visible }
func (p *ConsolePanel) Show()  { p.visible = true }
func (p *ConsolePanel) Hide()  { p.visible = false }
func (p *ConsolePanel) IsVisible() bool {
	return p.visible
}

// WrapWorkspace hosts workspace content in a bottom push drawer so the sidebar tree stays put.
func (p *ConsolePanel) WrapWorkspace(c *ui.Ctx, workspace ui.View) ui.View {
	return ui.Drawer("console-drawer", p.Layout(c), workspace).
		Open(p.visible).
		Edge(ui.EdgeBottom).
		Push().
		Size(180).
		Resizable(true).
		OnOpenChange(func(v bool) { p.visible = v }).
		Grow(1)
}

func (p *ConsolePanel) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	logs := logger.GetLogs()
	rows := []ui.View{
		ui.Row(
			ui.TextField("console-filter", p.filter).Placeholder("Filter logs…").IconStart(icons.Search).
				OnChange(func(s string) { p.filter = s }).Grow(1),
			ui.Button("console-clear", ui.Text("Clear")).OnClick(func() { logger.Clear() }),
			ui.Button("console-copy", ui.Text("Copy")).OnClick(func() {
				c.Clipboard().Set(p.formatLogs(logs))
			}),
			ui.IconButton("console-close", icons.X).OnClick(func() { p.Hide() }),
		).Gap(th.Spacing.S).Padding(th.Spacing.S),
	}
	for i := len(logs) - 1; i >= 0; i-- {
		log := logs[i]
		line := fmt.Sprintf("[%s] %s: %s", log.Time.Format(time.DateTime), strings.ToUpper(log.Level), log.Message)
		if log.Level == "print" {
			line = log.Message
		}
		if p.filter != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(p.filter)) {
			continue
		}
		style := ui.Spec{}.TextColor(ui.TokenForegroundMuted)
		switch log.Level {
		case "info":
			style = ui.Spec{}.TextColor(ui.TokenSuccess)
		case "error":
			style = ui.Spec{}.TextColor(ui.TokenError)
		case "warn":
			style = ui.Spec{}.TextColor(ui.TokenWarning)
		}
		rows = append(rows, ui.Text(line).Style(style))
	}
	if len(rows) == 1 {
		rows = append(rows, ui.Text("No logs available").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
	}
	return ui.Column(
		ui.Scroll("console-scroll", ui.Column(rows...).Gap(th.Spacing.XS).Padding(th.Spacing.S)),
	).Grow(1).Background(ui.TokenChrome)
}

func (p *ConsolePanel) formatLogs(logs []domain.Log) string {
	var b strings.Builder
	for _, log := range logs {
		fmt.Fprintf(&b, "[%s] %s: %s\n", log.Time.Format(time.DateTime), strings.ToUpper(log.Level), log.Message)
	}
	return b.String()
}

// NotificationHistory retains recent notifications for the panel.
type NotificationHistory struct {
	items []notificationItem
}

type notificationItem struct {
	text string
	kind ui.ToastVariant
	at   time.Time
}

func (n *NotificationHistory) Add(text string, kind ui.ToastVariant) {
	n.items = append(n.items, notificationItem{text: text, kind: kind, at: time.Now()})
	if len(n.items) > 100 {
		n.items = n.items[len(n.items)-100:]
	}
}

func (n *NotificationHistory) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	rows := []ui.View{}
	for i := len(n.items) - 1; i >= 0; i-- {
		item := n.items[i]
		style := ui.Spec{}.TextColor(ui.TokenForegroundMuted)
		switch item.kind {
		case ui.ToastError:
			style = ui.Spec{}.TextColor(ui.TokenError)
		case ui.ToastWarning:
			style = ui.Spec{}.TextColor(ui.TokenWarning)
		}
		rows = append(rows, ui.Text(item.text).Style(style))
	}
	if len(rows) == 0 {
		rows = append(rows, ui.Text("No notifications").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
	}
	return ui.Column(
		ui.Strong("Notifications").Margin(th.Spacing.S),
		ui.Scroll("notif-scroll", ui.Column(rows...).Gap(th.Spacing.S)),
	).Gap(th.Spacing.S).Grow(1)
}
