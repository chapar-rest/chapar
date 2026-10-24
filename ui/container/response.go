package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/assets"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/util"
)

// FormatBytes returns a human-readable size (e.g. "1.2 kB").
func FormatBytes(n int) string {
	if n < 0 {
		n = 0
	}
	return humanize.Bytes(uint64(n))
}

// DisplayBody returns bytes for the response editor. When raw is true, the
// original body is shown; otherwise Pretty is preferred when available.
//
// The result may be the response's own body buffer rather than a copy, so
// callers must treat it as read-only. Nothing mutates a response body once it
// is built, which is what lets the editor share it instead of copying tens of
// megabytes per tab — see ReplaceResponseEditor.
func DisplayBody(res *egress.Response, raw bool) []byte {
	if res == nil {
		return nil
	}
	if raw {
		if len(res.Body) > 0 {
			return res.Body
		}
		return []byte(res.JSON)
	}
	if res.Pretty != "" {
		return []byte(res.Pretty)
	}
	// Prefer the body buffer over res.JSON: both hold the same bytes when the
	// body was left unformatted, and the body can be read in place while
	// res.JSON (a string, kept for jsonpath) would have to be copied.
	if len(res.Body) > 0 {
		return res.Body
	}
	return []byte(res.JSON)
}

// BodyHighlighter returns a highlighter for the response body kind.
func BodyHighlighter(kind string) highlight.Highlighter {
	switch kind {
	case util.BodyKindJSON:
		return highlight.NewJSON()
	case util.BodyKindXML, util.BodyKindHTML:
		return highlight.NewXML()
	default:
		return highlight.Noop{}
	}
}

// NewResponseEditor returns a read-only, soft-wrapped editor for something a
// server sent back: a body, headers, metadata or cookies. It can be searched,
// selected and copied, but not typed into. data is copied.
func NewResponseEditor(data []byte, hl highlight.Highlighter, opts ...ui.EditorOption) *ui.Editor {
	opts = append([]ui.EditorOption{ui.WithReadOnly(), ui.WithSoftWrap(true)}, opts...)
	return StyleCodeEditor(ui.NewEditor(data, hl, opts...))
}

// StyleCodeEditor sets up how a code editor lays out its text: wrapped rows
// hang one tab stop past their line's indent, so nested JSON keeps its shape,
// and the line-number gutter is only as wide as its digits.
func StyleCodeEditor(ed *ui.Editor) *ui.Editor {
	ed.SetWrapIndent(ui.WrapHanging)
	ed.SetGutter(ui.GutterSpec{MinDigits: 2, Fill: ui.TokenChromeMuted, Separator: true})
	return ed
}

// ReplaceEditor closes the previous editor and returns a new read-only one
// with data. data is copied, so the caller may keep using it.
func ReplaceEditor(old *ui.Editor, data []byte, hl highlight.Highlighter) *ui.Editor {
	if old != nil {
		old.Close()
	}
	return NewResponseEditor(data, hl)
}

// ReplaceResponseEditor rebuilds the response body editor for res.
//
// The editor is built over the display bytes without copying them. Response
// bodies and their formatted forms are never modified after the response is
// built, so the editor can read them in place; for a large body that saves a
// full copy of the document per tab.
func ReplaceResponseEditor(old *ui.Editor, res *egress.Response, raw bool) *ui.Editor {
	if old != nil {
		old.Close()
	}
	var kind string
	if res != nil {
		kind = res.BodyKind
	}
	return NewResponseEditor(DisplayBody(res, raw), BodyHighlighter(kind), ui.WithSharedContent())
}

// FailedStatus is the status line text for a request that failed; the error
// itself is shown in the response pane by ErrorView.
const FailedStatus = "Request failed"

// ErrorView shows a failed request: the confused Chapar with msg under it in
// red. Right-click copies the message.
func ErrorView(id string, th *theme.Theme, ctx *ui.Ctx, deps Deps, msg string) ui.View {
	return ui.Column(
		ui.Column(
			ui.Image(id+"-confused", assets.ChaparConfusedPNG).Width(160),
		).Align(ui.AlignCenter).PaddingTop(th.Spacing.L),
		ui.ContextMenu(id+"-menu", ui.Paragraph(msg).
			TextAlign(ui.AlignCenter).
			Style(ui.Spec{}.TextColor(ui.TokenError)),
			[]ui.MenuItem{{Label: "Copy", OnSelect: func() {
				if clip := ctx.Clipboard(); clip != nil {
					clip.Set(msg)
					deps.Toast("Copied")
				}
			}}}),
	).Gap(th.Spacing.M).Grow(1)
}

