package testcase

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/testrun"
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
			ui.Caption("Run cases by tag with chapar-cli test --tag."),
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
