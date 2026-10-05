package testcase

import (
	"fmt"
	"strings"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/testrun"
	"github.com/chapar-rest/chapar/ui/container"
	"github.com/chapar-rest/chapar/ui/vars"
)

// maxCardWidth keeps a card's rows readable in a wide pane.
const maxCardWidth = 760

// compactCardWidth is the card width below which the header shows Run as
// an icon and each assertion and capture takes two lines.
const compactCardWidth = 620

func (c *Container) compact() bool { return c.cardW > 0 && c.cardW < compactCardWidth }

// stepCard is one step: a header that is always shown, and a body with
// its assertions, captures, execution settings and request overrides.
func (c *Container) stepCard(ctx *ui.Ctx, th *theme.Theme, i int, m *stepModel) ui.View {
	// The border is drawn on the card's edge, so the content sits one
	// stroke inside it, and the header's corners follow the border's inner
	// curve instead of covering it.
	radius := th.Radius.XLarge
	inner := radius - th.Stroke.Thin
	header := c.cardHeader(th, i, m).
		Height(48).
		PaddingXY(th.Spacing.S, 0).
		Background(ui.TokenChrome).
		RadiusTopLeft(inner).
		RadiusTopRight(inner)
	rows := []ui.View{header}
	if m.open {
		header.RadiusBottomLeft(0).RadiusBottomRight(0)
		rows = append(rows, ui.HLine(th.Stroke.Thin, th.Border), c.cardBody(ctx, th, m))
	} else {
		header.RadiusBottomLeft(inner).RadiusBottomRight(inner)
	}
	card := ui.Column(rows...).
		Background(ui.TokenSurface).
		Border(ui.TokenBorder, th.Stroke.Thin).
		Radius(radius).
		Padding(th.Stroke.Thin).
		MaxWidth(maxCardWidth).
		Layout(ctx)
	if i == 0 {
		c.measure(ctx, card)
	}
	return ui.Raw(card)
}

func (c *Container) cardHeader(th *theme.Theme, i int, m *stepModel) *ui.Node {
	k := m.key
	n := len(c.sections[c.section])
	chevron := icons.ChevronRight
	if m.open {
		chevron = icons.ChevronDown
	}
	opts, byID := c.requestOptions()
	opts, sel := c.selectedRequest(m, opts)
	compact := c.compact()
	reqWidth := float32(240)
	if compact {
		reqWidth = 180
	}

	canRun := !c.running && c.section == testrun.SectionSteps && c.deps.Tests != nil && !m.disabled
	runTip := "Run this step, with setup and teardown"
	if c.section != testrun.SectionSteps {
		runTip = "Setup and teardown run with the steps"
	}
	run := ui.View(ui.Button("tc-run-step-"+k, ui.Text("Run")).Primary().IconStart(icons.Play).
		Tooltip(runTip).Disabled(!canRun).OnClick(func() { c.start([]string{m.id}) }))
	if compact {
		// A different id: widget state is kept per id, and a Button's
		// is not an IconButton's.
		run = ui.IconButton("tc-run-step-icon-"+k, icons.Play).Tooltip(runTip).Disabled(!canRun).
			OnClick(func() { c.start([]string{m.id}) })
	}

	name := ui.EditableLabel("tc-name-"+k, m.name).
		Placeholder("Untitled step").
		OnSave(func(s string) {
			if s = strings.TrimSpace(s); s != m.name {
				m.name = s
				c.markDirty()
			}
		}).
		// A long name is cut short rather than pushing the controls on its
		// right out of a narrow card.
		Ellipsis(ui.EllipsisEnd).
		Grow(1)
	cells := []ui.View{
		ui.IconButton("tc-open-"+k, chevron).Tooltip("Show or hide the step").OnClick(func() { m.open = !m.open }),
		c.statusIcon(th, m),
		name,
	}
	// A narrow card keeps its room for the name; the icon and the dimmed
	// Run still say the step is off.
	if m.disabled && !compact {
		cells = append(cells, ui.Badge("Disabled"))
	}
	cells = append(cells,
		ui.Select("tc-req-"+k, opts).Width(reqWidth).Selected(sel).OnChange(func(v string) {
			if r := byID[v]; r != nil {
				m.request = domain.TestRequestRef{ID: r.MetaData.ID, Ref: testrun.RefOf(r)}
				// A step still named by default takes its request's name.
				if autoStepID.MatchString(m.id) {
					taken := c.stepIDs()
					delete(taken, m.id)
					m.id = uniqueStepID(r.MetaData.Name, taken)
				}
				c.markDirty()
			}
		}),
		run,
		c.stepMenu(i, n, m),
	)
	return ui.Row(cells...).Gap(th.Spacing.S).Align(ui.AlignCenter)
}

