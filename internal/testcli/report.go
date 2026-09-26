package testcli

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/testrun"
)

// console prints runs to a terminal as they go.
type console struct {
	w     io.Writer
	color bool
	width int // of the step name column for the current case
}

const (
	green  = "\x1b[32m"
	red    = "\x1b[31m"
	yellow = "\x1b[33m"
	dim    = "\x1b[2m"
	bold   = "\x1b[1m"
	reset  = "\x1b[0m"
)

// line prints s and a newline. A console that cannot be written to has
// nowhere to report it, so errors are dropped.
func (c *console) line(s string) {
	_, _ = fmt.Fprintln(c.w, s)
}

func (c *console) paint(code, s string) string {
	if !c.color {
		return s
	}
	return code + s + reset
}

func (c *console) caseStarted(tc *domain.TestCase, env string) {
	c.width = 0
	for _, s := range tc.AllSteps() {
		c.width = max(c.width, utf8.RuneCountInString(displayName(s)))
	}
	line := c.paint(bold, tc.GetName())
	if env != "" {
		line += c.paint(dim, " · env "+env)
	}
	c.line(line)
}

func displayName(s domain.TestStep) string {
	switch {
	case s.Name != "":
		return s.Name
	case s.Request.Ref != "":
		return s.Request.Ref
	}
	return s.ID
}

func (c *console) event(e testrun.Event) {
	if e.Kind != testrun.EventStepFinished {
		return
	}
	s := e.Step
	name := s.Name + strings.Repeat(" ", max(0, c.width-utf8.RuneCountInString(s.Name)))

	var mark string
	switch s.Status {
	case testrun.StatusPassed:
		mark = c.paint(green, "✓")
	case testrun.StatusFailed, testrun.StatusError:
		mark = c.paint(red, "✗")
	default:
		mark = c.paint(yellow, "○")
	}

	line := "  " + mark + " " + name
	if s.Response != nil {
		line += fmt.Sprintf("  %3d", s.Response.Status)
	}
	if s.Status != testrun.StatusSkipped {
		line += c.paint(dim, "  "+duration(s.Duration))
	}
	if s.Attempts > 1 {
		line += c.paint(dim, fmt.Sprintf("  %d attempts", s.Attempts))
	}
	if s.Section != testrun.SectionSteps {
		line += c.paint(dim, "  ("+s.Section+")")
	}
	if s.Status == testrun.StatusSkipped || s.Status == testrun.StatusCancelled {
		line += c.paint(dim, "  "+s.Message)
	}
	c.line(line)

	if s.Status != testrun.StatusFailed && s.Status != testrun.StatusError {
		return
	}
	shown := false
	for _, a := range s.Assertions {
		if !a.Passed {
			c.line("      " + c.paint(red, describe(a)))
			shown = true
		}
	}
	for _, cp := range s.Captures {
		if cp.Error != "" {
			c.line("      " + c.paint(red, "capture "+cp.Var+": "+cp.Error))
			shown = true
		}
	}
	if !shown && s.Message != "" {
		c.line("      " + c.paint(red, s.Message))
	}
}

// describe says what an assertion checked and why it failed.
func describe(a testrun.AssertionResult) string {
	if a.Source == testrun.SourceScript {
		return "script test " + a.Path + ": " + a.Message
	}
	subject := a.Target
	if a.Key != "" {
		subject += " " + a.Key
	}
	if a.Path != "" {
		subject += " " + a.Path
	}
	check := subject + " " + a.Op
	if a.Expected != nil {
		b, _ := json.Marshal(a.Expected)
		check += " " + string(b)
	}
	return check + ": " + a.Message
}

func (c *console) caseFinished(run *testrun.Run) {
	n := run.Counts()
	var parts []string
	for _, st := range []testrun.Status{testrun.StatusPassed, testrun.StatusFailed, testrun.StatusError, testrun.StatusSkipped, testrun.StatusCancelled} {
		if n[st] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n[st], st))
		}
	}
	status := string(run.Status)
	if run.Status == testrun.StatusPassed {
		status = c.paint(green, status)
	} else {
		status = c.paint(red, status)
	}
	if run.Message != "" {
		c.line("  " + c.paint(red, run.Message))
	}
	c.line(fmt.Sprintf("%s · %s · %s\n", status, strings.Join(parts, ", "), duration(run.Duration)))
}

