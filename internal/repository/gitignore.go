package repository

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chapar-rest/chapar/internal/cookies"
)

// workspaceIgnores are the paths a workspace kept in git never tracks: the
// machine-local state (cookies, secret values) and files the OS drops.
var workspaceIgnores = []string{
	cookies.StateDir + "/",
	".DS_Store",
	"Thumbs.db",
}

// EnsureGitignore makes the .gitignore at the root of a workspace list the
// paths it must never track. Lines the user wrote are kept; missing ones are
// added at the end.
func EnsureGitignore(workspaceDir string) error {
	file := filepath.Join(workspaceDir, ".gitignore")
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	have := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		have[strings.TrimSpace(line)] = true
	}

	var missing []string
	for _, p := range workspaceIgnores {
		if !have[p] {
			missing = append(missing, p)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	out := string(data)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += strings.Join(missing, "\n") + "\n"
	return os.WriteFile(file, []byte(out), 0o644)
}