// stepMenu holds the actions that are not in the header.
func (c *Container) stepMenu(i, n int, m *stepModel) ui.View {
	disable := ui.MenuItem{Label: "Disable", OnSelect: func() { m.disabled = true; c.markDirty() }}
	if m.disabled {
		disable = ui.MenuItem{Label: "Enable", OnSelect: func() { m.disabled = false; c.markDirty() }}
	}
	return ui.IconButton("tc-more-"+m.key, icons.Ellipsis).Menu([]ui.MenuItem{
		{Label: "Move up", Disabled: i == 0, OnSelect: func() { c.moveStep(i, -1) }},
		{Label: "Move down", Disabled: i == n-1, OnSelect: func() { c.moveStep(i, 1) }},
		{Label: "Duplicate", OnSelect: func() { c.duplicateStep(i) }},
		disable,
		ui.MenuSeparator,
		{Label: "Delete", OnSelect: func() { c.removeStep(i) }},
	})
}

// measure keeps the width the step cards were laid out at. Layout only
// knows it after the views are built, so a change shows on the next frame.
func (c *Container) measure(ctx *ui.Ctx, el *layout.Element) {
	prev := el.AfterLayout
	el.AfterLayout = func(e *layout.Element) bool {
		// Frame is filled only after this hook; the solved size is known.
		w, _ := e.LayoutSize()
		if (w < compactCardWidth) != (c.cardW < compactCardWidth) || c.cardW == 0 {
			// Not WakeNow: a repaint asked for during layout is dropped once
			// this frame presents. Animate wakes the next frame.
			ctx.Animate(0)
		}
		c.cardW = w
		if prev != nil {
			return prev(e)
		}
		return false
	}
}

// statusIcon shows how the step did in the last run.
func (c *Container) statusIcon(th *theme.Theme, m *stepModel) ui.View {
	if m.disabled {
		return ui.Icon(icons.CircleSlash, 14, th.ForegroundSubtle)
	}
	r := c.results[resultKey(c.section, m.id)]
	if r != nil && r.Status == testrun.StatusRunning {
		return ui.Spinner("tc-spin-"+m.key, 14)
	}
	icon, color := statusLook(th, r)
	return ui.Icon(icon, 14, color)
}

func statusLook(th *theme.Theme, r *testrun.StepResult) (icons.Icon, render.Color) {
	if r == nil {
		return icons.CircleDashed, th.ForegroundSubtle
	}
	switch r.Status {
	case testrun.StatusPassed:
		return icons.CircleCheck, th.Success
	case testrun.StatusFailed, testrun.StatusError:
		return icons.CircleX, th.Error
	case testrun.StatusRunning:
		return icons.LoaderCircle, th.Accent
	case testrun.StatusCancelled:
		return icons.CircleStop, th.Warning
	}
	return icons.CircleDashed, th.ForegroundSubtle
}

func (c *Container) cardBody(ctx *ui.Ctx, th *theme.Theme, m *stepModel) ui.View {
	return ui.Column(
		c.assertions(th, m),
		c.captures(th, m),
		ui.Column(
			c.execution(th, m),
			c.overrides(ctx, th, m),
		).Gap(th.Spacing.XS),
	).Gap(th.Spacing.XXL).Padding(th.Spacing.L)
}

// section is the header of Assertions and Captures: a title, a count, a
// hint that shows how variables are written, and Add.
func section(th *theme.Theme, id, title string, count int, hint, token string, add func()) ui.View {
	return ui.Row(
		ui.Strong(title),
		ui.Badge(fmt.Sprint(count)),
		ui.Row(
			ui.Caption(hint),
			ui.Text(token).Size(th.Typography.Caption.Size).Style(ui.Spec{}.TextColor(ui.TokenInfo)),
		).Gap(th.Spacing.XS).Align(ui.AlignCenter),
		ui.Spacer(),
		ui.Button(id, ui.Text("Add")).Ghost().HoverFill().IconStart(icons.Plus).OnClick(add),
	).Gap(th.Spacing.S).Align(ui.AlignCenter)
}

