package testcase

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/testrun"
	"github.com/chapar-rest/chapar/ui/container"
)

func (c *Container) resultsPane(ctx *ui.Ctx, th *theme.Theme) ui.View {
	id := c.ID()
	if c.run == nil {
		msg := "Run the test case to see how each step does."
		if c.deps.Tests == nil {
			msg = "Running test cases is not available."
		}
		return ui.EmptyState("No results yet", msg).Grow(1)
	}

	list := make([]ui.View, 0, len(c.run.Steps))
	for i, s := range c.run.Steps {
		list = append(list, c.resultRow(th, i, s))
	}
	if c.running {
		list = append(list, ui.Row(ui.Spinner("tc-spin-"+id, 14), ui.Caption("Running…")).
			Gap(th.Spacing.S).Align(ui.AlignCenter).PaddingXY(th.Spacing.S, th.Spacing.XS))
	}

	return ui.Column(
		c.summary(th),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Splitter("tc-res-split-"+id, ui.Vertical,
			ui.Scroll("tc-res-list-"+id, ui.Column(list...).Gap(th.Spacing.XXS).Padding(th.Spacing.XS)),
			c.detail(ctx, th),
		).Percents(40, 60).HandleOnHover().Grow(1),
	).Grow(1)
}

func (c *Container) summary(th *theme.Theme) ui.View {
	run := c.run
	var badge ui.View
	switch run.Status {
	case testrun.StatusPassed:
		badge = ui.Badge("Passed").Tone(ui.BadgeSuccess)
	case testrun.StatusFailed:
		badge = ui.Badge("Failed").Tone(ui.BadgeError)
	case testrun.StatusError:
		badge = ui.Badge("Error").Tone(ui.BadgeError)
	case testrun.StatusCancelled:
		badge = ui.Badge("Stopped").Tone(ui.BadgeWarning)
	default:
		badge = ui.Badge("Running").Tone(ui.BadgeAccent)
	}

	counts := run.Counts()
	var parts []string
	for _, st := range []testrun.Status{testrun.StatusPassed, testrun.StatusFailed, testrun.StatusError, testrun.StatusSkipped} {
		if counts[st] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[st], st))
		}
	}
	info := strings.Join(parts, " · ")
	if !c.running {
		info += " · " + formatDuration(run.Duration)
	}
	if run.EnvName != "" {
		info += " · env " + run.EnvName
	}

	failed := c.failedStepIDs()
	return ui.Row(
		badge,
		ui.Caption(info),
		ui.Spacer(),
		ui.Button("tc-rerun-"+c.ID(), ui.Text("Re-run failed")).IconStart(icons.RotateCcw).
			Disabled(c.running || len(failed) == 0).
			OnClick(func() { c.start(failed) }),
	).Gap(th.Spacing.S).Align(ui.AlignCenter).PaddingXY(th.Spacing.M, th.Spacing.S)
}

// failedStepIDs are the steps of the last run that did not pass.
func (c *Container) failedStepIDs() []string {
	var out []string
	for _, s := range c.run.Steps {
		if s.Section == testrun.SectionSteps && (s.Status == testrun.StatusFailed || s.Status == testrun.StatusError) {
			out = append(out, s.StepID)
		}
	}
	return out
}

func (c *Container) resultRow(th *theme.Theme, i int, s testrun.StepResult) ui.View {
	icon, color := statusLook(th, &s)
	cells := []ui.View{ui.Icon(icon, 16, color), ui.Text(s.Name).Ellipsis(ui.EllipsisEnd).Grow(1)}
	if s.Section != testrun.SectionSteps {
		cells = append(cells, ui.Badge(s.Section))
	}
	if s.Response != nil {
		cells = append(cells, ui.Caption(fmt.Sprint(s.Response.Status)))
	}
	if s.Status != testrun.StatusSkipped && s.Status != testrun.StatusRunning {
		cells = append(cells, ui.Caption(formatDuration(s.Duration)))
	}
	if s.Attempts > 1 {
		cells = append(cells, ui.Caption(fmt.Sprintf("×%d", s.Attempts)))
	}
	b := ui.Button(fmt.Sprintf("tc-res-%s-%d", c.ID(), i),
		ui.Row(cells...).Gap(th.Spacing.S).Align(ui.AlignCenter).Grow(1),
	).Ghost().HoverFill().OnClick(func() { c.selected = i })
	if i == c.selected {
		b = b.Background(ui.TokenListActive)
	}
	return b.Grow(1)
}

