package testcase

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/internal/variables"
)

type Runner struct {
	requests       *state.Requests
	environments   *state.Environments
	egressService  *egress.Service
	scriptExecutor scripting.Executor
}

func NewRunner(
	requests *state.Requests,
	environments *state.Environments,
	egressService *egress.Service,
	scriptExecutor scripting.Executor,
) *Runner {
	return &Runner{
		requests:       requests,
		environments:   environments,
		egressService:  egressService,
		scriptExecutor: scriptExecutor,
	}
}

// Run executes a test case and returns the results
func (r *Runner) Run(ctx context.Context, testCase *domain.TestCase, environmentID string) (*domain.TestResult, error) {
	result := &domain.TestResult{
		TestCaseID:   testCase.ID(),
		TestCaseName: testCase.GetName(),
		Status:       domain.TestResultStatusRunning,
		StartTime:    time.Now(),
		TotalSteps:   len(testCase.Spec.Steps),
		StepResults:  make([]domain.TestStepResult, 0, len(testCase.Spec.Steps)),
	}

	// Initialize test variables
	testVars := make(map[string]string)
	for _, v := range testCase.Spec.Variables {
		if v.Value != "" {
			testVars[v.Name] = v.Value
		} else if v.ValueFrom != nil {
			// Load from OS environment
			if v.ValueFrom.OsEnv != "" {
				testVars[v.Name] = os.Getenv(v.ValueFrom.OsEnv)
			}
		}
	}

	// Execute each step
	for i, step := range testCase.Spec.Steps {
		select {
		case <-ctx.Done():
			result.Status = domain.TestResultStatusFailed
			result.Error = "Test execution cancelled"
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, ctx.Err()
		default:
		}

		stepResult := r.executeStep(ctx, step, i, testVars, environmentID)
		result.StepResults = append(result.StepResults, stepResult)

		if stepResult.Status == domain.TestResultStatusPassed {
			result.PassedSteps++
		} else {
			result.FailedSteps++
		}

		// Stop on first failure if needed (could be configurable)
		if stepResult.Status == domain.TestResultStatusFailed {
			result.Status = domain.TestResultStatusFailed
			break
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if result.Status == domain.TestResultStatusRunning {
		if result.FailedSteps == 0 {
			result.Status = domain.TestResultStatusPassed
		} else {
			result.Status = domain.TestResultStatusFailed
		}
	}

	return result, nil
}

func (r *Runner) executeStep(
	ctx context.Context,
	step domain.TestStep,
	stepIndex int,
	testVars map[string]string,
	environmentID string,
) domain.TestStepResult {
	stepResult := domain.TestStepResult{
		StepName:  step.Name,
		StepIndex: stepIndex,
		Status:    domain.TestResultStatusRunning,
		StartTime: time.Now(),
	}

	// Execute before hooks
	if len(step.Before) > 0 {
		for _, hook := range step.Before {
			if err := r.executeHook(hook, testVars, nil, environmentID); err != nil {
				stepResult.BeforeHookErrors = append(stepResult.BeforeHookErrors, err.Error())
			}
		}
	}

	// Find the request
	request, err := r.findRequest(step.Request.Collection, step.Request.Request)
	if err != nil {
		stepResult.Status = domain.TestResultStatusFailed
		stepResult.Error = fmt.Sprintf("Failed to find request: %v", err)
		stepResult.EndTime = time.Now()
		stepResult.Duration = stepResult.EndTime.Sub(stepResult.StartTime)
		return stepResult
	}

	// Clone the request to avoid modifying the original
	requestClone := request.Clone()

	// Apply overrides
	if step.Override != nil {
		r.applyOverrides(requestClone, step.Override, testVars)
	}

	// Apply test variables to the request
	r.applyTestVariables(requestClone, testVars)

	// Save the cloned request temporarily in state for execution
	tempID := requestClone.ID()
	r.requests.AddRequest(requestClone)
	defer r.requests.RemoveRequest(requestClone)

	// Store request details
	if requestClone.MetaData.Type == domain.RequestTypeHTTP {
		stepResult.RequestMethod = requestClone.Spec.HTTP.Method
		stepResult.RequestURL = requestClone.Spec.HTTP.URL
	}

	// Execute the request
	response, err := r.egressService.Send(tempID, environmentID)
	if err != nil {
		stepResult.Status = domain.TestResultStatusFailed
		stepResult.Error = fmt.Sprintf("Request execution failed: %v", err)
		stepResult.EndTime = time.Now()
		stepResult.Duration = stepResult.EndTime.Sub(stepResult.StartTime)
		return stepResult
	}

	// Extract response details
	egressResp, ok := response.(*egress.Response)
	if ok {
		stepResult.ResponseStatus = egressResp.StatusCode
		stepResult.ResponseSize = egressResp.Size
	}

	// Execute assertions
	if step.Assert != nil {
		assertionResults := r.executeAssertions(step.Assert, egressResp)
		stepResult.AssertionResults = assertionResults

		// Check if any assertion failed
		for _, ar := range assertionResults {
			if !ar.Passed {
				stepResult.Status = domain.TestResultStatusFailed
				if stepResult.Error == "" {
					stepResult.Error = "Assertion failed"
				}
				break
			}
		}
	}

	// Execute after hooks
	if len(step.After) > 0 {
		for _, hook := range step.After {
			if err := r.executeHook(hook, testVars, egressResp, environmentID); err != nil {
				stepResult.AfterHookErrors = append(stepResult.AfterHookErrors, err.Error())
			}
		}
	}

	// Mark as passed if no failures
	if stepResult.Status == domain.TestResultStatusRunning {
		stepResult.Status = domain.TestResultStatusPassed
	}

	stepResult.EndTime = time.Now()
	stepResult.Duration = stepResult.EndTime.Sub(stepResult.StartTime)

	return stepResult
}

func (r *Runner) findRequest(collectionName, requestName string) (*domain.Request, error) {
	// If collection name is empty, search in standalone requests
	if collectionName == "" {
		requests := r.requests.GetRequests()
		for _, req := range requests {
			if req.GetName() == requestName && req.CollectionID == "" {
				return req, nil
			}
		}
		return nil, fmt.Errorf("standalone request %s not found", requestName)
	}

	// Search in collection requests
	requests := r.requests.GetRequests()
	for _, req := range requests {
		if req.GetName() == requestName && req.CollectionName == collectionName {
			return req, nil
		}
	}

	return nil, fmt.Errorf("request %s not found in collection %s", requestName, collectionName)
}

func (r *Runner) applyOverrides(request *domain.Request, override *domain.TestStepOverride, testVars map[string]string) {
	if request.MetaData.Type != domain.RequestTypeHTTP {
		return
	}

	httpSpec := request.Spec.HTTP
	if httpSpec == nil || httpSpec.Request == nil {
		return
	}

	// Override headers
	if len(override.Headers) > 0 {
		for _, header := range override.Headers {
			found := false
			for i, h := range httpSpec.Request.Headers {
				if h.Key == header.Key {
					httpSpec.Request.Headers[i].Value = header.Value
					found = true
					break
				}
			}
			if !found {
				httpSpec.Request.Headers = append(httpSpec.Request.Headers, header)
			}
		}
	}

	// Override path params
	if len(override.PathParams) > 0 {
		for _, param := range override.PathParams {
			found := false
			for i, p := range httpSpec.Request.PathParams {
				if p.Key == param.Key {
					httpSpec.Request.PathParams[i].Value = param.Value
					found = true
					break
				}
			}
			if !found {
				httpSpec.Request.PathParams = append(httpSpec.Request.PathParams, param)
			}
		}
	}

	// Override query params
	if len(override.QueryParams) > 0 {
		for _, param := range override.QueryParams {
			found := false
			for i, p := range httpSpec.Request.QueryParams {
				if p.Key == param.Key {
					httpSpec.Request.QueryParams[i].Value = param.Value
					found = true
					break
				}
			}
			if !found {
				httpSpec.Request.QueryParams = append(httpSpec.Request.QueryParams, param)
			}
		}
	}

	// Override body
	if override.Body != "" {
		httpSpec.Request.Body.Data = override.Body
	}

	// Override variables
	if len(override.Variables) > 0 {
		for _, v := range override.Variables {
			testVars[v.Key] = v.Value
		}
	}
}

func (r *Runner) applyTestVariables(request *domain.Request, testVars map[string]string) {
	if request.MetaData.Type != domain.RequestTypeHTTP {
		return
	}

	httpSpec := request.Spec.HTTP
	if httpSpec == nil {
		return
	}

	// Apply variables to URL
	httpSpec.URL = r.replaceVariables(httpSpec.URL, testVars)

	if httpSpec.Request == nil {
		return
	}

	// Apply variables to headers
	for i, header := range httpSpec.Request.Headers {
		httpSpec.Request.Headers[i].Value = r.replaceVariables(header.Value, testVars)
	}

	// Apply variables to path params
	for i, param := range httpSpec.Request.PathParams {
		httpSpec.Request.PathParams[i].Value = r.replaceVariables(param.Value, testVars)
	}

	// Apply variables to query params
	for i, param := range httpSpec.Request.QueryParams {
		httpSpec.Request.QueryParams[i].Value = r.replaceVariables(param.Value, testVars)
	}

	// Apply variables to body
	httpSpec.Request.Body.Data = r.replaceVariables(httpSpec.Request.Body.Data, testVars)
}

func (r *Runner) replaceVariables(input string, testVars map[string]string) string {
	result := input

	// First apply built-in variables
	builtinVars := variables.GetVariables()
	for key, value := range builtinVars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}

	// Then apply test variables
	for key, value := range testVars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}

	return result
}

