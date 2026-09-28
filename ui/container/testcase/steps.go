package testcase

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/testrun"
	"github.com/chapar-rest/chapar/ui/container"
)

// autoStepID matches the IDs Add step gives.
var autoStepID = regexp.MustCompile(`^step(-\d+)?$`)

var targetOptions = []ui.SelectOption{
	{Label: "Status", Value: domain.TestTargetStatus},
	{Label: "Header", Value: domain.TestTargetHeader},
	{Label: "Body (JSON)", Value: domain.TestTargetBody},
	{Label: "Body text", Value: domain.TestTargetText},
	{Label: "Cookie", Value: domain.TestTargetCookie},
	{Label: "Time (ms)", Value: domain.TestTargetTime},
	{Label: "Size (bytes)", Value: domain.TestTargetSize},
	{Label: "gRPC metadata", Value: domain.TestTargetMetadata},
	{Label: "gRPC trailer", Value: domain.TestTargetTrailer},
}

// captureOptions are the targets a value can be captured from.
var captureOptions = []ui.SelectOption{
	{Label: "Body (JSON)", Value: domain.TestTargetBody},
	{Label: "Header", Value: domain.TestTargetHeader},
	{Label: "Cookie", Value: domain.TestTargetCookie},
	{Label: "Status", Value: domain.TestTargetStatus},
	{Label: "Body text", Value: domain.TestTargetText},
	{Label: "gRPC metadata", Value: domain.TestTargetMetadata},
	{Label: "gRPC trailer", Value: domain.TestTargetTrailer},
}

// typeOptions are the JSON types the "type is" operator checks.
var typeOptions = []ui.SelectOption{
	{Label: "string", Value: "string"},
	{Label: "number", Value: "number"},
	{Label: "boolean", Value: "boolean"},
	{Label: "null", Value: "null"},
	{Label: "array", Value: "array"},
	{Label: "object", Value: "object"},
}

var opOptions = []ui.SelectOption{
	{Label: "equals", Value: domain.TestOpEq},
	{Label: "not equals", Value: domain.TestOpNe},
	{Label: "exists", Value: domain.TestOpExists},
	{Label: "not exists", Value: domain.TestOpNotExists},
	{Label: "contains", Value: domain.TestOpContains},
	{Label: "not contains", Value: domain.TestOpNotContains},
	{Label: "one of", Value: domain.TestOpIn},
	{Label: ">", Value: domain.TestOpGt},
	{Label: "≥", Value: domain.TestOpGte},
	{Label: "<", Value: domain.TestOpLt},
	{Label: "≤", Value: domain.TestOpLte},
	{Label: "matches", Value: domain.TestOpMatches},
	{Label: "type is", Value: domain.TestOpType},
	{Label: "length", Value: domain.TestOpLength},
}

func optionIndex(value string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == value {
			return i
		}
	}
	return 0
}

func (c *Container) editorPane(ctx *ui.Ctx, th *theme.Theme) ui.View {
	id := c.ID()
	var body ui.View
	switch c.tab {
	case tabVariables:
		body = c.variablesTab(th)
	case tabSettings:
		body = c.settingsTab(th)
	case tabYAML:
		body = c.yamlTab(th)
	default:
		body = c.stepsTab(ctx, th)
	}
	return ui.Column(
		ui.Tabs("tc-edit-tabs-"+id, c.editTabs).
			Selected(c.tab).
			Closable(false).
			OnSelectItem(func(i int, _ string) { c.setTab(i) }).
			TabBackground(th.Background),
		c.problemsView(th),
		ui.ViewOf(body).Grow(1),
	).Grow(1)
}

