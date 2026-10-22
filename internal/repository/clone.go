package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chapar-rest/chapar/internal/domain"
)

// AdoptClone turns a freshly cloned repository in tmpDir, a folder inside
// dataDir, into a workspace. A workspace's folder is named after it, so
// the clone moves to dataDir/<name>, where name comes from the clone's
// _workspace.yaml. A repository without one becomes a workspace named
// fallback, and gets the file. existing are the workspaces dataDir holds.
func AdoptClone(dataDir, tmpDir, fallback string, existing []*domain.Workspace) (*domain.Workspace, error) {
	file := filepath.Join(tmpDir, "_workspace.yaml")
	ws, err := LoadFromYaml[domain.Workspace](file)
	switch {
	case errors.Is(err, os.ErrNotExist):
		ws = domain.NewWorkspace(fallback)
		if err := SaveToYaml(file, ws); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, fmt.Errorf("the repository's _workspace.yaml could not be read: %w", err)
	}
	name := ws.GetName()
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return nil, fmt.Errorf("the repository's workspace name %q can't be a folder name", name)
	}
	for _, w := range existing {
		if w.ID() == ws.ID() {
			return nil, fmt.Errorf("this repository holds the workspace %q, which is already here as %q",
				name, w.GetName())
		}
	}
	dest := filepath.Join(dataDir, name)
	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("a workspace named %q already exists; rename it first", name)
	}
	if err := os.Rename(tmpDir, dest); err != nil {
		return nil, err
	}
	if err := EnsureGitignore(dest); err != nil {
		return nil, err
	}
	return ws, nil
}

// CloneName suggests a workspace name for a repository URL: its last path
// element without .git.
func CloneName(url string) string {
	url = strings.TrimRight(url, "/")
	if i := strings.LastIndexAny(url, "/:"); i >= 0 {
		url = url[i+1:]
	}
	url = strings.TrimSuffix(url, ".git")
	if url == "" || strings.HasPrefix(url, ".") {
		return "Cloned workspace"
	}
	return url
}
