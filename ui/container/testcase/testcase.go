// Package testcase is the editor of a test case: its steps, variables and
// settings, as a form or as YAML, with the results of its last run.
package testcase

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/testrun"
	"github.com/chapar-rest/chapar/ui/container"
)

// Editor tabs.
const (
	tabSteps = iota
	tabVariables
	tabSettings
	tabYAML
)

// Variables table columns.
const (
	varColKey   = "key"
	varColValue = "value"
	varColOsEnv = "osenv"
	varColAct   = "act"
)

type Container struct {
	tc   *domain.TestCase // metadata and settings; steps live in sections
	deps container.Deps

	sections map[string][]*stepModel
	section  string // the section the steps tab shows
	vars     *ui.Table

	tab      int
	editTabs []ui.TabModel
	cardW    float32 // width of the step cards in the last layout
	yamlEd   *ui.Editor
	yamlBase string
	yamlErr  string

	dirty    bool
	problems []testrun.Problem
	checked  bool // problems are up to date

	// The current or last run. events is filled by the run's goroutine and
	// drained by Layout.
	mu       sync.Mutex
	events   []testrun.Event
	running  bool
	cancel   context.CancelFunc
	run      *testrun.Run
	results  map[string]*testrun.StepResult // by section/step id
	selected int                            // index into run.Steps
	respEd   *ui.Editor
	respFor  string // the step result respEd shows
	runSeq   int    // counts runs, so a new run's bodies replace the old
	// detailTab is the Checks, Body or Headers tab of the selected step.
	detailTab  int
	detailTabs []ui.TabModel
}

func Open(tc *domain.TestCase, deps container.Deps) *Container {
	c := &Container{
		deps:    deps,
		section: testrun.SectionSteps,
		editTabs: []ui.TabModel{
			{Title: "Steps"}, {Title: "Variables"}, {Title: "Settings"}, {Title: "YAML"},
		},
		results:    map[string]*testrun.StepResult{},
		selected:   -1,
		detailTabs: []ui.TabModel{{Title: "Checks"}, {Title: "Body"}, {Title: "Headers"}},
	}
	c.vars = c.newVarsTable()
	c.load(copyCase(tc))
	return c
}

// copyCase returns a deep copy that keeps the ID.
func copyCase(tc *domain.TestCase) *domain.TestCase {
	cp := tc.Clone()
	cp.MetaData.ID = tc.MetaData.ID
	return cp
}

// load replaces the form with tc.
func (c *Container) load(tc *domain.TestCase) {
	for _, steps := range c.sections {
		for _, m := range steps {
			m.close()
		}
	}
	c.tc = tc
	c.sections = map[string][]*stepModel{}
	for section, steps := range map[string][]domain.TestStep{
		testrun.SectionSetup:    tc.Spec.Setup,
		testrun.SectionSteps:    tc.Spec.Steps,
		testrun.SectionTeardown: tc.Spec.Teardown,
	} {
		for _, s := range steps {
			c.sections[section] = append(c.sections[section], loadStep(s, c.markDirty))
		}
	}
	rows := make([]ui.TableRow, 0, len(tc.Spec.Variables))
	for _, v := range tc.Spec.Variables {
		osEnv := ""
		if v.From != nil {
			osEnv = v.From.OsEnv
		}
		rows = append(rows, ui.TableRow{ID: uuid.NewString(), Cells: map[string]string{
			varColKey: v.Key, varColValue: v.Value, varColOsEnv: osEnv,
		}})
	}
	c.vars.SetRows(rows)
	c.checked = false
}