// emptyRows says what an empty list of assertions or captures means.
func emptyRows(th *theme.Theme, text string) ui.View {
	return ui.Row(ui.Caption(text)).
		PaddingXY(th.Spacing.M, th.Spacing.MNudge).
		Border(ui.TokenBorder, th.Stroke.Thin).
		BorderStyle(ui.BorderDashed).
		Radius(th.Radius.Large)
}

// stacked lays out one assertion or capture on two lines in a narrow
// card, framed so the lines read as one entry.
func stacked(th *theme.Theme, lines ...ui.View) ui.View {
	return ui.Column(lines...).Gap(th.Spacing.XS).
		Padding(th.Spacing.XS).
		Border(ui.TokenBorder, th.Stroke.Thin).
		Radius(th.Radius.Medium)
}

func (c *Container) assertions(th *theme.Theme, m *stepModel) ui.View {
	rows := []ui.View{section(th, "tc-assert-add-"+m.key, "Assertions", len(m.asserts), "Values may use", "{{variables}}", func() {
		m.asserts = append(m.asserts, &assertModel{key: newKey(), target: domain.TestTargetStatus, op: domain.TestOpEq})
		c.markDirty()
	})}
	if len(m.asserts) == 0 {
		rows = append(rows, emptyRows(th, "No assertions. The step passes if the call completes without error."))
	}
	for i, a := range m.asserts {
		rows = append(rows, c.assertRow(th, m, i, a))
	}
	return ui.Column(rows...).Gap(th.Spacing.S)
}

func (c *Container) assertRow(th *theme.Theme, m *stepModel, i int, a *assertModel) ui.View {
	k := a.key
	target := ui.Select("tc-a-target-"+k, targetOptions).Width(150).Selected(optionIndex(a.target, targetOptions)).
		OnChange(func(v string) { a.target = v; c.markDirty() })
	op := ui.Select("tc-a-op-"+k, opOptions).Width(130).Selected(optionIndex(a.op, opOptions)).
		OnChange(func(v string) { a.op = v; c.markDirty() })

	// Header and body sources say which header or JSON path they check.
	var sel ui.View
	switch {
	case usesPath(a.target):
		sel = ui.TextField("tc-a-sel-"+k, a.sel).Placeholder("$.data.id").Width(160).
			OnChange(func(s string) { a.sel = s; c.markDirty() })
	case usesKey(a.target):
		sel = ui.TextField("tc-a-sel-"+k, a.sel).Placeholder("Name").Width(160).
			OnChange(func(s string) { a.sel = s; c.markDirty() })
	}

	noValue := a.op == domain.TestOpExists || a.op == domain.TestOpNotExists
	value := ui.View(container.AssistURLField(ui.TextField("tc-a-val-"+k, a.value), c.varSource()).
		Placeholder(valueHint(a.op)).Disabled(noValue).
		OnChange(func(s string) { a.value = s; c.markDirty() }).
		Grow(1))
	if a.op == domain.TestOpType {
		if optionIndex(a.value, typeOptions) == 0 && a.value != typeOptions[0].Value {
			a.value = typeOptions[0].Value
		}
		value = ui.Select("tc-a-type-"+k, typeOptions).Selected(optionIndex(a.value, typeOptions)).
			OnChange(func(v string) { a.value = v; c.markDirty() }).
			Grow(1)
	}
	remove := ui.IconButton("tc-a-del-"+k, icons.Trash2).Tooltip("Remove").OnClick(func() {
		m.asserts = append(m.asserts[:i:i], m.asserts[i+1:]...)
		c.markDirty()
	})

	if c.compact() {
		first := []ui.View{target}
		if sel != nil {
			first = append(first, ui.ViewOf(sel).Grow(1))
		}
		return stacked(th,
			ui.Row(first...).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.Row(op, value, remove).Gap(th.Spacing.S).Align(ui.AlignCenter),
		)
	}
	cells := []ui.View{target}
	if sel != nil {
		cells = append(cells, sel)
	}
	cells = append(cells, op, value, remove)
	return ui.Row(cells...).Gap(th.Spacing.S).Align(ui.AlignCenter)
}

func valueHint(op string) string {
	switch op {
	case domain.TestOpIn:
		return "[200, 201]"
	case domain.TestOpMatches:
		return "Regular expression"
	case domain.TestOpGt, domain.TestOpGte, domain.TestOpLt, domain.TestOpLte, domain.TestOpLength:
		return "Number"
	case domain.TestOpExists, domain.TestOpNotExists:
		return ""
	}
	return "Expected value"
}