func (c *console) summary(runs []*testrun.Run, took time.Duration) {
	passed := 0
	for _, r := range runs {
		if r.Status == testrun.StatusPassed {
			passed++
		}
	}
	line := fmt.Sprintf("%d of %d test cases passed · %s", passed, len(runs), duration(took))
	if passed == len(runs) {
		line = c.paint(green, line)
	} else {
		line = c.paint(red, line)
	}
	c.line(line)
}

// duration formats d for the console.
func duration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return "<1ms"
	case d > time.Second:
		return d.Round(10 * time.Millisecond).String()
	}
	return d.Round(time.Millisecond).String()
}

// JUnit XML, the format CI systems read test results from.
type junitSuites struct {
	XMLName  xml.Name     `xml:"testsuites"`
	Name     string       `xml:"name,attr"`
	Tests    int          `xml:"tests,attr"`
	Failures int          `xml:"failures,attr"`
	Errors   int          `xml:"errors,attr"`
	Skipped  int          `xml:"skipped,attr"`
	Time     float64      `xml:"time,attr"`
	Suites   []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name      string      `xml:"name,attr"`
	Tests     int         `xml:"tests,attr"`
	Failures  int         `xml:"failures,attr"`
	Errors    int         `xml:"errors,attr"`
	Skipped   int         `xml:"skipped,attr"`
	Time      float64     `xml:"time,attr"`
	Timestamp string      `xml:"timestamp,attr"`
	Cases     []junitCase `xml:"testcase"`
	SystemErr string      `xml:"system-err,omitempty"`
}

type junitCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *junitProblem `xml:"failure"`
	Error     *junitProblem `xml:"error"`
	Skipped   *junitProblem `xml:"skipped"`
}

type junitProblem struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

func writeJUnit(w io.Writer, runs []*testrun.Run, took time.Duration) error {
	out := junitSuites{Name: "chapar", Time: took.Seconds()}
	for _, run := range runs {
		suite := junitSuite{
			Name:      run.CaseName,
			Time:      run.Duration.Seconds(),
			Timestamp: run.Started.UTC().Format("2006-01-02T15:04:05"),
			SystemErr: run.Message,
		}
		for _, s := range run.Steps {
			tc := junitCase{Name: s.Name, Classname: run.CaseName + "." + s.Section, Time: s.Duration.Seconds()}
			problem := &junitProblem{Message: s.Message, Text: details(s)}
			switch s.Status {
			case testrun.StatusFailed:
				tc.Failure = problem
				suite.Failures++
			case testrun.StatusError, testrun.StatusCancelled:
				tc.Error = problem
				suite.Errors++
			case testrun.StatusSkipped:
				tc.Skipped = &junitProblem{Message: s.Message}
				suite.Skipped++
			}
			suite.Cases = append(suite.Cases, tc)
		}
		suite.Tests = len(suite.Cases)
		out.Tests += suite.Tests
		out.Failures += suite.Failures
		out.Errors += suite.Errors
		out.Skipped += suite.Skipped
		out.Suites = append(out.Suites, suite)
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// details is the failure text of a step: what was sent, what came back
// and every assertion that failed.
func details(s testrun.StepResult) string {
	var b strings.Builder
	if r := s.Request; r != nil {
		_, _ = fmt.Fprintf(&b, "%s %s %s\n", r.Protocol, r.Method, r.URL)
	}
	if r := s.Response; r != nil {
		_, _ = fmt.Fprintf(&b, "response %d, %d bytes, %s\n", r.Status, r.Size, r.Time)
	}
	for _, a := range s.Assertions {
		if !a.Passed {
			b.WriteString(describe(a) + "\n")
		}
	}
	for _, c := range s.Captures {
		if c.Error != "" {
			b.WriteString("capture " + c.Var + ": " + c.Error + "\n")
		}
	}
	return b.String()
}

func writeJSON(w io.Writer, runs []*testrun.Run) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		Runs []*testrun.Run `json:"runs"`
	}{runs})
}

// report is one --report flag: a format and the file it goes to.
type report struct {
	format, path string
}

func (r report) write(runs []*testrun.Run, took time.Duration) error {
	f, err := os.Create(r.path)
	if err != nil {
		return err
	}
	switch r.format {
	case "junit":
		err = writeJUnit(f, runs, took)
	case "json":
		err = writeJSON(f, runs)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