// problemsView lists what is wrong with the case, so Run can be trusted.
func (c *Container) problemsView(th *theme.Theme) ui.View {
	if c.tab == tabYAML || len(c.problems) == 0 {
		return ui.Column()
	}
	lines := make([]ui.View, 0, len(c.problems))
	for i, p := range c.problems {
		if i == 4 && len(c.problems) > 5 {
			lines = append(lines, ui.Caption(fmt.Sprintf("… and %d more", len(c.problems)-4)))
			break
		}
		lines = append(lines, ui.Row(
			ui.Icon(icons.TriangleAlert, 14, th.Warning),
			ui.Caption(c.describeProblem(p)),
		).Gap(th.Spacing.XS).Align(ui.AlignCenter))
	}
	return ui.Column(lines...).Gap(th.Spacing.XXS).
		PaddingXY(th.Spacing.M, th.Spacing.S).
		Background(ui.TokenWarningSurface)
}

// describeProblem names a step by its name rather than its index.
func (c *Container) describeProblem(p testrun.Problem) string {
	for section, steps := range c.sections {
		for i, m := range steps {
			prefix := fmt.Sprintf("%s[%d]", section, i)
			if p.Where == prefix || strings.HasPrefix(p.Where, prefix+".") {
				where := m.displayName() + strings.TrimPrefix(p.Where, prefix)
				return where + ": " + p.Message
			}
		}
	}
	return p.Error()
}

func (c *Container) stepsTab(ctx *ui.Ctx, th *theme.Theme) ui.View {
	id := c.ID()
	count := func(s string) string {
		if n := len(c.sections[s]); n > 0 {
			return fmt.Sprintf(" (%d)", n)
		}
		return ""
	}
	sections := []string{testrun.SectionSetup, testrun.SectionSteps, testrun.SectionTeardown}
	sectionIdx := 1
	for i, s := range sections {
		if s == c.section {
			sectionIdx = i
		}
	}
	header := ui.Row(
		ui.Segmented("tc-section-"+id,
			ui.SegmentItem{Label: "Setup" + count(testrun.SectionSetup), Value: testrun.SectionSetup},
			ui.SegmentItem{Label: "Steps" + count(testrun.SectionSteps), Value: testrun.SectionSteps},
			ui.SegmentItem{Label: "Teardown" + count(testrun.SectionTeardown), Value: testrun.SectionTeardown},
		).Selected(sectionIdx).OnSelectItem(func(_ int, v string) { c.section = v }),
		ui.Spacer(),
		ui.Button("tc-add-step-"+id, ui.Text("Add step")).IconStart(icons.Plus).OnClick(c.addStep),
	).Gap(th.Spacing.S).Align(ui.AlignCenter).PaddingXY(th.Spacing.M, th.Spacing.S)

	steps := c.sections[c.section]
	if len(steps) == 0 {
		return ui.Column(header, emptyView(th, emptyTitle(c.section), emptyDetail(c.section))).Grow(1)
	}
	cards := make([]ui.View, 0, len(steps))
	for i, m := range steps {
		cards = append(cards, c.stepCard(ctx, th, i, m))
	}
	return ui.Column(
		header,
		ui.Scroll("tc-steps-"+id+"-"+c.section,
			ui.Column(cards...).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.S),
		).Grow(1),
	).Grow(1)
}