func (r *Runner) executeAssertions(assert *domain.TestStepAssertion, response *egress.Response) []domain.AssertionResult {
	var results []domain.AssertionResult

	// Assert status code
	if assert.StatusCode > 0 {
		passed := response.StatusCode == assert.StatusCode
		results = append(results, domain.AssertionResult{
			Type:     "statusCode",
			Field:    "Status Code",
			Operator: domain.TestAssertOperatorEquals,
			Expected: fmt.Sprintf("%d", assert.StatusCode),
			Actual:   fmt.Sprintf("%d", response.StatusCode),
			Passed:   passed,
			Message:  fmt.Sprintf("Expected status code %d, got %d", assert.StatusCode, response.StatusCode),
		})
	}

	// Assert headers
	for _, headerAssert := range assert.Headers {
		operator := headerAssert.Operator
		if operator == "" {
			operator = domain.TestAssertOperatorEquals
		}

		actualValue := response.ResponseHeaders[headerAssert.Key]
		passed := r.evaluateAssertion(actualValue, headerAssert.Value, operator)

		results = append(results, domain.AssertionResult{
			Type:     "header",
			Field:    headerAssert.Key,
			Operator: operator,
			Expected: headerAssert.Value,
			Actual:   actualValue,
			Passed:   passed,
			Message:  fmt.Sprintf("Header %s assertion", headerAssert.Key),
		})
	}

	// Assert body
	for _, bodyAssert := range assert.Body {
		var actualValue interface{}
		var err error

		// Extract value using JSONPath
		if bodyAssert.Path != "" {
			actualValue, err = jsonpath.Get(string(response.Body), bodyAssert.Path)
			if err != nil {
				results = append(results, domain.AssertionResult{
					Type:     "body",
					Field:    bodyAssert.Path,
					Operator: bodyAssert.Operator,
					Expected: bodyAssert.Value,
					Actual:   "",
					Passed:   false,
					Message:  fmt.Sprintf("Failed to evaluate JSONPath: %v", err),
				})
				continue
			}
		}

		actualStr := fmt.Sprintf("%v", actualValue)
		passed := r.evaluateAssertion(actualStr, bodyAssert.Value, bodyAssert.Operator)

		results = append(results, domain.AssertionResult{
			Type:     "body",
			Field:    bodyAssert.Path,
			Operator: bodyAssert.Operator,
			Expected: bodyAssert.Value,
			Actual:   actualStr,
			Passed:   passed,
			Message:  fmt.Sprintf("Body assertion for path %s", bodyAssert.Path),
		})
	}

	return results
}

