package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

// A malformed file must not stop the catalog from loading: the rest of the
// workspace loads and the file is reported once, however often it reloads.
func TestCatalogLoadSkipsMalformedFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	data := t.TempDir()
	repo, err := repository.NewFilesystemV2(data, "ws")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRequest(domain.NewHTTPRequest("Good"), nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateEnvironment(domain.NewEnvironment("Dev")); err != nil {
		t.Fatal(err)
	}
	dir, err := repo.EntityPath(domain.KindRequest)
	if err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "Bad.yaml")
	if err := os.WriteFile(bad, []byte("metadata:\n  id: [unclosed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cat := newCatalog(repo)
	if err := cat.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cat.Requests) != 1 || cat.Requests[0].MetaData.Name != "Good" {
		t.Fatalf("requests = %v, want only Good", cat.Requests)
	}
	if len(cat.Environments) != 1 {
		t.Fatalf("environments = %d, want 1", len(cat.Environments))
	}
	skipped := cat.DrainSkipped()
	if len(skipped) != 1 || skipped[0].Path != bad {
		t.Fatalf("skipped = %v, want %s", skipped, bad)
	}

	if err := cat.Load(); err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if again := cat.DrainSkipped(); len(again) != 0 {
		t.Fatalf("second load reported %v again", again)
	}
}