func (c *Container) detail(ctx *ui.Ctx, th *theme.Theme) ui.View {
	if c.selected < 0 || c.selected >= len(c.run.Steps) {
		return ui.EmptyState("", "Select a step to see what it sent, got and checked.").Grow(1)
	}
	s := c.run.Steps[c.selected]
	id := c.ID()
	rows := []ui.View{}

	title := ui.Row(ui.Strong(s.Name))
	if r := s.Request; r != nil {
		title = ui.Row(ui.Strong(s.Name), ui.Caption(strings.TrimSpace(r.Method+" "+r.URL)).Ellipsis(ui.EllipsisEnd)).
			Gap(th.Spacing.S).Align(ui.AlignCenter)
	}
	rows = append(rows, title)

	if s.Message != "" && s.Status != testrun.StatusPassed {
		variant := ui.AlertError
		if s.Status == testrun.StatusSkipped || s.Status == testrun.StatusCancelled {
			variant = ui.AlertInfo
		}
		rows = append(rows, ui.Alert(s.Message, variant))
	}

	for _, a := range s.Assertions {
		rows = append(rows, assertionLine(th, a))
	}
	for _, cp := range s.Captures {
		if cp.Error != "" {
			rows = append(rows, ui.Row(ui.Icon(icons.CircleX, 14, th.Error), ui.Text("capture "+cp.Var+": "+cp.Error)).
				Gap(th.Spacing.XS).Align(ui.AlignCenter))
			continue
		}
		rows = append(rows, ui.Row(ui.Icon(icons.ArrowDown, 14, th.ForegroundMuted),
			ui.Text(cp.Var+" = "+cp.Value).Ellipsis(ui.EllipsisEnd)).Gap(th.Spacing.XS).Align(ui.AlignCenter))
	}

	if r := s.Response; r != nil {
		info := fmt.Sprintf("Response %d · %s · %s", r.Status, container.FormatBytes(r.Size), formatDuration(r.Time))
		if r.Truncated {
			info += " · body cut at 64 kB"
		}
		rows = append(rows, ui.Caption(info))
		key := fmt.Sprintf("%s/%s/%d", s.Section, s.StepID, s.Attempts)
		if c.respEd == nil || c.respFor != key {
			c.respEd = container.ReplaceEditor(c.respEd, prettyBody([]byte(r.Body)), bodyHighlighter([]byte(r.Body)))
			c.respFor = key
		}
		rows = append(rows, ui.ViewOf(c.respEd).Height(320).Border(ui.TokenBorder, th.Stroke.Thin))
	}
	return ui.Scroll("tc-detail-"+id, ui.Column(rows...).Gap(th.Spacing.S).Padding(th.Spacing.M)).Grow(1)
}

func assertionLine(th *theme.Theme, a testrun.AssertionResult) ui.View {
	icon, color := icons.CircleCheck, th.Success
	if !a.Passed {
		icon, color = icons.CircleX, th.Error
	}
	text := describe(a)
	line := ui.Row(ui.Icon(icon, 14, color), ui.Text(text).Ellipsis(ui.EllipsisEnd).Grow(1)).
		Gap(th.Spacing.XS).Align(ui.AlignCenter)
	if a.Passed || a.Message == "" {
		return line
	}
	return ui.Column(line, ui.Caption(a.Message).PaddingLeft(th.Spacing.L)).Gap(th.Spacing.XXS)
}

// describe says what an assertion checked.
func describe(a testrun.AssertionResult) string {
	if a.Source == testrun.SourceScript {
		return "script: " + a.Path
	}
	subject := a.Target
	if a.Key != "" {
		subject += " " + a.Key
	}
	if a.Path != "" {
		subject += " " + a.Path
	}
	op := a.Op
	for _, o := range opOptions {
		if o.Value == a.Op {
			op = o.Label
		}
	}
	if a.Expected == nil {
		return subject + " " + op
	}
	return subject + " " + op + " " + valueText(a.Expected)
}

func prettyBody(b []byte) []byte {
	var v any
	if json.Unmarshal(b, &v) != nil {
		return b
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return b
	}
	return out
}

func bodyHighlighter(b []byte) highlight.Highlighter {
	if json.Valid(b) {
		return highlight.NewJSON()
	}
	return highlight.Noop{}
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return "<1ms"
	case d < time.Second:
		return d.Round(time.Millisecond).String()
	}
	return d.Round(10 * time.Millisecond).String()
}
