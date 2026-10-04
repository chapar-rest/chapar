package websocket

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	wsegress "github.com/chapar-rest/chapar/internal/egress/websocket"
	"github.com/chapar-rest/chapar/ui/container"
)

// maxLogEvents bounds the log: a busy feed must not grow memory forever.
// The oldest events go first.
const maxLogEvents = 5000

// previewRunes is how much of a message its row shows.
const previewRunes = 200

type logEntry struct {
	seq int
	ev  wsegress.Event
}

// messageLog is the Messages tab: every event of the connections opened
// from this tab, newest first, and the selected one in full below.
type messageLog struct {
	id       string
	entries  []logEntry
	nextSeq  int
	sent     int
	received int

	table    *ui.Table
	detail   *ui.Editor
	selected int // seq of the selected entry; 0 is none
	filter   string
}

func (l *messageLog) init(id string) {
	l.id = id
	l.table = ui.NewTable([]ui.TableColumn{
		{ID: "msg", Label: "Message", Kind: ui.TableColText, Locked: true},
		{ID: "time", Label: "Time", Kind: ui.TableColText, Width: 130, Locked: true},
		{ID: "size", Label: "Size", Kind: ui.TableColText, Width: 80, Locked: true},
	}, nil)
	l.table.Editable = false
	l.table.Selectable = true
	l.table.MinHeight = 120
	l.table.OnRowClick = func(rowID string) {
		seq, err := strconv.Atoi(rowID)
		if err != nil {
			return
		}
		l.selected = seq
		l.showDetail()
	}
	l.detail = container.NewResponseEditor(nil, highlight.Noop{})
}

func (l *messageLog) close() {
	if l.detail != nil {
		l.detail.Close()
	}
}

func (l *messageLog) len() int { return len(l.entries) }

func (l *messageLog) counts() (sent, received int) { return l.sent, l.received }

func (l *messageLog) add(events []wsegress.Event) {
	for _, e := range events {
		l.nextSeq++
		l.entries = append(l.entries, logEntry{seq: l.nextSeq, ev: e})
		if e.IsMessage() {
			switch e.Dir {
			case wsegress.DirSent:
				l.sent++
			case wsegress.DirReceived:
				l.received++
			}
		}
	}
	if over := len(l.entries) - maxLogEvents; over > 0 {
		l.entries = append(l.entries[:0:0], l.entries[over:]...)
	}
	l.rebuild()
}

func (l *messageLog) clear() {
	l.entries = nil
	l.sent, l.received = 0, 0
	l.selected = 0
	l.rebuild()
	l.showDetail()
}

func (l *messageLog) rebuild() {
	rows := make([]ui.TableRow, 0, len(l.entries))
	for i := len(l.entries) - 1; i >= 0; i-- {
		e := l.entries[i]
		rows = append(rows, ui.TableRow{
			ID:       strconv.Itoa(e.seq),
			Selected: e.seq == l.selected,
			Icon:     eventIcon(e.ev),
			Cells: map[string]string{
				"msg":  preview(e.ev),
				"time": e.ev.At.Format("15:04:05.000"),
				"size": eventSize(e.ev),
			},
		})
	}
	l.table.SetRows(rows)
	l.table.SetFilter(l.filter)
}

func (l *messageLog) entry(seq int) (logEntry, bool) {
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].seq == seq {
			return l.entries[i], true
		}
	}
	return logEntry{}, false
}

func (l *messageLog) showDetail() {
	e, ok := l.entry(l.selected)
	if !ok {
		l.detail = container.ReplaceEditor(l.detail, nil, highlight.Noop{})
		return
	}
	data, hl := detailOf(e.ev)
	l.detail = container.ReplaceEditor(l.detail, data, hl)
}

func (l *messageLog) view(th *theme.Theme, ctx *ui.Ctx, deps container.Deps) ui.View {
	if len(l.entries) == 0 {
		return ui.Column(
			ui.Muted("No messages yet. Connect, then send a message from the Message tab."),
		).Grow(1)
	}
	toolbar := ui.Row(
		ui.TextField("ws-filter-"+l.id, l.filter).
			Placeholder("Filter messages").
			IconStart(icons.Search).
			OnChange(func(s string) {
				l.filter = s
				l.table.SetFilter(s)
			}).
			Grow(1),
		ui.Button("ws-clear-"+l.id, ui.Text("Clear")).IconStart(icons.Eraser).
			Tooltip("Clear the message log").
			OnClick(l.clear),
	).Gap(th.Spacing.S).Align(ui.AlignCenter)

	var detail ui.View = ui.Muted("Select a message to see it in full.")
	if _, ok := l.entry(l.selected); ok {
		detail = container.ResponseEditorMenu(l.detail, ctx, deps, "message.txt")
	}
	return ui.Column(
		toolbar,
		ui.Splitter("ws-log-split-"+l.id, ui.Vertical, ui.ViewOf(l.table).Grow(1), ui.Column(detail).Grow(1)).
			Percents(60, 40).
			HandleOnHover().
			Grow(1),
	).Gap(th.Spacing.S).Grow(1)
}

func eventIcon(e wsegress.Event) icons.Icon {
	switch {
	case e.Dir == wsegress.DirSent:
		return icons.ArrowUp
	case e.Dir == wsegress.DirReceived:
		return icons.ArrowDown
	case e.Kind == wsegress.KindError:
		return icons.CircleAlert
	case e.Kind == wsegress.KindOpen:
		return icons.Plug
	case e.Kind == wsegress.KindClose:
		return icons.Unplug
	}
	return icons.Info
}

// preview is the one-line form of an event its row shows.
func preview(e wsegress.Event) string {
	switch e.Kind {
	case wsegress.KindText:
		return oneLine(string(e.Data))
	case wsegress.KindBinary:
		n := min(len(e.Data), 16)
		s := fmt.Sprintf("Binary · %s", hex.EncodeToString(e.Data[:n]))
		if n < len(e.Data) {
			s += "…"
		}
		return s
	case wsegress.KindPing:
		return "Ping"
	case wsegress.KindPong:
		return "Pong"
	}
	return e.Info
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= previewRunes {
		return s
	}
	r := []rune(s)
	return string(r[:previewRunes]) + "…"
}

func eventSize(e wsegress.Event) string {
	if !e.IsMessage() {
		return ""
	}
	return container.FormatBytes(len(e.Data))
}

// detailOf is the full form of an event: JSON indented, binary as a hex
// dump.
func detailOf(e wsegress.Event) ([]byte, highlight.Highlighter) {
	switch e.Kind {
	case wsegress.KindText:
		if json.Valid(e.Data) {
			var out bytes.Buffer
			if err := json.Indent(&out, e.Data, "", container.EditorIndent()); err == nil {
				return out.Bytes(), highlight.NewJSON()
			}
		}
		return e.Data, highlight.Noop{}
	case wsegress.KindBinary:
		return []byte(hex.Dump(e.Data)), highlight.Noop{}
	case wsegress.KindPing, wsegress.KindPong:
		if len(e.Data) > 0 {
			return []byte(fmt.Sprintf("%s payload:\n%s", strings.ToUpper(e.Kind[:1])+e.Kind[1:], hex.Dump(e.Data))), highlight.Noop{}
		}
	}
	text := e.Info
	if text == "" {
		text = preview(e)
	}
	return []byte(text), highlight.Noop{}
}
