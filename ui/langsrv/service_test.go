package langsrv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

// The settings page asks every server for its status on each frame it builds,
// so the PATH lookup behind it is cached; PathReady must drop that cache, since
// it means PATH itself changed.
func TestLookPathCachesUntilPathChanges(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	s := newTestService()
	cfg := domain.LanguageServerConfig{Enabled: true, Command: "fake-langserver"}

	if !s.Missing(cfg) {
		t.Fatal("server reported present before it was installed")
	}

	bin := filepath.Join(dir, "fake-langserver")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if !s.Missing(cfg) {
		t.Fatal("lookup was not cached: PATH was searched again")
	}

	s.PathReady()
	if s.Missing(cfg) {
		t.Fatal("PathReady did not drop the cached lookup")
	}
	if got := s.Status(cfg); got != "Ready · "+bin+" (starts when an editor needs it)" {
		t.Fatalf("Status = %q", got)
	}
}
