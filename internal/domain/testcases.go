package domain

import (
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

const (
	KindTestCase = "TestCase"

	TestHookTypePythonScript = "pythonScript"
	TestHookTypeSetEnv       = "setEnv"

	TestAssertOperatorEquals       = "equals"
	TestAssertOperatorNotEquals    = "notEquals"
	TestAssertOperatorExists       = "exists"
	TestAssertOperatorNotExists    = "notExists"
	TestAssertOperatorContains     = "contains"
	TestAssertOperatorNotContains  = "notContains"
	TestAssertOperatorGreaterThan  = "greaterThan"
	TestAssertOperatorLessThan     = "lessThan"
	TestAssertOperatorMatchesRegex = "matchesRegex"

	TestResultStatusPending = "pending"
	TestResultStatusRunning = "running"
	TestResultStatusPassed  = "passed"
	TestResultStatusFailed  = "failed"
)

type TestCase struct {
	ApiVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	MetaData   TestCaseMeta `yaml:"metadata"`
	Spec       TestCaseSpec `yaml:"spec"`
}

type TestCaseMeta struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type TestCaseSpec struct {
	Description string         `yaml:"description"`
	Variables   []TestVariable `yaml:"variables"`
	Steps       []TestStep     `yaml:"steps"`
}

type TestVariable struct {
	Name      string               `yaml:"name"`
	Value     string               `yaml:"value,omitempty"`
	ValueFrom *TestVariableValueFrom `yaml:"valueFrom,omitempty"`
}

type TestVariableValueFrom struct {
	OsEnv string `yaml:"osEnv"`
	Type  string `yaml:"type"`
}

type TestStep struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	RunNaked    bool                `yaml:"runNaked"`
	Request     TestStepRequest     `yaml:"request"`
	Override    *TestStepOverride   `yaml:"override,omitempty"`
	Assert      *TestStepAssertion  `yaml:"assert,omitempty"`
	Before      []TestHook          `yaml:"before,omitempty"`
	After       []TestHook          `yaml:"after,omitempty"`
}

type TestStepRequest struct {
	Collection string `yaml:"collection"`
	Request    string `yaml:"request"`
}

type TestStepOverride struct {
	Headers     []KeyValue `yaml:"headers,omitempty"`
	PathParams  []KeyValue `yaml:"pathParams,omitempty"`
	QueryParams []KeyValue `yaml:"queryParams,omitempty"`
	Body        string     `yaml:"body,omitempty"`
	Variables   []KeyValue `yaml:"variables,omitempty"`
}

type TestStepAssertion struct {
	StatusCode int                      `yaml:"statusCode,omitempty"`
	Headers    []TestHeaderAssertion    `yaml:"headers,omitempty"`
	Body       []TestBodyAssertion      `yaml:"body,omitempty"`
}

type TestHeaderAssertion struct {
	Key      string `yaml:"key"`
	Value    string `yaml:"value"`
	Operator string `yaml:"operator,omitempty"`
}

type TestBodyAssertion struct {
	Path     string `yaml:"path"`
	Operator string `yaml:"operator"`
	Value    string `yaml:"value,omitempty"`
}

type TestHook struct {
	Type      string                `yaml:"type"`
	Script    string                `yaml:"script,omitempty"`
	Name      string                `yaml:"name,omitempty"`
	Value     string                `yaml:"value,omitempty"`
	ValueFrom *TestHookValueFrom    `yaml:"valueFrom,omitempty"`
}

type TestHookValueFrom struct {
	Type string `yaml:"type"`
	Path string `yaml:"path,omitempty"`
	Key  string `yaml:"key,omitempty"`
}

// Entity interface implementation
func (t *TestCase) ID() string {
	return t.MetaData.ID
}

func (t *TestCase) GetKind() string {
	return t.Kind
}

func (t *TestCase) GetName() string {
	return t.MetaData.Name
}

func (t *TestCase) SetName(name string) {
	t.MetaData.Name = name
}

func (t *TestCase) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(t)
}

// Clone creates a deep copy of the test case
func (t *TestCase) Clone() *TestCase {
	clone := *t
	clone.MetaData.ID = uuid.NewString()
	
	// Deep clone variables
	if len(t.Spec.Variables) > 0 {
		clone.Spec.Variables = make([]TestVariable, len(t.Spec.Variables))
		copy(clone.Spec.Variables, t.Spec.Variables)
	}
	
	// Deep clone steps
	if len(t.Spec.Steps) > 0 {
		clone.Spec.Steps = make([]TestStep, len(t.Spec.Steps))
		for i, step := range t.Spec.Steps {
			clone.Spec.Steps[i] = cloneTestStep(step)
		}
	}
	
	return &clone
}

