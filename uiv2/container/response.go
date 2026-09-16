package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/util"
	"github.com/dustin/go-humanize"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
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
	if res.IsJSON && res.JSON != "" {
		return []byte(res.JSON)
	}
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

// ReplaceEditor closes the previous editor and returns a new one with data.
func ReplaceEditor(old *ui.Editor, data []byte, hl highlight.Highlighter) *ui.Editor {
	if old != nil {
		old.Close()
	}
	return ui.NewEditor(data, hl, ui.WithSoftWrap(true))
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

// ResponseEditorMenu wraps an editor with Select All / Copy / Save context menu.
func ResponseEditorMenu(id string, ed *ui.Editor, ctx *ui.Ctx, deps Deps, defaultName string) ui.View {
	if ed == nil {
		return ui.Text("").Grow(1)
	}
	// Do not set Width on ContextMenu — that sizes the wrapper (and shrinks the
	// editor). Menu width defaults to 180 inside layoutContextMenu.
	primary := input.ModCtrl | input.ModSuper
	return ui.ContextMenu(id+"-menu", ui.ViewOf(ed).Grow(1), []ui.MenuItem{
		{Label: "Select All", OnSelect: func() {
			ed.HandleKeys([]input.KeyEvent{{Key: input.KeyA, Mods: primary}})
		}},
		{Label: "Copy", OnSelect: func() {
			clip := ctx.Clipboard()
			if clip == nil {
				return
			}
			// Prefer selection (Cmd/Ctrl+C). If nothing was selected, copy() is a
			// no-op — fall back to the whole buffer (former CopySelection).
			prev := clip.Get()
			ed.HandleKeys([]input.KeyEvent{{Key: input.KeyC, Mods: primary}})
			if clip.Get() == prev {
				clip.Set(string(ed.Bytes()))
			}
			deps.Toast("Copied")
		}},
		{Label: "Save…", OnSelect: func() {
			SaveEditorBytes(deps, defaultName, ed.Bytes())
		}},
	}).Grow(1)
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

// TimelineState holds selection and a retained ListView for the Timeline tab.
type TimelineState struct {
	Selected int
	hover    int
	List     *ui.ListView
	steps    []egress.TimelineStep
	wake     func()

	detailTitle    string
	detailSubtitle string
	detailBody     string
}

// Ensure constructs the list once.
func (t *TimelineState) Ensure() {
	if t.List == nil {
		th := theme.Current()
		t.List = ui.NewListView(ui.ListViewConfig{Gap: th.Spacing.XXS})
		t.hover = -1
	}
}

// EnsureTimelineDetail is kept for callers that used the old editor API.
func (t *TimelineState) EnsureTimelineDetail() { t.Ensure() }

// Close releases timeline widgets.
func (t *TimelineState) Close() {
	if t.List != nil {
		t.List.Clear()
		t.List = nil
	}
	t.steps = nil
	t.hover = -1
	t.Selected = -1
	t.detailTitle = ""
	t.detailSubtitle = ""
	t.detailBody = ""
}

// SetSteps refreshes selection, list rows, and detail card from the response timeline.
func (t *TimelineState) SetSteps(steps []egress.TimelineStep) {
	t.Ensure()
	t.steps = append([]egress.TimelineStep(nil), steps...)
	t.hover = -1
	if len(t.steps) == 0 {
		t.Selected = -1
		t.List.Clear()
		t.detailTitle = "No timeline"
		t.detailSubtitle = ""
		t.detailBody = "Send a request to see steps."
		return
	}
	if t.Selected < 0 || t.Selected >= len(t.steps) {
		t.Selected = 0
	}
	t.rebuildList()
	t.showDetail(t.steps[t.Selected])
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

func (t *TimelineState) rebuildList() {
	t.Ensure()
	th := theme.Current()
	rowH := th.Metrics.ControlHeight
	items := make([]*layout.Element, 0, len(t.steps))
	for i := range t.steps {
		items = append(items, t.makeRow(i, rowH))
	}
	t.List.SetItems(items)
}

func (t *TimelineState) makeRow(i int, rowH float32) *layout.Element {
	el := layout.New(layout.Box().H(rowH).FlexShrink(0))
	el.Paint = func(dl *render.DrawList, text *shape.Engine) {
		if i < 0 || i >= len(t.steps) {
			return
		}
		step := t.steps[i]
		th := theme.Current()
		fr := el.Frame
		switch {
		case t.Selected == i:
			dl.AddRect(fr, th.ListActive)
		case t.hover == i:
			dl.AddRect(fr, th.ListHover)
		}
		pad := th.Spacing.S
		style := th.Typography.Body
		label := fmt.Sprintf("%s  %s", step.Name, step.Duration.Round(time.Millisecond))
		_, lh := text.MeasureAt(label, style.Size)
		ty := fr.Y + (fr.H-lh)/2
		text.DrawStringTopAt(dl, label, fr.X+pad, ty, th.Foreground, style.Size)
		if phase := step.Phase; phase != "" {
			cap := th.Typography.Caption
			tw, _ := text.MeasureAt(phase, cap.Size)
			text.DrawStringTopAt(dl, phase, fr.X+fr.W-pad-tw, ty, th.ForegroundMuted, cap.Size)
		}
	}
	el.OnMouse = func(e *layout.Element, m *input.Mouse) {
		if !e.Frame.Contains(m.X, m.Y) {
			if t.hover == i {
				t.hover = -1
				t.invalidate()
			}
			return
		}
		m.SetCursor(input.CursorPointer)
		if t.hover != i {
			t.hover = i
			t.invalidate()
		}
		if m.Released && i >= 0 && i < len(t.steps) {
			t.Selected = i
			t.showDetail(t.steps[i])
			t.invalidate()
			m.Consumed = true
		}
	}
	return el
}

func (t *TimelineState) invalidate() {
	if t.wake != nil {
		t.wake()
	}
}

// TimelineView renders a ListView of steps with a detail card below.
func TimelineView(id string, th *theme.Theme, ctx *ui.Ctx, deps Deps, state *TimelineState, steps []egress.TimelineStep) ui.View {
	state.Ensure()
	state.wake = func() {
		ctx.MarkNeedsPaint()
		deps.WakeNow()
	}
	if len(steps) == 0 && len(state.steps) == 0 {
		return ui.Column(
			ui.Muted("No timeline yet. Send a request to see steps."),
		).Gap(th.Spacing.S).Grow(1)
	}
	if !timelineStepsEqual(steps, state.steps) {
		state.SetSteps(steps)
	}
	if len(state.steps) == 0 {
		return ui.Column(
			ui.Muted("No timeline yet. Send a request to see steps."),
		).Gap(th.Spacing.S).Grow(1)
	}

	bodyLines := strings.Split(state.detailBody, "\n")
	parts := make([]ui.View, 0, len(bodyLines))
	for _, line := range bodyLines {
		parts = append(parts, ui.Muted(line))
	}
	card := ui.Card(state.detailTitle, state.detailSubtitle, ui.Column(parts...).Gap(th.Spacing.XXS)).
		Flat().
		Grow(1)

	return ui.Column(
		ui.Caption("Steps"),
		ui.ViewOf(state.List).Height(160),
		ui.Scroll(id+"-detail", card).Grow(1),
	).Gap(th.Spacing.S).Grow(1)
}

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