func (c *Container) newVarsTable() *ui.Table {
	t := ui.NewTable([]ui.TableColumn{
		{ID: varColKey, Label: "Name", Kind: ui.TableColEditable},
		{ID: varColValue, Label: "Value", Kind: ui.TableColEditable},
		{ID: varColOsEnv, Label: "Or from OS variable", Kind: ui.TableColEditable, Width: 200},
		{ID: varColAct, Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
	}, []ui.TableAction{{Icon: icons.Trash2, Tooltip: "Delete"}})
	t.HighlightSelected = false
	t.CollapseEmpty = true
	t.Actions[0].OnClick = func(rowID string) {
		t.RemoveRow(rowID)
		c.markDirty()
	}
	t.OnCellChange = func(_, _, _ string) { c.markDirty() }
	return t
}

func (c *Container) ID() string           { return c.tc.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindTestCase }
func (c *Container) Title() string        { return c.tc.MetaData.Name }

func (c *Container) Dirty() bool {
	if c.dirty || c.yamlModified() {
		return true
	}
	for _, steps := range c.sections {
		for _, m := range steps {
			if m.modified() {
				return true
			}
		}
	}
	return false
}

func (c *Container) yamlModified() bool {
	return c.tab == tabYAML && c.yamlEd != nil && string(c.yamlEd.Bytes()) != c.yamlBase
}

func (c *Container) markDirty() {
	c.dirty = true
	c.checked = false
	c.deps.ReportDirty(true)
}

func (c *Container) Close() {
	c.stop()
	for _, steps := range c.sections {
		for _, m := range steps {
			m.close()
		}
	}
	if c.yamlEd != nil {
		c.yamlEd.Close()
	}
	if c.respEd != nil {
		c.respEd.Close()
	}
}

// dump builds the test case the form describes, with the problems of text
// that did not parse.
func (c *Container) dump() (*domain.TestCase, []testrun.Problem) {
	tc := &domain.TestCase{
		ApiVersion: domain.ApiVersion,
		Kind:       domain.KindTestCase,
		MetaData:   c.tc.MetaData,
		Spec: domain.TestCaseSpec{
			Description: c.tc.Spec.Description,
			Tags:        append([]string(nil), c.tc.Spec.Tags...),
			Options:     c.tc.Spec.Options,
			Steps:       []domain.TestStep{},
		},
	}
	var problems []testrun.Problem
	for _, row := range c.vars.Rows {
		key := strings.TrimSpace(row.Cells[varColKey])
		value, osEnv := row.Cells[varColValue], strings.TrimSpace(row.Cells[varColOsEnv])
		if key == "" && value == "" && osEnv == "" {
			continue
		}
		v := domain.TestVariable{Key: key, Value: value}
		if osEnv != "" {
			v.From = &domain.TestVariableSource{OsEnv: osEnv}
		}
		tc.Spec.Variables = append(tc.Spec.Variables, v)
	}
	for _, section := range []string{testrun.SectionSetup, testrun.SectionSteps, testrun.SectionTeardown} {
		var steps []domain.TestStep
		for i, m := range c.sections[section] {
			steps = append(steps, m.dump(fmt.Sprintf("%s[%d]", section, i), &problems))
		}
		switch section {
		case testrun.SectionSetup:
			tc.Spec.Setup = steps
		case testrun.SectionSteps:
			if steps != nil {
				tc.Spec.Steps = steps
			}
		case testrun.SectionTeardown:
			tc.Spec.Teardown = steps
		}
	}
	return tc, problems
}

// current returns the test case being edited, applying the YAML tab first
// when it is open. ok is false when the YAML does not parse.
func (c *Container) current() (tc *domain.TestCase, problems []testrun.Problem, ok bool) {
	if c.tab == tabYAML && !c.applyYAML() {
		return nil, nil, false
	}
	tc, problems = c.dump()
	return tc, problems, true
}

// check refreshes the problems list after an edit.
func (c *Container) check() {
	if c.checked {
		return
	}
	c.checked = true
	if c.tab == tabYAML {
		return
	}
	tc, problems := c.dump()
	var find testrun.RequestFinder
	if c.deps.Catalog != nil {
		find = testrun.Finder(c.deps.Catalog)
	}
	c.problems = append(problems, testrun.Validate(tc, find)...)
}

func (c *Container) Save() error {
	tc, _, ok := c.current()
	if !ok {
		return fmt.Errorf("the YAML has an error: %s", c.yamlErr)
	}
	if err := c.deps.Repo.UpdateTestCase(tc); err != nil {
		return err
	}
	c.tc = tc
	c.dirty = false
	for _, steps := range c.sections {
		for _, m := range steps {
			m.markSaved()
		}
	}
	if c.tab == tabYAML && c.yamlEd != nil {
		c.yamlBase = string(c.yamlEd.Bytes())
	}
	c.deps.ReportDirty(false)
	c.deps.ReportTitle(tc.GetName())
	if c.deps.Report.Saved != nil {
		c.deps.Report.Saved()
	}
	c.deps.Toast("Test case saved")
	return nil
}

// Send runs the whole case, as ⌘Enter does for a request.
func (c *Container) Send() { c.start(nil) }

// start runs the case, or only the given steps of it.
func (c *Container) start(only []string) {
	if c.running || c.deps.Tests == nil {
		return
	}
	tc, formProblems, ok := c.current()
	if !ok {
		c.deps.ShowError(fmt.Errorf("fix the YAML before running: %s", c.yamlErr))
		return
	}
	c.checked = false
	c.check()
	if len(formProblems) > 0 || len(c.problems) > 0 {
		n := len(c.problems)
		if c.tab == tabYAML {
			n = len(formProblems) + len(testrun.Validate(tc, testrun.Finder(c.deps.Catalog)))
		}
		c.deps.ShowError(fmt.Errorf("the test case has %d problem(s); fix them before running", max(n, 1)))
		return
	}

	var env *domain.Environment
	if c.deps.ActiveEnv != nil {
		if active := c.deps.ActiveEnv(); active != nil {
			env = container.CopyEnv(active)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.running, c.cancel = true, cancel
	c.runSeq++
	c.run = &testrun.Run{CaseName: tc.GetName(), Status: testrun.StatusRunning, Started: time.Now()}
	if env != nil {
		c.run.EnvName = env.GetName()
	}
	if only == nil {
		c.results = map[string]*testrun.StepResult{}
	}
	c.selected = -1

	runner := c.deps.Tests
	go func() {
		runner.Run(ctx, tc, testrun.Options{Env: env, Only: only, OnEvent: c.post})
	}()
}

// post queues an event from the run's goroutine for Layout.
func (c *Container) post(e testrun.Event) {
	c.mu.Lock()
	c.events = append(c.events, e)
	c.mu.Unlock()
	c.deps.WakeNow()
}

func (c *Container) stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// drain applies the events the run posted since the last frame.
func (c *Container) drain() {
	c.mu.Lock()
	events := c.events
	c.events = nil
	c.mu.Unlock()

	for _, e := range events {
		switch e.Kind {
		case testrun.EventStepStarted, testrun.EventStepFinished:
			s := *e.Step
			c.results[resultKey(s.Section, s.StepID)] = &s
			replaced := false
			for i := range c.run.Steps {
				if c.run.Steps[i].Section == s.Section && c.run.Steps[i].StepID == s.StepID {
					c.run.Steps[i] = s
					replaced = true
				}
			}
			if !replaced {
				c.run.Steps = append(c.run.Steps, s)
			}
			// Follow the run until the user picks a step.
			if e.Kind == testrun.EventStepFinished && s.Status != testrun.StatusPassed &&
				s.Status != testrun.StatusSkipped && c.selected < 0 {
				c.selected = len(c.run.Steps) - 1
				if replaced {
					c.selected = c.stepIndex(s.Section, s.StepID)
				}
			}
		case testrun.EventRunFinished:
			selected := c.selected
			c.run = e.Run
			c.selected = selected
			c.running, c.cancel = false, nil
			if c.run.Message != "" {
				c.deps.ShowError(fmt.Errorf("%s", c.run.Message))
			}
		}
	}
}

func (c *Container) stepIndex(section, id string) int {
	for i, s := range c.run.Steps {
		if s.Section == section && s.StepID == id {
			return i
		}
	}
	return -1
}

func resultKey(section, id string) string { return section + "/" + id }

// setTab switches editor tabs, carrying edits between the form and YAML.
func (c *Container) setTab(i int) {
	if i == c.tab {
		return
	}
	if c.tab == tabYAML {
		if !c.applyYAML() {
			return
		}
	}
	if i == tabYAML {
		tc, _ := c.dump()
		b, err := yaml.Marshal(tc)
		if err != nil {
			c.deps.ShowError(err)
			return
		}
		c.yamlBase = string(b)
		c.yamlErr = ""
		if c.yamlEd != nil {
			c.yamlEd.Close()
		}
		c.yamlEd = ui.NewEditor(b, highlight.Noop{})
	}
	c.tab = i
	c.checked = false
}

// applyYAML loads the YAML tab into the form. It reports false, and keeps
// the error for the tab to show, when the YAML is not a valid test case.
func (c *Container) applyYAML() bool {
	if c.yamlEd == nil {
		return true
	}
	text := string(c.yamlEd.Bytes())
	if text == c.yamlBase {
		return true
	}
	var tc domain.TestCase
	if err := yaml.UnmarshalStrict([]byte(text), &tc); err != nil {
		c.yamlErr = err.Error()
		return false
	}
	if tc.Kind != domain.KindTestCase {
		c.yamlErr = fmt.Sprintf("kind must be %s", domain.KindTestCase)
		return false
	}
	// The file keeps its identity; rename from the title instead.
	tc.MetaData = c.tc.MetaData
	c.load(&tc)
	c.yamlBase, c.yamlErr = text, ""
	c.markDirty()
	return true
}

func (c *Container) Layout(ctx *ui.Ctx) ui.View {
	c.drain()
	if c.running {
		ctx.Animate(50 * time.Millisecond)
	}
	c.check()
	th := ctx.Theme()
	split := container.SplitPaneAxis(prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit)
	return ui.Column(
		c.titleRow(th),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Splitter("tc-split-"+c.ID(), split, c.editorPane(ctx, th), c.resultsPane(ctx, th)).
			Percents(55, 45).
			HandleOnHover().
			Grow(1),
	).Grow(1)
}

func (c *Container) titleRow(th *theme.Theme) ui.View {
	id := c.ID()
	envName := "No environment"
	if c.deps.ActiveEnv != nil {
		if env := c.deps.ActiveEnv(); env != nil {
			envName = env.GetName()
		}
	}
	run := ui.Button("tc-run-"+id, ui.Text("Run")).Primary().IconStart(icons.Play).Hint("⌘↵").
		Disabled(c.deps.Tests == nil).
		OnClick(c.Send)
	if c.running {
		run = ui.Button("tc-stop-"+id, ui.Text("Stop")).IconStart(icons.CircleStop).OnClick(c.stop)
	}
	return ui.Row(
		ui.EditableLabel("tc-title-"+id, c.tc.MetaData.Name).
			Placeholder("Name").
			OnSave(func(s string) {
				s = strings.TrimSpace(s)
				if s == "" || s == c.tc.MetaData.Name {
					return
				}
				c.tc.MetaData.Name = s
				c.markDirty()
				c.deps.ReportTitle(s)
			}).
			Grow(1),
		ui.Row(
			ui.Caption("Env: "+envName),
			ui.IconButton("tc-export-"+id, icons.FileDown).Tooltip("Export to run with chapar-cli test").OnClick(c.Export),
			ui.Button("tc-save-"+id, ui.Text("Save")).IconStart(icons.Save).Hint("⌘S").
				Disabled(!c.Dirty()).
				OnClick(func() {
					if err := c.Save(); err != nil {
						c.deps.ShowError(err)
					}
				}),
			run,
		).Gap(th.Spacing.S).Align(ui.AlignCenter),
	).Gap(th.Spacing.S).
		Justify(ui.JustifyBetween).
		Align(ui.AlignCenter).
		PaddingTop(th.Spacing.XS).
		PaddingBottom(th.Spacing.S).
		PaddingLeft(th.Spacing.M).
		PaddingRight(th.Spacing.M)
}

// TabIcon marks test case tabs apart from requests.
func (c *Container) TabIcon(th *theme.Theme) (icons.Icon, render.Color) {
	return icons.FlaskConical, th.Accent
}
