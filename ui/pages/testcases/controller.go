package testcases

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/internal/testcase"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Controller struct {
	view  *View
	state *state.TestCases

	repo repository.RepositoryV2

	testRunner     *testcase.Runner
	requests       *state.Requests
	environments   *state.Environments
	egressService  *egress.Service
	scriptExecutor scripting.Executor

	explorer *explorer.Explorer

	activeTabID string

	// Map to store YAML content for each test case
	yamlContent map[string]string
}

func NewController(
	view *View,
	repo repository.RepositoryV2,
	testState *state.TestCases,
	requests *state.Requests,
	environments *state.Environments,
	egressService *egress.Service,
	scriptExecutor scripting.Executor,
	explorer *explorer.Explorer,
) *Controller {
	c := &Controller{
		view:           view,
		state:          testState,
		repo:           repo,
		requests:       requests,
		environments:   environments,
		egressService:  egressService,
		scriptExecutor: scriptExecutor,
		explorer:       explorer,
		yamlContent:    make(map[string]string),
	}

	c.testRunner = testcase.NewRunner(requests, environments, egressService, scriptExecutor)

	view.SetOnNewTestCase(c.onNewTestCase)
	view.SetOnImportTestCase(c.onImportTestCase)
	view.SetOnTitleChanged(c.onTitleChanged)
	view.SetOnTreeViewNodeClicked(c.onTreeViewNodeClicked)
	view.SetOnTabSelected(c.onTabSelected)
	view.SetOnYAMLChanged(c.onYAMLChanged)
	view.SetOnSave(c.onSave)
	view.SetOnRun(c.onRun)
	view.SetOnTabClose(c.onTabClose)
	view.SetOnTreeViewMenuClicked(c.onTreeViewMenuClicked)
	testState.AddTestCaseChangeListener(c.onTestCaseChange)

	return c
}

func (c *Controller) OpenTestCase(id string) {
	c.openTestCase(id)
}

func (c *Controller) onNewTestCase() {
	tc := domain.NewTestCase("New Test Case")
	
	// Create a template YAML
	templateYAML := c.generateTemplateYAML(tc)
	
	if err := c.repo.CreateTestCase(tc); err != nil {
		c.view.showError(fmt.Errorf("failed to create test case: %w", err))
		return
	}

	tc, err := c.state.GetPersistedTestCase(tc.MetaData.ID)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get test case from file: %w", err))
		return
	}

	c.state.AddTestCase(tc, state.SourceController)
	c.view.AddTreeViewNode(tc)
	c.yamlContent[tc.MetaData.ID] = templateYAML
	
	// Open the test case immediately
	c.openTestCase(tc.MetaData.ID)
}

func (c *Controller) generateTemplateYAML(tc *domain.TestCase) string {
	// Add a sample step to the template
	tc.Spec.Description = "Test case description"
	tc.Spec.Variables = []domain.TestVariable{
		{
			Name:  "example_var",
			Value: "example_value",
		},
	}
	tc.Spec.Steps = []domain.TestStep{
		{
			Name:        "Example Step",
			Description: "This is an example step",
			Request: domain.TestStepRequest{
				Collection: "MyCollection",
				Request:    "MyRequest",
			},
			Assert: &domain.TestStepAssertion{
				StatusCode: 200,
				Body: []domain.TestBodyAssertion{
					{
						Path:     "$.status",
						Operator: domain.TestAssertOperatorEquals,
						Value:    "success",
					},
				},
			},
		},
	}

	yamlBytes, _ := yaml.Marshal(tc)
	return string(yamlBytes)
}

func (c *Controller) onImportTestCase() {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Declined {
			return
		}

		if result.Error != nil {
			c.view.showError(fmt.Errorf("failed to get file: %w", result.Error))
			return
		}

		var tc domain.TestCase
		if err := yaml.Unmarshal(result.Data, &tc); err != nil {
			c.view.showError(fmt.Errorf("failed to parse test case YAML: %w", err))
			return
		}

		if err := c.repo.CreateTestCase(&tc); err != nil {
			c.view.showError(fmt.Errorf("failed to import test case: %w", err))
			return
		}

		if err := c.LoadData(); err != nil {
			c.view.showError(fmt.Errorf("failed to load test cases: %w", err))
			return
		}

		c.yamlContent[tc.MetaData.ID] = string(result.Data)
	}, "yaml", "yml")
}

