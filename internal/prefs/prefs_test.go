package prefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

// A config saved by an older build is moved to the current script runner
// image, and the change is written back.
func TestLoadUpgradesExecutorImage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "global-config.yaml")
	old := "spec:\n  scripting:\n    enabled: true\n    useDocker: true\n    dockerImage: chapar/python-executor:0.3.0\n"
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{}
	if _, err := m.loadGlobalConfig(); err != nil {
		t.Fatal(err)
	}
	if got := m.globalConfig.Spec.Scripting.DockerImage; got != domain.PythonExecutorImage {
		t.Fatalf("loaded image %q, want %q", got, domain.PythonExecutorImage)
	}
	if !m.globalConfig.Spec.Scripting.Enabled {
		t.Fatal("the rest of the scripting config was lost")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), domain.PythonExecutorImage) {
		t.Fatalf("saved config does not name %s:\n%s", domain.PythonExecutorImage, data)
	}
}
