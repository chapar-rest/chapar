package testrun

import (
	"time"
)

// Status of a run or a step.
type Status string

const (
	StatusRunning Status = "running"
	StatusPassed  Status = "passed"
	// StatusFailed means the step ran and an assertion or capture failed.
	StatusFailed Status = "failed"
	// StatusError means the step could not run: its request is missing,
	// the send failed, timed out, or a post-request action raised.
	StatusError     Status = "error"
	StatusSkipped   Status = "skipped"
	StatusCancelled Status = "cancelled"
)

// Sections of a test case, in run order.
const (
	SectionSetup    = "setup"
	SectionSteps    = "steps"
	SectionTeardown = "teardown"
)

// Run is the result of running one test case.
type Run struct {
	ID       string        `json:"id"`
	CaseID   string        `json:"caseId"`
	CaseName string        `json:"caseName"`
	EnvName  string        `json:"envName,omitempty"`
	Status   Status        `json:"status"`
	Started  time.Time     `json:"started"`
	Duration time.Duration `json:"duration"`
	Steps    []StepResult  `json:"steps"`
	// Message is a run-level problem, such as failing to save the env.
	Message string `json:"message,omitempty"`
}

// Counts returns how many steps ended in each status.
func (r *Run) Counts() map[Status]int {
	out := map[Status]int{}
	for _, s := range r.Steps {
		out[s.Status]++
	}
	return out
}

// StepResult is the result of one step.
type StepResult struct {
	Section  string           `json:"section"`
	StepID   string           `json:"stepId"`
	Name     string           `json:"name"`
	Status   Status           `json:"status"`
	Attempts int              `json:"attempts,omitempty"`
	Duration time.Duration    `json:"duration,omitempty"`
	Request  *RequestSummary  `json:"request,omitempty"`
	Response *ResponseSummary `json:"response,omitempty"`

	Assertions []AssertionResult `json:"assertions,omitempty"`
	Captures   []CaptureResult   `json:"captures,omitempty"`
	// Message says why the step failed, errored or was skipped.
	Message string `json:"message,omitempty"`
}

// RequestSummary is what a step sent, before variables were substituted.
type RequestSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Method   string `json:"method,omitempty"`
	URL      string `json:"url,omitempty"`
}

// maxBody is how much of a response body a result keeps.
const maxBody = 64 << 10

// ResponseSummary is what a step got back.
type ResponseSummary struct {
	Status  int               `json:"status"`
	Size    int               `json:"size"`
	Time    time.Duration     `json:"time"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	// Truncated is set when Body holds only the first maxBody bytes.
	Truncated bool `json:"truncated,omitempty"`
}

// EventKind says what happened in an Event.
type EventKind int

const (
	EventStepStarted EventKind = iota
	EventStepFinished
	EventRunFinished
)

// Event reports progress while a run is going. Step is set for step
// events, Run for EventRunFinished.
type Event struct {
	Kind EventKind
	Step *StepResult
	Run  *Run
}