// PostRequestNotice shows why the post-request actions failed, above a
// response that still arrived. It is nil when they did not fail. Right-click
// copies the message.
func PostRequestNotice(id string, ctx *ui.Ctx, deps Deps, res *egress.Response) ui.View {
	if res == nil || res.PostRequestError == nil {
		return nil
	}
	msg := "Post-request failed: " + res.PostRequestError.Error() + " (see Timeline)"
	return ui.ContextMenu(id+"-post-err", ui.Paragraph(msg).
		Style(ui.Spec{}.TextColor(ui.TokenError)),
		[]ui.MenuItem{{Label: "Copy", OnSelect: func() {
			if clip := ctx.Clipboard(); clip != nil {
				clip.Set(msg)
				deps.Toast("Copied")
			}
		}}})
}

// ResponseTabsRow lays out response tabs with a stable Raw checkbox on the right
// so switching tabs does not change chrome height.
func ResponseTabsRow(id string, th *theme.Theme, tabs []ui.TabModel, selected int, onSelect func(i int, title string), raw bool, onRawChange func(raw bool)) ui.View {
	return ui.Row(
		ui.Tabs(id+"-tabs", tabs).
			Selected(selected).
			Closable(false).
			OnSelectItem(onSelect).
			TabBackground(th.Background).
			Grow(1),
		ui.Checkbox(id+"-raw", "Raw").
			Check(raw).
			OnToggle(func(v bool) { onRawChange(v) }),
	).Gap(th.Spacing.S).Align(ui.AlignCenter)
}

// ResponseEditorMenu adds Save… to the editor's right-click menu and makes
// Copy take the whole body when nothing is selected.
func ResponseEditorMenu(ed *ui.Editor, ctx *ui.Ctx, deps Deps, defaultName string) ui.View {
	if ed == nil {
		return ui.Text("").Grow(1)
	}
	ed.ContextMenu = func(items []ui.MenuItem) []ui.MenuItem {
		for i := range items {
			if items[i].Label != "Copy" {
				continue
			}
			items[i].Disabled = false
			items[i].OnSelect = func() {
				if !ed.Copy() {
					clip := ctx.Clipboard()
					if clip == nil {
						return
					}
					clip.Set(string(ed.Bytes()))
				}
				deps.Toast("Copied")
			}
		}
		return append(items, ui.MenuSeparator, ui.MenuItem{Label: "Save…", OnSelect: func() {
			SaveEditorBytes(deps, defaultName, ed.Bytes())
		}})
	}
	if ed.HighlightOversize() {
		return ui.Column(HighlightOffNotice(ed), ui.ViewOf(ed).Grow(1)).
			Gap(ctx.Theme().Spacing.XS).Grow(1)
	}
	return ui.ViewOf(ed).Grow(1)
}

// HighlightOffNotice tells the reader why ed shows no syntax colors — its
// document is over the size limit — and offers to highlight it anyway.
func HighlightOffNotice(ed *ui.Editor) ui.View {
	th := theme.Current()
	msg := fmt.Sprintf("Syntax highlighting is off: this body is %s, over the %s limit (Settings › Editor).",
		humanize.IBytes(uint64(len(ed.Bytes()))), humanize.IBytes(uint64(highlight.MaxBytes)))
	return ui.Row(
		ui.Icon(icons.Info, th.Metrics.IconSizeSM, th.Info),
		ui.Paragraph(msg).Size(th.Typography.Caption.Size).Grow(1),
		ui.Button(fmt.Sprintf("hl-anyway-%p", ed), ui.Text("Highlight anyway")).
			Subtle().
			Tooltip("Color this body; large documents take more memory and time").
			OnClick(ed.HighlightAnyway),
	).Gap(th.Spacing.S).
		PaddingXY(th.Spacing.S, th.Spacing.XS).
		Background(ui.TokenInfoSurface).
		Style(ui.Spec{}.Radius(th.Radius.Medium))
}

// SaveEditorBytes opens a save dialog and writes data to the chosen path.
func SaveEditorBytes(deps Deps, defaultName string, data []byte) {
	if deps.Files == nil {
		return
	}
	fd := deps.Files()
	if fd == nil {
		return
	}
	if defaultName == "" {
		defaultName = "response.txt"
	}
	ext := filepath.Ext(defaultName)
	filters := []ui.FileFilter{{Label: "All files", Exts: nil}}
	if ext != "" {
		filters = []ui.FileFilter{
			{Label: strings.ToUpper(strings.TrimPrefix(ext, ".")) + " files", Exts: []string{ext}},
			{Label: "All files", Exts: nil},
		}
	}
	fd.Show(ui.FileDialogOpts{
		Title:          "Save response",
		Mode:           ui.FileDialogSaveFile,
		Filters:        filters,
		ShowSaveFilter: true,
		OnConfirm: func(paths []string) {
			if len(paths) == 0 {
				return
			}
			if err := os.WriteFile(paths[0], data, 0o644); err != nil {
				deps.ShowError(err)
				return
			}
			deps.Toast("Saved")
		},
	})
}

