package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

func TestFilesystemV2_TestCases(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	tc := domain.NewTestCase("Smoke")
	tc.Spec.Steps = []domain.TestStep{{ID: "a", Request: domain.TestRequestRef{Ref: "Get"}}}
	if err := fs.CreateTestCase(tc); err != nil {
		t.Fatal(err)
	}
	dir, _ := fs.EntityPath(domain.KindTestCase)
	if _, err := os.Stat(filepath.Join(dir, "Smoke.yaml")); err != nil {
		t.Fatal(err)
	}

	// Importing the same file again gives it a new ID and a free name.
	again := *tc
	if err := fs.CreateTestCase(&again); err != nil {
		t.Fatal(err)
	}
	if again.ID() == tc.ID() || again.GetName() != "Smoke_1" {
		t.Fatalf("import = %s %s", again.ID(), again.GetName())
	}

	tc.SetName("Renamed")
	tc.Spec.Description = "changed"
	if err := fs.UpdateTestCase(tc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Smoke.yaml")); !os.IsNotExist(err) {
		t.Fatalf("old file still there: %v", err)
	}

	loaded, err := fs.LoadTestCases()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*domain.TestCase{}
	for _, l := range loaded {
		byName[l.GetName()] = l
	}
	if len(loaded) != 2 || byName["Renamed"].Spec.Description != "changed" || byName["Renamed"].ID() != tc.ID() {
		t.Fatalf("loaded %d: %+v", len(loaded), byName)
	}

	if err := fs.DeleteTestCase(tc); err != nil {
		t.Fatal(err)
	}
	if loaded, _ := fs.LoadTestCases(); len(loaded) != 1 {
		t.Fatalf("after delete: %d", len(loaded))
	}
}

func TestFilesystemV2_LegacyTestCase(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	dir, err := fs.EntityPath(domain.KindTestCase)
	if err != nil {
		t.Fatal(err)
	}
	// The format PR #164 wrote.
	legacy := `apiVersion: v1
kind: TestCase
metadata:
  id: 5f5b7aea
  name: New Test Case
spec:
  steps:
  - name: Example Step
    runNaked: false
    request:
      collection: MyCollection
      request: MyRequest
    assert:
      statusCode: 200
`
	if err := os.WriteFile(filepath.Join(dir, "old.yaml"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte("kind: TestCase\nspec: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = fs.LoadTestCases()
	files, ok := SkippedFiles(err)
	if !ok || len(files) != 2 {
		t.Fatalf("err = %v", err)
	}
	for _, f := range files {
		legacyFile := filepath.Base(f.Path) == "old.yaml"
		if got := errors.Is(f.Err, ErrLegacyTestCase); got != legacyFile {
			t.Errorf("%s: legacy = %v, err = %v", filepath.Base(f.Path), got, f.Err)
		}
	}
}