func (r *Runner) evaluateAssertion(actual, expected, operator string) bool {
	switch operator {
	case domain.TestAssertOperatorEquals:
		return actual == expected
	case domain.TestAssertOperatorNotEquals:
		return actual != expected
	case domain.TestAssertOperatorExists:
		return actual != "" && actual != "<nil>"
	case domain.TestAssertOperatorNotExists:
		return actual == "" || actual == "<nil>"
	case domain.TestAssertOperatorContains:
		return strings.Contains(actual, expected)
	case domain.TestAssertOperatorNotContains:
		return !strings.Contains(actual, expected)
	case domain.TestAssertOperatorMatchesRegex:
		matched, err := regexp.MatchString(expected, actual)
		return err == nil && matched
	default:
		return false
	}
}

func (r *Runner) executeHook(hook domain.TestHook, testVars map[string]string, response *egress.Response, environmentID string) error {
	switch hook.Type {
	case domain.TestHookTypeSetEnv:
		return r.executeSetEnvHook(hook, testVars, response, environmentID)
	case domain.TestHookTypePythonScript:
		return r.executePythonHook(hook, testVars, response, environmentID)
	default:
		return fmt.Errorf("unknown hook type: %s", hook.Type)
	}
}

func (r *Runner) executeSetEnvHook(hook domain.TestHook, testVars map[string]string, response *egress.Response, environmentID string) error {
	var value string

	if hook.Value != "" {
		value = hook.Value
	} else if hook.ValueFrom != nil && response != nil {
		// Extract value from response
		switch hook.ValueFrom.Type {
		case "body":
			if hook.ValueFrom.Path != "" {
				extracted, err := jsonpath.Get(string(response.Body), hook.ValueFrom.Path)
				if err != nil {
					return fmt.Errorf("failed to extract value from body: %w", err)
				}
				value = fmt.Sprintf("%v", extracted)
			}
		case "headers":
			if hook.ValueFrom.Key != "" {
				value = response.ResponseHeaders[hook.ValueFrom.Key]
			}
		case "cookie":
			if hook.ValueFrom.Key != "" {
				for _, cookie := range response.Cookies {
					if cookie.Name == hook.ValueFrom.Key {
						value = cookie.Value
						break
					}
				}
			}
		}
	}

	// Set in test variables
	if hook.Name != "" {
		testVars[hook.Name] = value

		// Also set in environment if environment is active
		if environmentID != "" {
			env := r.environments.GetEnvironment(environmentID)
			if env != nil {
				found := false
				for i, kv := range env.Spec.Values {
					if kv.Key == hook.Name {
						env.Spec.Values[i].Value = value
						found = true
						break
					}
				}
				if !found {
					env.Spec.Values = append(env.Spec.Values, domain.KeyValue{
						Key:    hook.Name,
						Value:  value,
						Enable: true,
					})
				}
				r.environments.UpdateEnvironment(env, state.SourceView, false)
			}
		}
	}

	return nil
}