// DefaultResponseFilename picks a filename from body kind.
func DefaultResponseFilename(kind string) string {
	switch kind {
	case util.BodyKindJSON:
		return "response.json"
	case util.BodyKindXML:
		return "response.xml"
	case util.BodyKindHTML:
		return "response.html"
	default:
		return "response.txt"
	}
}

// TimelineState holds selection and a retained table for the Timeline tab.
type TimelineState struct {
	Selected int
	table    *ui.Table
	steps    []egress.TimelineStep
	starts   []time.Duration // when each step began, as an offset from the first
	total    time.Duration   // when the last step ended

	detailTitle    string
	detailSubtitle string
	detailBody     string
}

// Timeline table column IDs.
const (
	tlColStep   = "step"
	tlColPhase  = "phase"
	tlColSpan   = "span"
	tlColLength = "dur"
)

// Ensure constructs the table once.
func (t *TimelineState) Ensure() {
	if t.table != nil {
		return
	}
	// Each step's bar sits on one shared scale, so the table reads as a
	// waterfall: where in the request each step happened, and how long it
	// took.
	t.table = ui.NewTable([]ui.TableColumn{
		{ID: tlColStep, Label: "Step", Kind: ui.TableColText, Width: 130},
		{ID: tlColPhase, Label: "Phase", Kind: ui.TableColText, Width: 80},
		{ID: tlColSpan, Label: "Timing", Kind: ui.TableColText},
		{ID: tlColLength, Label: "Duration", Kind: ui.TableColText, Width: 90, Locked: true},
	}, nil)
	t.table.Selectable = true
	t.table.Editable = false
	t.table.OnRowClick = func(rowID string) {
		for i := range t.steps {
			if timelineRowID(i) == rowID {
				t.selectStep(i)
				return
			}
		}
	}
	t.table.CellView = func(row ui.TableRow, col ui.TableColumn) ui.View {
		if col.ID != tlColSpan {
			return nil
		}
		i := -1
		for k := range t.steps {
			if timelineRowID(k) == row.ID {
				i = k
				break
			}
		}
		if i < 0 {
			return nil
		}
		step := t.steps[i]
		from := t.starts[i]
		// A step too short to see still gets a sliver, and an empty
		// timeline a scale, so every row shows a bar.
		total := max(t.total, time.Microsecond)
		bar := ui.RangeBar("tl-span-"+row.ID, float32(from), float32(from+step.Duration), float32(total)).
			Tooltip(fmt.Sprintf("%s %s", step.Name, step.Duration.Round(time.Microsecond)))
		if step.Err != "" {
			bar = bar.Tone(ui.BadgeError)
		} else if h, ok := stepHue(step); ok {
			bar = bar.Hue(h)
		}
		return ui.Row(bar.Grow(1)).Align(ui.AlignCenter).Grow(1)
	}
}

// EnsureTimelineDetail is kept for callers that used the old editor API.
func (t *TimelineState) EnsureTimelineDetail() { t.Ensure() }

// Close releases timeline widgets.
func (t *TimelineState) Close() {
	t.table = nil
	t.steps = nil
	t.starts = nil
	t.total = 0
	t.Selected = -1
	t.detailTitle = ""
	t.detailSubtitle = ""
	t.detailBody = ""
}

// SetSteps refreshes selection, table rows, and detail card from the response timeline.
func (t *TimelineState) SetSteps(steps []egress.TimelineStep) {
	t.Ensure()
	t.steps = append([]egress.TimelineStep(nil), steps...)
	// Steps run one after another, so each starts where the previous ended.
	t.starts = make([]time.Duration, len(t.steps))
	t.total = 0
	for i, step := range t.steps {
		t.starts[i] = t.total
		t.total += step.Duration
	}
	if len(t.steps) == 0 {
		t.Selected = -1
		t.table.SetRows(nil)
		t.detailTitle = "No timeline"
		t.detailSubtitle = ""
		t.detailBody = "Send a request to see steps."
		return
	}
	if t.Selected < 0 || t.Selected >= len(t.steps) {
		t.Selected = 0
	}
	rows := make([]ui.TableRow, len(t.steps))
	for i, step := range t.steps {
		phase := step.Phase
		if phase == "" {
			phase = "—"
		}
		rows[i] = ui.TableRow{
			ID: timelineRowID(i),
			Cells: map[string]string{
				tlColStep:   step.Name,
				tlColPhase:  phase,
				tlColLength: stepDuration(step.Duration),
			},
		}
	}
	t.table.SetRows(rows)
	t.selectStep(t.Selected)
}

