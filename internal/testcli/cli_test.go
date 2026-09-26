package testcli

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/secret"
)

func TestMain(m *testing.M) {
	// prefs and the secret store read the user's config folder: point it
	// at a scratch one before anything loads it.
	home, err := os.MkdirTemp("", "chapar-testcli-home-*")
	if err != nil {
		panic(err)
	}
	for _, k := range []string{"HOME", "APPDATA"} {
		if err := os.Setenv(k, home); err != nil {
			panic(err)
		}
	}
	_ = os.Setenv("NO_COLOR", "1")
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}

// lockedToken looks encrypted to the repository, and no key can open it.
var lockedToken = secret.Prefix + "AAAA"

// newWorkspace writes a workspace with a Todos collection, a dev env and
// two test cases, and returns its folder.
func newWorkspace(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/todos":
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"data":{"id":"t1","token":"tok-1"}}`)
		case r.Method == "GET" && r.URL.Path == "/todos/t1":
			_, _ = fmt.Fprint(w, `{"data":{"id":"t1","done":false}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	data := t.TempDir()
	repo, err := repository.NewFilesystemV2(data, "ws")
	if err != nil {
		t.Fatal(err)
	}
	col := domain.NewCollection("Todos")
	must(t, repo.CreateCollection(col))
	create := domain.NewHTTPRequest("Create")
	create.MetaData.ID = "create"
	create.Spec.HTTP.Method, create.Spec.HTTP.URL = "POST", "{{base}}/todos"
	create.Spec.HTTP.Request.Variables = []domain.Variable{{
		TargetEnvVariable: "token", From: domain.VariableFromBody, JsonPath: "$.data.token", Enable: true,
	}}
	must(t, repo.CreateRequest(create, col))
	get := domain.NewHTTPRequest("Get")
	get.MetaData.ID = "get"
	get.Spec.HTTP.Method, get.Spec.HTTP.URL = "GET", "{{base}}/todos/{{todoId}}"
	must(t, repo.CreateRequest(get, col))

	env := domain.NewEnvironment("dev")
	env.MetaData.ID = "dev-id"
	env.Spec.Values = []domain.KeyValue{
		{Key: "base", Value: srv.URL, Enable: true},
		{Key: "apiKey", Value: lockedToken, Enable: true, Secret: true},
	}
	must(t, repo.CreateEnvironment(env))

	smoke := domain.NewTestCase("smoke")
	smoke.Spec.Tags = []string{"smoke"}
	createStep := domain.TestStep{
		ID:      "create",
		Request: domain.TestRequestRef{Ref: "Todos/Create"},
		Assert:  []domain.TestAssertion{{Target: "status", Op: "eq", Value: 201}},
		Capture: []domain.TestCapture{{Var: "todoId", From: "body", Path: "$.data.id"}},
	}
	smoke.Spec.Steps = []domain.TestStep{
		createStep,
		{
			ID:      "get",
			Request: domain.TestRequestRef{ID: "get"},
			Assert:  []domain.TestAssertion{{Target: "body", Path: "$.data.done", Op: "eq", Value: false}},
		},
	}
	must(t, repo.CreateTestCase(smoke))

	broken := domain.NewTestCase("broken")
	broken.Spec.Steps = []domain.TestStep{
		createStep,
		{
			ID:      "get",
			Request: domain.TestRequestRef{ID: "get"},
			Assert:  []domain.TestAssertion{{Target: "status", Op: "eq", Value: 500}},
		},
		{ID: "never", Request: domain.TestRequestRef{ID: "get"}},
	}
	must(t, repo.CreateTestCase(broken))

	return filepath.Join(data, "ws")
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = Main(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestPassingRunWritesReports(t *testing.T) {
	ws := newWorkspace(t)
	reports := t.TempDir()
	junit, js := filepath.Join(reports, "junit.xml"), filepath.Join(reports, "run.json")

	// Flags after the argument parse too.
	code, out, errOut := runCLI("--workspace", ws, "smoke", "--env", "dev",
		"--report", "junit="+junit, "--report", "json="+js)
	if code != ExitPassed {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	for _, want := range []string{"smoke · env dev", "✓ Todos/Create  201", "✓ get           200", "passed · 2 passed", "1 of 1 test cases passed"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(errOut, "secret values apiKey are locked") {
		t.Errorf("stderr lacks the locked-secret warning:\n%s", errOut)
	}

	var suites junitSuites
	b, err := os.ReadFile(junit)
	must(t, err)
	must(t, xml.Unmarshal(b, &suites))
	if suites.Tests != 2 || suites.Failures != 0 || suites.Suites[0].Cases[0].Classname != "smoke.steps" {
		t.Errorf("junit = %+v", suites)
	}
	var report struct{ Runs []map[string]any }
	b, err = os.ReadFile(js)
	must(t, err)
	must(t, json.Unmarshal(b, &report))
	if len(report.Runs) != 1 || report.Runs[0]["status"] != "passed" {
		t.Errorf("json = %s", b)
	}
}

func TestFailingRun(t *testing.T) {
	ws := newWorkspace(t)
	junit := filepath.Join(t.TempDir(), "junit.xml")
	code, out, _ := runCLI("--workspace", ws, "--env", "dev", "--report", "junit="+junit)
	if code != ExitFailed {
		t.Fatalf("exit %d\n%s", code, out)
	}
	for _, want := range []string{
		"✗ get           200",
		"status eq 500: got 200, want 500",
		"○ never         step get did not pass",
		"failed · 1 passed, 1 failed, 1 skipped",
		"1 of 2 test cases passed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q:\n%s", want, out)
		}
	}
	// broken runs before smoke: cases run sorted by name.
	if strings.Index(out, "broken") > strings.Index(out, "smoke") {
		t.Errorf("cases out of order:\n%s", out)
	}

	b, err := os.ReadFile(junit)
	must(t, err)
	var suites junitSuites
	must(t, xml.Unmarshal(b, &suites))
	if suites.Failures != 1 || suites.Skipped != 1 || !strings.Contains(string(b), "status eq 500") {
		t.Errorf("junit:\n%s", b)
	}
}

func TestBailAndTags(t *testing.T) {
	ws := newWorkspace(t)
	code, out, _ := runCLI("--workspace", ws, "--env", "dev", "--bail")
	if code != ExitFailed || strings.Contains(out, "smoke") {
		t.Fatalf("--bail ran on after a failure (exit %d):\n%s", code, out)
	}
	code, out, _ = runCLI("--workspace", ws, "--env", "dev", "--tag", "smoke,other")
	if code != ExitPassed || strings.Contains(out, "broken") {
		t.Fatalf("--tag picked the wrong cases (exit %d):\n%s", code, out)
	}
}

func TestCaseFile(t *testing.T) {
	ws := newWorkspace(t)
	file := filepath.Join(t.TempDir(), "extra.yaml")
	must(t, os.WriteFile(file, []byte(`apiVersion: v1
kind: TestCase
metadata:
  id: extra
spec:
  steps:
  - id: create
    request:
      ref: Todos/Create
    assert:
    - target: body
      path: $.data.id
      op: exists
`), 0o644))
	code, out, errOut := runCLI("--workspace", ws, "--env", "dev", file)
	if code != ExitPassed || !strings.Contains(out, "extra") {
		t.Fatalf("exit %d\n%s\n%s", code, out, errOut)
	}
}

func TestUsageErrors(t *testing.T) {
	ws := newWorkspace(t)
	bad := filepath.Join(t.TempDir(), "bad.yaml")
	must(t, os.WriteFile(bad, []byte(`apiVersion: v1
kind: TestCase
metadata: {name: bad}
spec:
  steps:
  - id: a
    request: {ref: Todos/Nope}
    assert:
    - {target: status, op: near, value: 1}
`), 0o644))

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown workspace", []string{"--workspace", "no-such-workspace"}, `workspace "no-such-workspace" not found`},
		{"unknown env", []string{"--workspace", ws, "--env", "prod"}, `environment "prod" not found; have: dev`},
		{"unknown case", []string{"--workspace", ws, "nope"}, `no test case named "nope"`},
		{"no tagged cases", []string{"--workspace", ws, "--tag", "none"}, "no test cases to run"},
		{"bad report", []string{"--workspace", ws, "--report", "html=x"}, "want junit=path or json=path"},
		{"invalid case", []string{"--workspace", ws, bad}, `bad: steps[0].request: request "Todos/Nope" not found`},
		{"bad flag", []string{"--nope"}, "flag provided but not defined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, errOut := runCLI(tt.args...)
			if code != ExitUsage || !strings.Contains(errOut, tt.want) {
				t.Fatalf("exit %d, stderr:\n%s\nwant %q", code, errOut, tt.want)
			}
		})
	}
	if _, errOut := runCLIErr(ws, bad); !strings.Contains(errOut, `unknown op "near"`) {
		t.Errorf("stderr lacks the second problem:\n%s", errOut)
	}
}