// emptyView is a centered title and a description that wraps, so it fits
// a narrow pane; ui.EmptyState keeps its text on one line.
func emptyView(th *theme.Theme, title, detail string) ui.View {
	rows := []ui.View{}
	if title != "" {
		rows = append(rows, ui.Paragraph(title).TextAlign(ui.AlignCenter).Weight(600))
	}
	rows = append(rows, ui.Paragraph(detail).TextAlign(ui.AlignCenter).
		Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
	// The column stretches, so the paragraphs get a width to wrap at; they
	// center their lines themselves.
	return ui.Column(
		ui.Spacer(),
		ui.Column(rows...).Gap(th.Spacing.S),
		ui.Spacer(),
	).Padding(th.Spacing.XL).Grow(1)
}

func emptyTitle(section string) string {
	switch section {
	case testrun.SectionSetup:
		return "No setup steps"
	case testrun.SectionTeardown:
		return "No teardown steps"
	}
	return "No steps yet"
}

func emptyDetail(section string) string {
	switch section {
	case testrun.SectionSetup:
		return "Setup runs first, for logging in or creating data. If it fails, the steps are skipped."
	case testrun.SectionTeardown:
		return "Teardown always runs last, even after a failure, to clean up."
	}
	return "Add a step to send one of your requests and check its response."
}

func (c *Container) addStep() {
	taken := c.stepIDs()
	m := loadStep(domain.TestStep{ID: uniqueStepID("step", taken)}, c.markDirty)
	m.open = true
	c.sections[c.section] = append(c.sections[c.section], m)
	c.markDirty()
}

func (c *Container) stepIDs() map[string]bool {
	taken := map[string]bool{}
	for _, steps := range c.sections {
		for _, m := range steps {
			taken[m.id] = true
		}
	}
	return taken
}

func (c *Container) moveStep(i, delta int) {
	steps := c.sections[c.section]
	j := i + delta
	if j < 0 || j >= len(steps) {
		return
	}
	steps[i], steps[j] = steps[j], steps[i]
	c.markDirty()
}

func (c *Container) removeStep(i int) {
	steps := c.sections[c.section]
	steps[i].close()
	c.sections[c.section] = append(steps[:i:i], steps[i+1:]...)
	c.markDirty()
}

func (c *Container) duplicateStep(i int) {
	steps := c.sections[c.section]
	var problems []testrun.Problem
	s := steps[i].dump("", &problems)
	s.ID = uniqueStepID(s.ID, c.stepIDs())
	m := loadStep(s, c.markDirty)
	m.open = true
	c.sections[c.section] = append(steps[:i+1:i+1], append([]*stepModel{m}, steps[i+1:]...)...)
	c.markDirty()
}

// requestOptions lists every request, labeled as the sidebar shows it.
func (c *Container) requestOptions() ([]ui.SelectOption, map[string]*domain.Request) {
	opts := []ui.SelectOption{{Label: "Choose a request…", Value: ""}}
	byID := map[string]*domain.Request{}
	if c.deps.Catalog == nil {
		return opts, byID
	}
	for _, col := range c.deps.Catalog.AllCollections() {
		for _, r := range col.Spec.Requests {
			r.CollectionName = col.MetaData.Name
			opts = append(opts, ui.SelectOption{Label: col.MetaData.Name + " / " + domain.RequestDisplayName(r), Value: r.MetaData.ID})
			byID[r.MetaData.ID] = r
		}
	}
	for _, r := range c.deps.Catalog.StandaloneRequests() {
		opts = append(opts, ui.SelectOption{Label: domain.RequestDisplayName(r), Value: r.MetaData.ID})
		byID[r.MetaData.ID] = r
	}
	return opts, byID
}

// selectedRequest is the option to show for a step's request. A ref that
// matches nothing gets its own option, so the step still says what it
// wanted.
func (c *Container) selectedRequest(m *stepModel, opts []ui.SelectOption) ([]ui.SelectOption, int) {
	if m.request.ID == "" && m.request.Ref == "" {
		return opts, 0
	}
	if c.deps.Catalog != nil {
		if req, err := testrun.Resolve(c.deps.Catalog, m.request); err == nil {
			return opts, optionIndex(req.MetaData.ID, opts)
		}
	}
	label := m.request.Ref
	if label == "" {
		label = m.request.ID
	}
	opts = append(opts, ui.SelectOption{Label: "Missing: " + label, Value: "missing"})
	return opts, len(opts) - 1
}

func (c *Container) stepCard(ctx *ui.Ctx, th *theme.Theme, i int, m *stepModel) ui.View {
	k := m.key
	n := len(c.sections[c.section])
	chevron := icons.ChevronRight
	if m.open {
		chevron = icons.ChevronDown
	}
	opts, byID := c.requestOptions()
	opts, sel := c.selectedRequest(m, opts)

	compact := c.compact()
	reqWidth := float32(260)
	if compact {
		reqWidth = 200
	}
	actions := []ui.View{
		ui.IconButton("tc-run-step-"+k, icons.Play).Tooltip("Run this step (with setup and teardown)").
			Disabled(c.running || c.section != testrun.SectionSteps || c.deps.Tests == nil).
			OnClick(func() { c.start([]string{m.id}) }),
	}
	if compact {
		actions = append(actions, c.stepMenu(i, n, m))
	} else {
		actions = append(actions,
			ui.IconButton("tc-up-"+k, icons.ArrowUp).Tooltip("Move up").Disabled(i == 0).
				OnClick(func() { c.moveStep(i, -1) }),
			ui.IconButton("tc-down-"+k, icons.ArrowDown).Tooltip("Move down").Disabled(i == n-1).
				OnClick(func() { c.moveStep(i, 1) }),
			ui.IconButton("tc-dup-"+k, icons.Copy).Tooltip("Duplicate").OnClick(func() { c.duplicateStep(i) }),
			ui.IconButton("tc-del-"+k, icons.Trash2).Tooltip("Delete").OnClick(func() { c.removeStep(i) }),
		)
	}

	header := ui.Row(append([]ui.View{
		ui.IconButton("tc-open-"+k, chevron).OnClick(func() { m.open = !m.open }),
		c.statusIcon(th, m),
		ui.TextField("tc-name-"+k, m.name).
			Placeholder("Step name").
			OnChange(func(s string) { m.name = s; c.markDirty() }).
			Grow(1),
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
	}, actions...)...).Gap(th.Spacing.XS).Align(ui.AlignCenter)

	rows := []ui.View{header}
	if m.open {
		rows = append(rows, c.stepBody(ctx, th, m))
	}
	card := ui.Column(rows...).Gap(th.Spacing.S).
		Padding(th.Spacing.S).
		Border(ui.TokenBorder, th.Stroke.Thin).
		Radius(th.Radius.Medium).
		Layout(ctx)
	if i == 0 {
		c.measure(card)
	}
	return ui.Raw(card)
}

// compactCardWidth is the card width below which the header folds its
// move, duplicate and delete buttons into a menu.
const compactCardWidth = 700

func (c *Container) compact() bool { return c.cardW > 0 && c.cardW < compactCardWidth }

// stacked lays out one assertion or capture on two lines, framed so the
// lines read as one entry.
func stacked(th *theme.Theme, lines ...ui.View) ui.View {
	return ui.Column(lines...).Gap(th.Spacing.XS).
		Padding(th.Spacing.XS).
		Border(ui.TokenBorder, th.Stroke.Thin).
		Radius(th.Radius.Small)
}

// measure keeps the width the step cards were laid out at. Layout only
// knows it after the views are built, so a change shows on the next frame.
func (c *Container) measure(el *layout.Element) {
	prev := el.AfterLayout
	el.AfterLayout = func(e *layout.Element) bool {
		// Frame is filled only after this hook; the solved size is known.
		w, _ := e.LayoutSize()
		if (w < compactCardWidth) != (c.cardW < compactCardWidth) || c.cardW == 0 {
			c.deps.WakeNow()
		}
		c.cardW = w
		if prev != nil {
			return prev(e)
		}
		return false
	}
}

// stepMenu holds the header buttons that do not fit a narrow card.
func (c *Container) stepMenu(i, n int, m *stepModel) ui.View {
	return ui.IconButton("tc-more-"+m.key, icons.Ellipsis).Menu([]ui.MenuItem{
		{Label: "Move up", Disabled: i == 0, OnSelect: func() { c.moveStep(i, -1) }},
		{Label: "Move down", Disabled: i == n-1, OnSelect: func() { c.moveStep(i, 1) }},
		{Label: "Duplicate", OnSelect: func() { c.duplicateStep(i) }},
		ui.MenuSeparator,
		{Label: "Delete", OnSelect: func() { c.removeStep(i) }},
	})
}

// statusIcon shows how the step did in the last run.
func (c *Container) statusIcon(th *theme.Theme, m *stepModel) ui.View {
	r := c.results[resultKey(c.section, m.id)]
	icon, color := statusLook(th, r)
	return ui.Icon(icon, 16, color)
}

func statusLook(th *theme.Theme, r *testrun.StepResult) (icons.Icon, render.Color) {
	if r == nil {
		return icons.Circle, th.ForegroundMuted
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
	return icons.CircleDashed, th.ForegroundMuted
}

func (c *Container) stepBody(ctx *ui.Ctx, th *theme.Theme, m *stepModel) ui.View {
	k := m.key
	return ui.Column(
		c.assertRows(th, m),
		c.captureRows(th, m),
		ui.Row(
			ui.Caption("ID"),
			ui.TextField("tc-id-"+k, m.id).Width(140).OnChange(func(s string) { m.id = s; c.markDirty() }),
			ui.Caption("Timeout"),
			ui.TextField("tc-timeout-"+k, m.timeout).Placeholder("30s").Width(70).
				OnChange(func(s string) { m.timeout = s; c.markDirty() }),
			ui.Caption("Retries"),
			ui.TextField("tc-retry-"+k, m.retryCount).Placeholder("0").Width(50).
				OnChange(func(s string) { m.retryCount = s; c.markDirty() }),
			ui.Caption("every"),
			ui.TextField("tc-delay-"+k, m.retryDelay).Placeholder("1s").Width(60).
				OnChange(func(s string) { m.retryDelay = s; c.markDirty() }),
			ui.Checkbox("tc-cont-"+k, "Continue on failure").Check(m.continueOnFailure).
				OnToggle(func(v bool) { m.continueOnFailure = v; c.markDirty() }),
		).Gap(th.Spacing.S).Align(ui.AlignCenter).Wrap(),
		c.overrides(ctx, th, m),
	).Gap(th.Spacing.M).PaddingLeft(th.Spacing.L)
}

func sectionHeader(th *theme.Theme, title, hint string, add ui.View) ui.View {
	return ui.Row(
		ui.Strong(title),
		ui.Caption(hint),
		ui.Spacer(),
		add,
	).Gap(th.Spacing.S).Align(ui.AlignCenter)
}

func (c *Container) assertRows(th *theme.Theme, m *stepModel) ui.View {
	k := m.key
	rows := []ui.View{sectionHeader(th, "Assertions", "values may use {{variables}}",
		ui.IconButton("tc-assert-add-"+k, icons.Plus).Tooltip("Add assertion").OnClick(func() {
			m.asserts = append(m.asserts, &assertModel{key: newKey(), target: domain.TestTargetStatus, op: domain.TestOpEq, value: "200"})
			c.markDirty()
		}))}
	for i, a := range m.asserts {
		rows = append(rows, c.assertRow(th, m, i, a))
	}
	return ui.Column(rows...).Gap(th.Spacing.XS)
}

func (c *Container) assertRow(th *theme.Theme, m *stepModel, i int, a *assertModel) ui.View {
	k := a.key
	sel := ui.View(ui.Row().Width(180))
	switch {
	case usesPath(a.target):
		sel = ui.TextField("tc-a-sel-"+k, a.sel).Placeholder("$.data.id").Width(180).
			OnChange(func(s string) { a.sel = s; c.markDirty() })
	case usesKey(a.target):
		sel = ui.TextField("tc-a-sel-"+k, a.sel).Placeholder("Name").Width(180).
			OnChange(func(s string) { a.sel = s; c.markDirty() })
	}
	noValue := a.op == domain.TestOpExists || a.op == domain.TestOpNotExists
	value := ui.View(ui.TextField("tc-a-val-"+k, a.value).Placeholder(valueHint(a.op)).Disabled(noValue).
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
	target := ui.Select("tc-a-target-"+k, targetOptions).Width(160).Selected(optionIndex(a.target, targetOptions)).
		OnChange(func(v string) { a.target = v; c.markDirty() })
	op := ui.Select("tc-a-op-"+k, opOptions).Width(120).Selected(optionIndex(a.op, opOptions)).
		OnChange(func(v string) { a.op = v; c.markDirty() })
	remove := ui.IconButton("tc-a-del-"+k, icons.X).Tooltip("Remove").OnClick(func() {
		m.asserts = append(m.asserts[:i:i], m.asserts[i+1:]...)
		c.markDirty()
	})
	if c.compact() {
		return stacked(th,
			ui.Row(target, ui.ViewOf(sel).Grow(1)).Gap(th.Spacing.XS).Align(ui.AlignCenter),
			ui.Row(op, value, remove).Gap(th.Spacing.XS).Align(ui.AlignCenter),
		)
	}
	return ui.Row(target, sel, op, value, remove).Gap(th.Spacing.XS).Align(ui.AlignCenter)
}

func valueHint(op string) string {
	switch op {
	case domain.TestOpIn:
		return "[200, 201]"
	case domain.TestOpType:
		return "string, number, boolean, null, array, object"
	case domain.TestOpMatches:
		return "regular expression"
	case domain.TestOpGt, domain.TestOpGte, domain.TestOpLt, domain.TestOpLte, domain.TestOpLength:
		return "number"
	case domain.TestOpExists, domain.TestOpNotExists:
		return ""
	}
	return `value, "quoted text" or {{variable}}`
}

func (c *Container) captureRows(th *theme.Theme, m *stepModel) ui.View {
	k := m.key
	rows := []ui.View{sectionHeader(th, "Captures", "store a value for later steps as {{name}}",
		ui.IconButton("tc-cap-add-"+k, icons.Plus).Tooltip("Add capture").OnClick(func() {
			m.captures = append(m.captures, &captureModel{key: newKey(), from: domain.TestTargetBody})
			c.markDirty()
		}))}
	for i, cp := range m.captures {
		ck := cp.key
		sel := ui.View(ui.Row().Width(180))
		switch {
		case usesPath(cp.from):
			sel = ui.TextField("tc-c-sel-"+ck, cp.sel).Placeholder("$.data.id").Width(180).
				OnChange(func(s string) { cp.sel = s; c.markDirty() })
		case usesKey(cp.from):
			sel = ui.TextField("tc-c-sel-"+ck, cp.sel).Placeholder("Name").Width(180).
				OnChange(func(s string) { cp.sel = s; c.markDirty() })
		}
		name := ui.TextField("tc-c-var-"+ck, cp.v).Placeholder("variable").Width(150).
			OnChange(func(s string) { cp.v = s; c.markDirty() })
		from := ui.Select("tc-c-from-"+ck, captureOptions).Width(150).Selected(optionIndex(cp.from, captureOptions)).
			OnChange(func(v string) { cp.from = v; c.markDirty() })
		remove := ui.IconButton("tc-c-del-"+ck, icons.X).Tooltip("Remove").OnClick(func() {
			m.captures = append(m.captures[:i:i], m.captures[i+1:]...)
			c.markDirty()
		})
		if c.compact() {
			rows = append(rows, stacked(th,
				ui.Row(ui.ViewOf(name).Grow(1), ui.Caption("from"), from).Gap(th.Spacing.XS).Align(ui.AlignCenter),
				ui.Row(ui.ViewOf(sel).Grow(1), remove).Gap(th.Spacing.XS).Align(ui.AlignCenter),
			))
			continue
		}
		rows = append(rows, ui.Row(name, ui.Caption("from"), from, sel, ui.Spacer(), remove).
			Gap(th.Spacing.XS).Align(ui.AlignCenter))
	}
	return ui.Column(rows...).Gap(th.Spacing.XS)
}

// overrides edits what a step changes in its request, folded by default.
func (c *Container) overrides(ctx *ui.Ctx, th *theme.Theme, m *stepModel) ui.View {
	k := m.key
	chevron := icons.ChevronRight
	if m.showOverrides {
		chevron = icons.ChevronDown
	}
	toggle := ui.Button("tc-ovr-"+k, ui.Text("Request overrides")).Ghost().HoverFill().IconStart(chevron).
		OnClick(func() { m.showOverrides = !m.showOverrides })
	if !m.showOverrides {
		return toggle
	}
	table := func(title, hint string, t *ui.Table) ui.View {
		return ui.Column(
			sectionHeader(th, title, hint, ui.IconButton("tc-ovr-add-"+title+"-"+k, icons.Plus).OnClick(func() {
				container.AddKVRow(t, c.markDirty)
			})),
			ui.ViewOf(t).Height(tableHeight(t)),
		).Gap(th.Spacing.XS)
	}
	return ui.Column(
		toggle,
		ui.Column(
			table("Variables", "for this step only", m.vars),
			table("Headers", "set or replace", m.headers),
			table("Query", "set or replace", m.query),
			ui.Column(
				sectionHeader(th, "Body", "replaces the request body when not empty", ui.Row()),
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

func (c *Container) variablesTab(th *theme.Theme) ui.View {
	id := c.ID()
	return ui.Column(
		ui.Row(
			ui.Caption("Variables of the case. They override the environment and can be used as {{name}}."),
			ui.Spacer(),
			ui.Button("tc-var-add-"+id, ui.Text("Add")).IconStart(icons.Plus).OnClick(func() {
				c.vars.AddRow(ui.TableRow{ID: newKey(), Cells: map[string]string{}})
				c.markDirty()
			}),
		).Gap(th.Spacing.S).Align(ui.AlignCenter),
		ui.ViewOf(c.vars).Grow(1),
	).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1)
}

func (c *Container) settingsTab(th *theme.Theme) ui.View {
	id := c.ID()
	o := &c.tc.Spec.Options
	return ui.Scroll("tc-settings-"+id, ui.Column(
		ui.Form("tc-settings-form-"+id,
			ui.FormText("tc-desc-"+id, "Description", "", c.tc.Spec.Description, func(s string) {
				c.tc.Spec.Description = s
				c.markDirty()
			}),
			ui.FormText("tc-timeout-"+id, "Step timeout", "Default for steps without their own, such as 10s. Empty waits as long as the request's own timeout.",
				durationText(o.Timeout), func(s string) {
					var problems []testrun.Problem
					if d := parseDuration(s, "options.timeout", &problems); len(problems) == 0 {
						o.Timeout = d
						c.markDirty()
					}
				}),
			ui.FormSwitch("tc-continue-"+id, "Continue after a failed step", "Run the remaining steps instead of skipping them.",
				o.ContinueOnFailure, func(v bool) { o.ContinueOnFailure = v; c.markDirty() }),
			ui.FormSwitch("tc-persist-"+id, "Save environment changes", "Keep what the requests' scripts and extract rules set in the environment. Off: the environment is left as it was.",
				o.PersistEnv, func(v bool) { o.PersistEnv = v; c.markDirty() }),
		),
		ui.Column(
			ui.Strong("Tags"),
			ui.Caption("Run cases by tag with chapar test --tag."),
			ui.TagEdit("tc-tags-"+id, c.tc.Spec.Tags).OnTags(func(tags []string) {
				c.tc.Spec.Tags = tags
				c.markDirty()
			}),
		).Gap(th.Spacing.XS),
	).Gap(th.Spacing.L).Padding(th.Spacing.M))
}

func (c *Container) yamlTab(th *theme.Theme) ui.View {
	rows := []ui.View{}
	if c.yamlErr != "" {
		rows = append(rows, ui.Alert(c.yamlErr, ui.AlertError))
	}
	rows = append(rows, ui.ViewOf(c.yamlEd).Grow(1))
	return ui.Column(rows...).Gap(th.Spacing.S).Padding(th.Spacing.S).Grow(1)
}
