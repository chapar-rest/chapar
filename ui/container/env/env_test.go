package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/secret"
	"github.com/chapar-rest/chapar/ui/container"
)

type memStore struct{ v string }

func (s *memStore) Get() (string, error) {
	if s.v == "" {
		return "", secret.ErrNoKey
	}
	return s.v, nil
}
func (s *memStore) Set(v string) error { s.v = v; return nil }
func (s *memStore) Delete() error      { s.v = ""; return nil }
func (s *memStore) Available() bool    { return true }
func (s *memStore) Name() string       { return "memory" }

// A secret with no value on this machine stays unset when the environment is
// saved, until the user types one in.
func TestUnsetSecretSurvivesSave(t *testing.T) {
	dir := t.TempDir()
	repo, err := repository.NewFilesystemV2(dir, "ws")
	if err != nil {
		t.Fatal(err)
	}
	m := secret.New(&memStore{}, filepath.Join(t.TempDir(), secret.MetaFileName))
	if _, err := m.Generate(secret.ModeOS); err != nil {
		t.Fatal(err)
	}
	repo.SetSecrets(m)

	// The shared file, as a clone brings it: the secret is named, its value
	// left out.
	envDir := filepath.Join(dir, "ws", "envs")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shared := "apiVersion: v1\nkind: Environment\nmetadata:\n  id: dev-id\n  name: dev\nspec:\n  values:\n  - id: \"1\"\n    key: token\n    value: \"\"\n    enable: true\n    secret: true\n"
	if err := os.WriteFile(filepath.Join(envDir, "dev.yaml"), []byte(shared), 0o644); err != nil {
		t.Fatal(err)
	}
	envs, err := repo.LoadEnvironments()
	if err != nil || len(envs) != 1 {
		t.Fatalf("load: %v %v", envs, err)
	}

	c := Open(envs[0], container.Deps{Repo: repo, Secrets: m})
	if !c.unset["1"] {
		t.Fatal("the row should be marked unset")
	}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if !c.unset["1"] {
		t.Fatal("saving without a value should keep the row unset")
	}

	c.table.SetCell("1", "value", "swordfish")
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if c.unset["1"] {
		t.Fatal("typing a value should clear the unset mark")
	}
	got, err := os.ReadFile(filepath.Join(envDir, "dev.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != shared {
		t.Fatalf("the shared file changed:\n%s", got)
	}
	local, err := os.ReadFile(filepath.Join(dir, "ws", ".state", "secrets", "dev-id.yaml"))
	if err != nil || !strings.Contains(string(local), secret.Prefix) {
		t.Fatalf("the value was not stored on this machine: %q %v", local, err)
	}
}

// Saving a secret on a machine with no key asks for one rather than fail; if
// the user backs out, nothing is written.
func TestSaveWithoutKeyWaitsForOne(t *testing.T) {
	dir := t.TempDir()
	repo, err := repository.NewFilesystemV2(dir, "ws")
	if err != nil {
		t.Fatal(err)
	}
	m := secret.New(&memStore{}, filepath.Join(t.TempDir(), secret.MetaFileName))
	repo.SetSecrets(m)

	envDir := filepath.Join(dir, "ws", "envs")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shared := "apiVersion: v1\nkind: Environment\nmetadata:\n  id: dev-id\n  name: dev\nspec:\n  values:\n  - id: \"1\"\n    key: token\n    value: \"\"\n    enable: true\n    secret: true\n"
	if err := os.WriteFile(filepath.Join(envDir, "dev.yaml"), []byte(shared), 0o644); err != nil {
		t.Fatal(err)
	}
	envs, err := repo.LoadEnvironments()
	if err != nil || len(envs) != 1 {
		t.Fatalf("load: %v %v", envs, err)
	}

	var shown []error
	c := Open(envs[0], container.Deps{Repo: repo, Secrets: m})
	c.deps.Report.Error = func(err error) { shown = append(shown, err) }
	c.table.SetCell("1", "value", "swordfish")
	if err := c.Save(); err != nil {
		t.Fatalf("save should wait for a key, not fail: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ws", ".state", "secrets", "dev-id.yaml")); !os.IsNotExist(err) {
		t.Fatalf("nothing should be stored without a key: %v", err)
	}
	if !c.unset["1"] {
		t.Fatal("the row should still be unset")
	}
}
