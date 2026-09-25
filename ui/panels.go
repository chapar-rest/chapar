package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/logger"
)

// ConsolePanel shows application logs in a bottom push drawer.
type ConsolePanel struct {
	level   int // index into consoleLevels; 0 shows every level
	visible bool

	// ed shows the filtered logs read-only so they can be selected and
	// copied. It is rebuilt only when shown lines change (see sig).
	ed     *ui.Editor
	sig    consoleSig
	synced bool
}

// consoleSig identifies what the console editor currently shows.
type consoleSig struct {
	count int
	last  time.Time
	lastM string
	level int
}

// consoleLevels are the level filter choices; an empty Value shows all.
var consoleLevels = []ui.SegmentItem{
	{Label: "All", Value: ""},
	{Label: "Info", Value: "info"},
	{Label: "Warn", Value: "warn"},
	{Label: "Error", Value: "error"},
}

// consoleBarHeight and consoleControlHeight size the compact header.
const (
	consoleBarHeight     = 30
	consoleControlHeight = 22
)

func (p *ConsolePanel) Toggle() { p.visible = !p.visible }
func (p *ConsolePanel) Show()   { p.visible = true }
func (p *ConsolePanel) Hide()   { p.visible = false }
func (p *ConsolePanel) IsVisible() bool {
	return p.visible
}

// Wrap hosts the page area in a bottom push drawer so the console is
// available on every page while the nav rail stays put.
func (p *ConsolePanel) Wrap(c *ui.Ctx, content ui.View) ui.View {
	return ui.Drawer("console-drawer", p.Layout(c), content).
		Open(p.visible).
		Edge(ui.EdgeBottom).
		Push().
		Size(180).
		Resizable(true).
		OnOpenChange(func(v bool) { p.visible = v }).
		Grow(1)
}

// Layout keeps a compact header fixed above the log view, so only the logs
// scroll. Text search is the editor's own Cmd+F.
func (p *ConsolePanel) Layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	logs := logger.GetLogs()

	var body ui.View
	if p.syncEditor(logs) {
		body = ui.ViewOf(p.ed).Grow(1)
	} else {
		body = ui.Column(
			ui.Caption("No logs").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
		).Padding(th.Spacing.S).Grow(1)
	}
	return ui.Column(p.header(c, logs), body).Grow(1).Background(ui.TokenChrome)
}

func (p *ConsolePanel) header(c *ui.Ctx, logs []domain.Log) ui.View {
	th := c.Theme()
	var errs, warns int
	for _, log := range logs {
		switch log.Level {
		case "error":
			errs++
		case "warn":
			warns++
		}
	}
	items := []ui.View{ui.Caption("CONSOLE")}
	if errs > 0 {
		items = append(items, ui.Badge(countLabel(errs, "error")).Tone(ui.BadgeError))
	}
	if warns > 0 {
		items = append(items, ui.Badge(countLabel(warns, "warning")).Tone(ui.BadgeWarning))
	}
	items = append(items,
		ui.Spacer(),
		ui.Segmented("console-level", consoleLevels...).
			Selected(p.level).
			OnSelectItem(func(i int, _ string) { p.level = i }).
			Height(consoleControlHeight),
		ui.IconButton("console-clear", icons.Trash2).
			Height(consoleControlHeight).
			Tooltip("Clear").
			OnClick(func() { logger.Clear() }),
		ui.IconButton("console-copy", icons.Copy).
			Height(consoleControlHeight).
			Tooltip("Copy all logs").
			OnClick(func() { c.Clipboard().Set(p.formatLogs(logs)) }),
		ui.IconButton("console-close", icons.X).
			Height(consoleControlHeight).
			Tooltip("Close").
			OnClick(func() { p.Hide() }),
	)
	return ui.Row(items...).
		Align(ui.AlignCenter).
		Gap(th.Spacing.XS).
		Height(consoleBarHeight).
		PaddingLeft(th.Spacing.S).
		PaddingRight(th.Spacing.XS)
}

func countLabel(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// syncEditor rebuilds the log editor when the shown lines change, keeping the
// scroll position. It reports whether any line matches the filter.
func (p *ConsolePanel) syncEditor(logs []domain.Log) bool {
	sig := consoleSig{count: len(logs), level: p.level}
	if n := len(logs); n > 0 {
		sig.last, sig.lastM = logs[n-1].Time, logs[n-1].Message
	}
	if p.synced && sig == p.sig {
		return p.ed != nil
	}
	p.sig, p.synced = sig, true

	var b strings.Builder
	var toks []highlight.Token
	level := consoleLevels[p.level].Value
	for i := len(logs) - 1; i >= 0; i-- {
		log := logs[i]
		if level != "" && log.Level != level {
			continue
		}
		line := fmt.Sprintf("[%s] %s: %s", log.Time.Format(time.TimeOnly), strings.ToUpper(log.Level), log.Message)
		if log.Level == "print" {
			line = log.Message
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		start := b.Len()
		b.WriteString(line)
		toks = append(toks, highlight.Token{Start: start, End: b.Len(), Class: logClass(log.Level)})
	}

	var scroll float32
	if p.ed != nil {
		scroll = p.ed.ScrollPx
		p.ed.Close()
		p.ed = nil
	}
	if b.Len() == 0 {
		return false
	}
	p.ed = ui.NewEditor([]byte(b.String()), &logHighlighter{tokens: toks},
		ui.WithReadOnly(), ui.WithoutGutter(), ui.WithSoftWrap(true))
	p.ed.ScrollPx = scroll
	return true
}

func logClass(level string) highlight.ColorClass {
	switch level {
	case "info":
		return highlight.ClassSuccess
	case "error":
		return highlight.ClassError
	case "warn":
		return highlight.ClassWarning
	}
	return highlight.ClassMuted
}

// logHighlighter colors each log entry by level. The console editor is
// read-only and rebuilt when logs change, so the tokens never go stale.
type logHighlighter struct {
	tokens []highlight.Token
	sent   bool
}

func (h *logHighlighter) Update([]byte)                     {}
func (h *logHighlighter) UpdateEdit([]byte, highlight.Edit) {}
func (h *logHighlighter) Close()                            {}
func (h *logHighlighter) Poll() ([]highlight.Token, bool) {
	if h.sent {
		return nil, false
	}
	h.sent = true
	return h.tokens, true
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
		rows = append(rows, ui.Paragraph(item.text).Style(style))
	}
	if len(rows) == 0 {
		rows = append(rows, ui.Text("No notifications").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
	}
	return ui.Column(
		ui.Strong("Notifications").Margin(th.Spacing.S),
		ui.Scroll("notif-scroll", ui.Column(rows...).Gap(th.Spacing.S)),
	).Gap(th.Spacing.S).Grow(1)
}
