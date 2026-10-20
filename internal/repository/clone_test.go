package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

func TestAdoptCloneNamesFolderAfterWorkspace(t *testing.T) {
	data := t.TempDir()
	tmp := filepath.Join(data, ".clone-1")
	if err := os.Mkdir(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	ws := domain.NewWorkspace("Payments API")
	if err := SaveToYaml(filepath.Join(tmp, "_workspace.yaml"), ws); err != nil {
		t.Fatal(err)
	}
	got, err := AdoptClone(data, tmp, "payments", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID() != ws.ID() || got.GetName() != "Payments API" {
		t.Errorf("workspace = %+v", got.MetaData)
	}
	if _, err := os.Stat(filepath.Join(data, "Payments API", "_workspace.yaml")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("temp folder still there")
	}
	ignore, _ := os.ReadFile(filepath.Join(data, "Payments API", ".gitignore"))
	if !strings.Contains(string(ignore), ".state/") {
		t.Errorf(".gitignore = %q", ignore)
	}

	// The same repository again clashes by ID.
	tmp2 := filepath.Join(data, ".clone-2")
	_ = os.Mkdir(tmp2, 0o755)
	_ = SaveToYaml(filepath.Join(tmp2, "_workspace.yaml"), ws)
	if _, err := AdoptClone(data, tmp2, "x", []*domain.Workspace{got}); err == nil {
		t.Error("adopted a workspace with an ID already present")
	}
}

func TestAdoptCloneWithoutWorkspaceFile(t *testing.T) {
	data := t.TempDir()
	tmp := filepath.Join(data, ".clone-1")
	if err := os.Mkdir(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := AdoptClone(data, tmp, "api", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.GetName() != "api" || got.ID() == "" {
		t.Errorf("workspace = %+v", got.MetaData)
	}
	if _, err := os.Stat(filepath.Join(data, "api", "_workspace.yaml")); err != nil {
		t.Error(err)
	}
}

func TestCloneName(t *testing.T) {
	for url, want := range map[string]string{
		"https://github.com/team/payments-api.git": "payments-api",
		"git@github.com:team/api.git":              "api",
		"https://host/team/api/":                   "api",
		"/srv/repos/api.git":                       "api",
		"https://host/.git":                        "Cloned workspace",
	} {
		if got := CloneName(url); got != want {
			t.Errorf("CloneName(%q) = %q, want %q", url, got, want)
		}
	}
}

// A clone in progress lives in a hidden folder, which is not a workspace.
func TestLoadWorkspacesSkipsHiddenFolders(t *testing.T) {
	data := t.TempDir()
	fs, err := NewFilesystemV2(data, "Main")
	if err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(data, ".clone-1")
	_ = os.Mkdir(tmp, 0o755)
	_ = SaveToYaml(filepath.Join(tmp, "_workspace.yaml"), domain.NewWorkspace("Half cloned"))
	list, err := fs.LoadWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range list {
		if w.GetName() == "Half cloned" {
			t.Error("loaded a hidden folder as a workspace")
		}
	}
}