func cloneTestStep(step TestStep) TestStep {
	clone := step
	
	if step.Override != nil {
		override := *step.Override
		if len(step.Override.Headers) > 0 {
			override.Headers = make([]KeyValue, len(step.Override.Headers))
			copy(override.Headers, step.Override.Headers)
		}
		if len(step.Override.PathParams) > 0 {
			override.PathParams = make([]KeyValue, len(step.Override.PathParams))
			copy(override.PathParams, step.Override.PathParams)
		}
		if len(step.Override.QueryParams) > 0 {
			override.QueryParams = make([]KeyValue, len(step.Override.QueryParams))
			copy(override.QueryParams, step.Override.QueryParams)
		}
		if len(step.Override.Variables) > 0 {
			override.Variables = make([]KeyValue, len(step.Override.Variables))
			copy(override.Variables, step.Override.Variables)
		}
		clone.Override = &override
	}
	
	if step.Assert != nil {
		assertion := *step.Assert
		if len(step.Assert.Headers) > 0 {
			assertion.Headers = make([]TestHeaderAssertion, len(step.Assert.Headers))
			copy(assertion.Headers, step.Assert.Headers)
		}
		if len(step.Assert.Body) > 0 {
			assertion.Body = make([]TestBodyAssertion, len(step.Assert.Body))
			copy(assertion.Body, step.Assert.Body)
		}
		clone.Assert = &assertion
	}
	
	if len(step.Before) > 0 {
		clone.Before = make([]TestHook, len(step.Before))
		copy(clone.Before, step.Before)
	}
	
	if len(step.After) > 0 {
		clone.After = make([]TestHook, len(step.After))
		copy(clone.After, step.After)
	}
	
	return clone
}

// NewTestCase creates a new test case with default values
func NewTestCase(name string) *TestCase {
	return &TestCase{
		ApiVersion: ApiVersion,
		Kind:       KindTestCase,
		MetaData: TestCaseMeta{
			ID:   uuid.NewString(),
			Name: name,
		},
		Spec: TestCaseSpec{
			Description: "Test case description",
			Variables:   []TestVariable{},
			Steps:       []TestStep{},
		},
	}
}

// TestResult represents the result of a test case execution
type TestResult struct {
	TestCaseID   string           `json:"testCaseId"`
	TestCaseName string           `json:"testCaseName"`
	Status       string           `json:"status"`
	StartTime    time.Time        `json:"startTime"`
	EndTime      time.Time        `json:"endTime"`
	Duration     time.Duration    `json:"duration"`
	TotalSteps   int              `json:"totalSteps"`
	PassedSteps  int              `json:"passedSteps"`
	FailedSteps  int              `json:"failedSteps"`
	StepResults  []TestStepResult `json:"stepResults"`
	Error        string           `json:"error,omitempty"`
}

// TestStepResult represents the result of a single test step execution
type TestStepResult struct {
	StepName         string                     `json:"stepName"`
	StepIndex        int                        `json:"stepIndex"`
	Status           string                     `json:"status"`
	StartTime        time.Time                  `json:"startTime"`
	EndTime          time.Time                  `json:"endTime"`
	Duration         time.Duration              `json:"duration"`
	RequestMethod    string                     `json:"requestMethod,omitempty"`
	RequestURL       string                     `json:"requestUrl,omitempty"`
	ResponseStatus   int                        `json:"responseStatus,omitempty"`
	ResponseSize     int                        `json:"responseSize,omitempty"`
	AssertionResults []AssertionResult          `json:"assertionResults,omitempty"`
	Error            string                     `json:"error,omitempty"`
	BeforeHookErrors []string                   `json:"beforeHookErrors,omitempty"`
	AfterHookErrors  []string                   `json:"afterHookErrors,omitempty"`
}

// AssertionResult represents the result of a single assertion
type AssertionResult struct {
	Type     string `json:"type"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message,omitempty"`
}

// IsRunning returns true if the test is currently running
func (r *TestResult) IsRunning() bool {
	return r.Status == TestResultStatusRunning
}

// IsPassed returns true if the test passed
func (r *TestResult) IsPassed() bool {
	return r.Status == TestResultStatusPassed
}

// IsFailed returns true if the test failed
func (r *TestResult) IsFailed() bool {
	return r.Status == TestResultStatusFailed
}

// GetPassRate returns the percentage of passed steps
func (r *TestResult) GetPassRate() float64 {
	if r.TotalSteps == 0 {
		return 0
	}
	return float64(r.PassedSteps) / float64(r.TotalSteps) * 100
}