func runCLIErr(ws, file string) (int, string) {
	code, _, errOut := runCLI("--workspace", ws, file)
	return code, errOut
}

func TestPersistEnvKeepsLockedSecrets(t *testing.T) {
	ws := newWorkspace(t)
	repo, err := repository.NewFilesystemV2(filepath.Dir(ws), filepath.Base(ws))
	must(t, err)
	cases, err := repo.LoadTestCases()
	must(t, err)
	for _, tc := range cases {
		if tc.GetName() == "smoke" {
			tc.Spec.Options.PersistEnv = true
			must(t, repo.UpdateTestCase(tc))
		}
	}

	if code, out, errOut := runCLI("--workspace", ws, "--env", "dev", "smoke"); code != ExitPassed {
		t.Fatalf("exit %d\n%s\n%s", code, out, errOut)
	}
	b, err := os.ReadFile(filepath.Join(ws, "envs", "dev.yaml"))
	must(t, err)
	env := string(b)
	if !strings.Contains(env, "tok-1") {
		t.Errorf("the extract rule's value was not saved:\n%s", env)
	}
	if !strings.Contains(env, lockedToken) {
		t.Errorf("the locked secret was lost:\n%s", env)
	}
	if strings.Contains(env, "todoId") {
		t.Errorf("a capture was saved to the env:\n%s", env)
	}
}
