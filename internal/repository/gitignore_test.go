package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureGitignoreCreatesFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, EnsureGitignore(dir))
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, ".state/\n.DS_Store\nThumbs.db\n", string(data))
}

func TestEnsureGitignoreKeepsUserLines(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".gitignore")
	require.NoError(t, os.WriteFile(file, []byte("*.log\n.DS_Store"), 0o644))

	require.NoError(t, EnsureGitignore(dir))
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, "*.log\n.DS_Store\n.state/\nThumbs.db\n", string(data))

	// A second run has nothing to add.
	require.NoError(t, EnsureGitignore(dir))
	again, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(again))
}