func (c *Container) captures(th *theme.Theme, m *stepModel) ui.View {
	rows := []ui.View{section(th, "tc-cap-add-"+m.key, "Captures", len(m.captures), "Stored for later steps as", "{{name}}", func() {
		m.captures = append(m.captures, &captureModel{key: newKey(), from: domain.TestTargetBody})
		c.markDirty()
	})}
	if len(m.captures) == 0 {
		rows = append(rows, emptyRows(th, "Nothing is captured from this step."))
	}
	// Variable names take the color {{names}} have in URL fields.
	varColor := th.Info
	for i, cp := range m.captures {
		ck := cp.key
		name := ui.TextField("tc-c-var-"+ck, cp.v).Placeholder("variable").Width(150).
			Highlight(func(v string) []ui.TextSpan { return []ui.TextSpan{{Start: 0, End: len(v), Color: varColor}} }).
			OnChange(func(s string) { cp.v = s; c.markDirty() })
		from := ui.Select("tc-c-from-"+ck, captureOptions).Width(130).Selected(optionIndex(cp.from, captureOptions)).
			OnChange(func(v string) { cp.from = v; c.markDirty() })
		var sel ui.View = ui.Spacer()
		switch {
		case usesPath(cp.from):
			sel = ui.TextField("tc-c-sel-"+ck, cp.sel).Placeholder("$.data.id").
				OnChange(func(s string) { cp.sel = s; c.markDirty() }).Grow(1)
		case usesKey(cp.from):
			sel = ui.TextField("tc-c-sel-"+ck, cp.sel).Placeholder("Name").
				OnChange(func(s string) { cp.sel = s; c.markDirty() }).Grow(1)
		}
		remove := ui.IconButton("tc-c-del-"+ck, icons.Trash2).Tooltip("Remove").OnClick(func() {
			m.captures = append(m.captures[:i:i], m.captures[i+1:]...)
			c.markDirty()
		})
		if c.compact() {
			rows = append(rows, stacked(th,
				ui.Row(ui.ViewOf(name).Grow(1), ui.Caption("from"), from).Gap(th.Spacing.S).Align(ui.AlignCenter),
				ui.Row(sel, remove).Gap(th.Spacing.S).Align(ui.AlignCenter),
			))
			continue
		}
		rows = append(rows, ui.Row(name, ui.Caption("from"), from, sel, remove).Gap(th.Spacing.S).Align(ui.AlignCenter))
	}
	return ui.Column(rows...).Gap(th.Spacing.S)
}

// varSource completes and explains {{names}}: the environment's, the
// case's variables and what the steps capture.
func (c *Container) varSource() vars.Source {
	return container.VarSource(c.deps, func() []vars.Entry {
		var out []vars.Entry
		for _, row := range c.vars.Rows {
			if k := strings.TrimSpace(row.Cells[varColKey]); k != "" {
				out = append(out, vars.Entry{Name: k, Value: row.Cells[varColValue], Kind: "test variable"})
			}
		}
		for _, steps := range c.sections {
			for _, m := range steps {
				for _, cp := range m.captures {
					if v := strings.TrimSpace(cp.v); v != "" {
						out = append(out, vars.Entry{Name: v, Kind: "captured by " + m.displayName()})
					}
				}
			}
		}
		return out
	})
}

// fold is a Disclosure header that also shows a summary badge.
func fold(th *theme.Theme, id, title, summary string, open bool, toggle func()) ui.View {
	chevron := icons.ChevronRight
	if open {
		chevron = icons.ChevronDown
	}
	return ui.Button(id, ui.Row(
		ui.Icon(chevron, th.Metrics.IconSizeSM, th.ForegroundMuted),
		ui.Strong(title),
		ui.Badge(summary),
	).Gap(th.Spacing.S).Align(ui.AlignCenter)).Ghost().HoverFill().OnClick(toggle)
}