// selectStep highlights step i and shows its detail.
func (t *TimelineState) selectStep(i int) {
	t.Selected = i
	id := timelineRowID(i)
	for k := range t.table.Rows {
		t.table.Rows[k].Selected = t.table.Rows[k].ID == id
	}
	t.showDetail(t.steps[i])
}

// stepDuration shows d in milliseconds, with decimals below 10 ms so a
// fast step does not read as zero.
func stepDuration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	if ms < 10 {
		return fmt.Sprintf("%.2f ms", ms)
	}
	return fmt.Sprintf("%.0f ms", ms)
}

func timelineRowID(i int) string { return fmt.Sprintf("step-%d", i) }

// stepHue colors a step's bar by what it waits on: the scripts, connection
// setup in cool hues, the server in orange, and the transfer in green.
func stepHue(step egress.TimelineStep) (theme.Hue, bool) {
	if step.Phase == egress.TimelinePhaseApp {
		return theme.HuePink, true
	}
	switch step.Name {
	case "DNS Lookup":
		return theme.HueTeal, true
	case "TCP Connect", "Connect":
		return theme.HueBlue, true
	case "TLS Handshake":
		return theme.HuePurple, true
	case "Request Sent":
		return theme.HueYellow, true
	case "Waiting (TTFB)", "Waiting":
		return theme.HueOrange, true
	case "Download", "Receive":
		return theme.HueGreen, true
	}
	return 0, false
}

func (t *TimelineState) showDetail(step egress.TimelineStep) {
	t.detailTitle = step.Name
	phase := step.Phase
	if phase == "" {
		phase = "—"
	}
	t.detailSubtitle = fmt.Sprintf("%s · %s", phase, step.Duration.Round(time.Microsecond))
	var b strings.Builder
	if step.Err != "" {
		fmt.Fprintf(&b, "Error: %s\n", step.Err)
	}
	if step.Detail != "" {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(step.Detail)
	}
	if b.Len() == 0 {
		b.WriteString("No additional detail.")
	}
	t.detailBody = strings.TrimRight(b.String(), "\n")
}

// TimelineView renders a waterfall table of steps with a detail card below.
func TimelineView(id string, th *theme.Theme, ctx *ui.Ctx, deps Deps, state *TimelineState, steps []egress.TimelineStep) ui.View {
	state.Ensure()
	if len(steps) == 0 && len(state.steps) == 0 {
		return ui.Column(
			MutedParagraph("No timeline yet. Send a request to see steps."),
		).Gap(th.Spacing.S).Grow(1)
	}
	if !timelineStepsEqual(steps, state.steps) {
		state.SetSteps(steps)
	}
	if len(state.steps) == 0 {
		return ui.Column(
			MutedParagraph("No timeline yet. Send a request to see steps."),
		).Gap(th.Spacing.S).Grow(1)
	}

	// The detail wraps and can be selected and copied: a script error or
	// traceback is often longer than the pane is wide.
	body := ui.Paragraph(state.detailBody).Selectable(id + "-detail").
		Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))
	card := ui.Card(state.detailTitle, state.detailSubtitle, body).Flat()

	return ui.Column(
		ui.Caption("Steps"),
		ui.ViewOf(state.table).Height(timelineTableHeight(th, len(state.steps))).Grow(0),
		// The detail sits at the bottom of the pane, as tall as its text;
		// it scrolls once that is taller than the room below the steps.
		ui.Column(ui.Scroll(id+"-detail-scroll", card).FitContent()).
			Grow(1).
			Justify(ui.JustifyEnd),
	).Gap(th.Spacing.S).Grow(1)
}

// timelineTableHeight fits the header and up to timelineVisibleSteps rows,
// so the detail below gets the rest of the pane; the table scrolls past that.
func timelineTableHeight(th *theme.Theme, n int) float32 {
	n = min(n, timelineVisibleSteps)
	return float32(n+1) * th.Metrics.ControlHeight
}

const timelineVisibleSteps = 4

func timelineStepsEqual(a, b []egress.TimelineStep) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].Duration != b[i].Duration || a[i].Phase != b[i].Phase || a[i].Err != b[i].Err || a[i].Detail != b[i].Detail {
			return false
		}
	}
	return true
}