func (c *Controller) onTestCaseChange(testCase *domain.TestCase, source state.Source, action state.Action) {
	if source == state.SourceController {
		return
	}

	switch action {
	case state.ActionAdd:
		c.view.AddTreeViewNode(testCase)
	case state.ActionUpdate:
		c.view.UpdateTreeViewNode(testCase)
	case state.ActionDelete:
		c.view.RemoveTreeViewNode(testCase.MetaData.ID)
		if c.activeTabID == testCase.MetaData.ID {
			c.activeTabID = ""
			c.view.CloseTab(testCase.MetaData.ID)
		}
		delete(c.yamlContent, testCase.MetaData.ID)
	}
}

func (c *Controller) onTitleChanged(id string, title string) {
	tc := c.state.GetTestCase(id)
	if tc == nil {
		return
	}

	tc.MetaData.Name = title
	
	// Also update YAML content
	if yamlStr, ok := c.yamlContent[id]; ok {
		var yamlData map[string]interface{}
		if err := yaml.Unmarshal([]byte(yamlStr), &yamlData); err == nil {
			if metadata, ok := yamlData["metadata"].(map[interface{}]interface{}); ok {
				metadata["name"] = title
				if newYAML, err := yaml.Marshal(yamlData); err == nil {
					c.yamlContent[id] = string(newYAML)
					
					// Mark as dirty
					c.view.SetTabDirty(id, true)
				}
			}
		}
	}

	c.view.SetContainerTitle(id, title)
	c.view.UpdateTreeViewNode(tc)
}

func (c *Controller) onTreeViewNodeClicked(id string) {
	c.openTestCase(id)
}

func (c *Controller) openTestCase(id string) {
	tc := c.state.GetTestCase(id)
	if tc == nil {
		return
	}

	// Check if tab is already open
	if c.view.IsTabOpen(id) {
		c.view.SwitchToTab(id)
		return
	}

	// Get or generate YAML content
	yamlStr, ok := c.yamlContent[id]
	if !ok {
		yamlBytes, err := yaml.Marshal(tc)
		if err != nil {
			c.view.showError(fmt.Errorf("failed to marshal test case to YAML: %w", err))
			return
		}
		yamlStr = string(yamlBytes)
		c.yamlContent[id] = yamlStr
	}

	c.view.OpenTab(tc, yamlStr)
	c.view.OpenContainer(tc, yamlStr)
	c.activeTabID = id
}

func (c *Controller) onTabSelected(id string) {
	c.activeTabID = id
}

func (c *Controller) onYAMLChanged(id string, yaml string) {
	c.yamlContent[id] = yaml
	
	// Compare with persisted version to set dirty state
	tc := c.state.GetTestCase(id)
	if tc == nil {
		return
	}

	tcFromFile, err := c.state.GetPersistedTestCase(id)
	if err != nil {
		// If we can't get from file, assume it's dirty
		c.view.SetTabDirty(id, true)
		return
	}

	// Marshal both to YAML and compare
	yamlFromFile, err := tcFromFile.MarshalYaml()
	if err != nil {
		c.view.SetTabDirty(id, true)
		return
	}

	// Compare YAML strings
	isDirty := yaml != string(yamlFromFile)
	c.view.SetTabDirty(id, isDirty)
}

func (c *Controller) onSave(id string) {
	c.saveTestCase(id)
}

func (c *Controller) saveTestCase(id string) {
	yamlStr, ok := c.yamlContent[id]
	if !ok {
		c.view.showError(fmt.Errorf("no YAML content found for test case"))
		return
	}

	// Parse YAML to validate
	var tc domain.TestCase
	if err := yaml.Unmarshal([]byte(yamlStr), &tc); err != nil {
		c.view.showError(fmt.Errorf("invalid YAML: %w", err))
		return
	}

	// Ensure ID is preserved
	if tc.MetaData.ID == "" {
		tc.MetaData.ID = id
	}

	if err := c.state.SaveTestCase(&tc, state.SourceController); err != nil {
		c.view.showError(fmt.Errorf("failed to save test case: %w", err))
		return
	}

	c.view.SetTabDirty(id, false)
	c.view.SetContainerTitle(id, tc.MetaData.Name)
	c.view.UpdateTreeNodeTitle(id, tc.MetaData.Name)
	c.view.UpdateTabTitle(id, tc.MetaData.Name)
}