func (r *Runner) executePythonHook(hook domain.TestHook, testVars map[string]string, response *egress.Response, environmentID string) error {
	if r.scriptExecutor == nil {
		return fmt.Errorf("script executor not available")
	}

	// Prepare script parameters
	scriptParams := &scripting.ExecParams{
		Req: &scripting.RequestData{
			Method:      "",
			URL:         "",
			Headers:     make(map[string]string),
			QueryParams: make(map[string]string),
			PathParams:  make(map[string]string),
			Body:        "",
		},
		Res: &scripting.ResponseData{
			StatusCode: 0,
			Headers:    make(map[string]string),
			Body:       "",
		},
	}

	// Get environment if available
	if environmentID != "" {
		env := r.environments.GetEnvironment(environmentID)
		if env != nil {
			scriptParams.Env = env
		}
	}

	if response != nil {
		scriptParams.Res.StatusCode = response.StatusCode
		scriptParams.Res.Headers = response.ResponseHeaders
		scriptParams.Res.Body = string(response.Body)
	}

	// Execute script
	result, err := r.scriptExecutor.Execute(context.Background(), hook.Script, scriptParams)
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	// Update test variables with script output
	if result != nil && len(result.SetEnvironments) > 0 {
		for key, value := range result.SetEnvironments {
			// Convert interface{} to string
			valueStr := fmt.Sprintf("%v", value)
			testVars[key] = valueStr

			// Also update environment if active
			if environmentID != "" {
				env := r.environments.GetEnvironment(environmentID)
				if env != nil {
					found := false
					for i, kv := range env.Spec.Values {
						if kv.Key == key {
							env.Spec.Values[i].Value = valueStr
							found = true
							break
						}
					}
					if !found {
						env.Spec.Values = append(env.Spec.Values, domain.KeyValue{
							Key:    key,
							Value:  valueStr,
							Enable: true,
						})
					}
					r.environments.UpdateEnvironment(env, state.SourceView, false)
				}
			}
		}
	}

	return nil
}