// execution is the step's timeout, retries and failure handling, folded
// behind a summary of them.
func (c *Container) execution(th *theme.Theme, m *stepModel) ui.View {
	k := m.key
	head := fold(th, "tc-exec-"+k, "Execution", c.execSummary(m), m.execOpen, func() { m.execOpen = !m.execOpen })
	if !m.execOpen {
		return head
	}
	items := []ui.FormItem{
		ui.FormText("tc-id-"+k, "Step ID", "Reference this step from others, and pick it with Run.", m.id,
			func(s string) { m.id = s; c.markDirty() }),
		ui.FormText("tc-timeout-"+k, "Timeout", "Fail the call if no response arrives in time, such as 30s. Empty uses the case's.", m.timeout,
			func(s string) { m.timeout = s; c.markDirty() }),
		ui.FormStepper("tc-retries-"+k, "Retries", "Extra attempts after a failure.", float64(m.retries), 0, 10, 1,
			func(v float64) { m.retries = int(v); c.markDirty() }),
	}
	if m.retries > 0 {
		items = append(items, ui.FormText("tc-interval-"+k, "Retry interval", "Wait between attempts, such as 1s.", m.retryDelay,
			func(s string) { m.retryDelay = s; c.markDirty() }))
	}
	items = append(items, ui.FormSwitch("tc-cont-"+k, "Continue on failure", "Run the next steps even if this one fails.",
		m.continueOnFailure, func(v bool) { m.continueOnFailure = v; c.markDirty() }))
	return ui.Column(head, ui.Form("tc-exec-form-"+k, items...).PaddingLeft(th.Spacing.L)).Gap(th.Spacing.S)
}

// execSummary is the Execution badge: timeout, retries, failure handling.
func (c *Container) execSummary(m *stepModel) string {
	timeout := strings.TrimSpace(m.timeout)
	switch {
	case timeout != "":
		timeout += " timeout"
	case c.tc.Spec.Options.Timeout > 0:
		timeout = durationText(c.tc.Spec.Options.Timeout) + " timeout"
	default:
		timeout = "No timeout"
	}
	parts := []string{timeout}
	switch {
	case m.retries == 0:
		parts = append(parts, "No retries")
	case strings.TrimSpace(m.retryDelay) != "":
		parts = append(parts, fmt.Sprintf("%s every %s", plural(m.retries, "retry", "retries"), strings.TrimSpace(m.retryDelay)))
	default:
		parts = append(parts, plural(m.retries, "retry", "retries"))
	}
	if m.continueOnFailure {
		parts = append(parts, "Continues on failure")
	}
	return strings.Join(parts, " · ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// overrides edits what a step changes in its request, folded behind a
// count of its overrides.
func (c *Container) overrides(ctx *ui.Ctx, th *theme.Theme, m *stepModel) ui.View {
	k := m.key
	n := len(m.vars.Rows) + len(m.headers.Rows) + len(m.query.Rows)
	if len(m.body.Bytes()) > 0 {
		n++
	}
	summary := "None"
	if n > 0 {
		summary = plural(n, "override", "overrides")
	}
	head := fold(th, "tc-ovr-"+k, "Request overrides", summary, m.showOverrides, func() { m.showOverrides = !m.showOverrides })
	if !m.showOverrides {
		return head
	}
	req := m.request.Ref
	if req == "" {
		req = "the chosen request"
	}
	table := func(title, hint string, t *ui.Table) ui.View {
		return ui.Column(
			ui.Row(ui.Strong(title), ui.Caption(hint), ui.Spacer(),
				ui.Button("tc-ovr-add-"+title+"-"+k, ui.Text("Add")).Ghost().HoverFill().IconStart(icons.Plus).OnClick(func() {
					container.AddKVRow(t, c.markDirty)
				}),
			).Gap(th.Spacing.S).Align(ui.AlignCenter),
			ui.ViewOf(t).Height(tableHeight(t)),
		).Gap(th.Spacing.XS)
	}
	return ui.Column(
		head,
		ui.Column(
			container.MutedParagraph("Uses the request as saved in "+req+". Override its variables, headers, query or body here.").Size(th.Typography.Caption.Size),
			table("Variables", "for this step only", m.vars),
			table("Headers", "set or replace", m.headers),
			table("Query", "set or replace", m.query),
			ui.Column(
				ui.Row(ui.Strong("Body"), ui.Caption("replaces the request body when not empty")).Gap(th.Spacing.S).Align(ui.AlignCenter),
				ui.ViewOf(m.body).Height(120).Border(ui.TokenBorder, th.Stroke.Thin),
			).Gap(th.Spacing.XS),
		).Gap(th.Spacing.M).PaddingLeft(th.Spacing.L),
	).Gap(th.Spacing.S)
}

// tableHeight fits a key/value table to its rows, as the step list scrolls
// rather than the table.
func tableHeight(t *ui.Table) float32 {
	return float32(36 + 34*max(1, len(t.Rows)))
}