func (c *Controller) onRun(id string) {
	tc := c.state.GetTestCase(id)
	if tc == nil {
		c.view.showError(fmt.Errorf("test case not found"))
		return
	}

	// Get YAML content to ensure we're running the latest version
	yamlStr, ok := c.yamlContent[id]
	if ok {
		var updatedTC domain.TestCase
		if err := yaml.Unmarshal([]byte(yamlStr), &updatedTC); err == nil {
			updatedTC.MetaData.ID = tc.MetaData.ID
			tc = &updatedTC
		}
	}

	// Clear previous results
	c.view.ClearTestResult(id)

	// Get active environment
	activeEnv := c.environments.GetActiveEnvironment()
	envID := ""
	if activeEnv != nil {
		envID = activeEnv.MetaData.ID
	}

	// Run test in background
	go func() {
		ctx := context.Background()
		result, err := c.testRunner.Run(ctx, tc, envID)
		if err != nil {
			c.view.showError(fmt.Errorf("test execution failed: %w", err))
			return
		}

		// Update UI with results
		c.view.UpdateTestResult(id, result)
	}()
}

func (c *Controller) onTabClose(id string) {
	tc := c.state.GetTestCase(id)
	if tc == nil {
		c.view.showError(fmt.Errorf("failed to get test case %s", id))
		return
	}

	tcFromFile, err := c.state.GetPersistedTestCase(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get test case from file: %w", err))
		return
	}

	// Compare YAML to check if data changed
	yamlStr, ok := c.yamlContent[id]
	if !ok {
		if c.activeTabID == id {
			c.activeTabID = ""
		}
		c.view.CloseTab(id)
		return
	}

	yamlFromFile, err := tcFromFile.MarshalYaml()
	if err != nil {
		c.view.showError(fmt.Errorf("failed to marshal test case: %w", err))
		return
	}

	// If data is not changed, close the tab
	if yamlStr == string(yamlFromFile) {
		if c.activeTabID == id {
			c.activeTabID = ""
		}
		c.view.CloseTab(id)
		return
	}

	// Show prompt to save changes
	c.view.ShowPrompt(id, "Save", "Do you want to save the changes? (Tips: you can always save the changes using CMD/CTRL+s)", widgets.ModalTypeWarn,
		func(selectedOption string, remember bool) {
			if selectedOption == "Cancel" {
				c.view.HidePrompt(id)
				return
			}

			if selectedOption == "Yes" {
				c.saveTestCase(id)
			}

			if c.activeTabID == id {
				c.activeTabID = ""
			}
			c.view.CloseTab(id)
			c.state.ReloadTestCase(id, state.SourceController)
		}, []widgets.Option{{Text: "Yes"}, {Text: "No"}, {Text: "Cancel"}}...,
	)
}

func (c *Controller) onTreeViewMenuClicked(id string, action string) {
	tc := c.state.GetTestCase(id)
	if tc == nil {
		return
	}

	switch action {
	case Run:
		c.onRun(id)
	case Duplicate:
		c.duplicateTestCase(tc)
	case Delete:
		c.deleteTestCase(tc)
	}
}

func (c *Controller) duplicateTestCase(tc *domain.TestCase) {
	clone := tc.Clone()
	clone.MetaData.Name = tc.MetaData.Name + " (Copy)"

	if err := c.repo.CreateTestCase(clone); err != nil {
		c.view.showError(fmt.Errorf("failed to duplicate test case: %w", err))
		return
	}

	clone, err := c.state.GetPersistedTestCase(clone.MetaData.ID)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get duplicated test case: %w", err))
		return
	}

	c.state.AddTestCase(clone, state.SourceController)
	c.view.AddTreeViewNode(clone)

	// Copy YAML content
	if yamlStr, ok := c.yamlContent[tc.MetaData.ID]; ok {
		c.yamlContent[clone.MetaData.ID] = yamlStr
	}
}

func (c *Controller) deleteTestCase(tc *domain.TestCase) {
	if err := c.state.RemoveTestCase(tc, state.SourceController, false); err != nil {
		c.view.showError(fmt.Errorf("failed to delete test case: %w", err))
		return
	}

	c.view.RemoveTreeViewNode(tc.MetaData.ID)
	if c.activeTabID == tc.MetaData.ID {
		c.view.CloseTab(tc.MetaData.ID)
		c.activeTabID = ""
	}

	delete(c.yamlContent, tc.MetaData.ID)
}

func (c *Controller) LoadData() error {
	testCases, err := c.state.LoadTestCases()
	if err != nil {
		return fmt.Errorf("failed to load test cases: %w", err)
	}

	c.view.PopulateTreeView(testCases)

	// Load YAML content for each test case
	for _, tc := range testCases {
		yamlBytes, err := yaml.Marshal(tc)
		if err == nil {
			c.yamlContent[tc.MetaData.ID] = string(yamlBytes)
		}
	}

	return nil
}
