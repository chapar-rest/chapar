package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chapar-rest/chapar/internal/domain"
)

const malformedYaml = "metadata:\n  id: [unclosed\n"

func TestFilesystemV2_LoadRequestsSkipsMalformedFile(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	dir, err := fs.EntityPath(domain.KindRequest)
	require.NoError(t, err)
	require.NoError(t, fs.CreateRequest(domain.NewHTTPRequest("Good"), nil))
	bad := filepath.Join(dir, "Bad.yaml")
	require.NoError(t, os.WriteFile(bad, []byte(malformedYaml), 0644))

	requests, err := fs.LoadRequests()
	files, ok := SkippedFiles(err)
	require.True(t, ok, "expected a skipped-files error, got %v", err)
	require.Len(t, files, 1)
	assert.Equal(t, bad, files[0].Path)
	require.Len(t, requests, 1)
	assert.Equal(t, "Good", requests[0].MetaData.Name)
}

func TestFilesystemV2_LoadCollectionsSkipsMalformedFiles(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	good := domain.NewCollection("Good")
	require.NoError(t, fs.CreateCollection(good))
	require.NoError(t, fs.CreateRequest(domain.NewHTTPRequest("Kept"), good))

	dir, err := fs.EntityPath(domain.KindCollection)
	require.NoError(t, err)
	badRequest := filepath.Join(dir, "Good", "Broken.yaml")
	require.NoError(t, os.WriteFile(badRequest, []byte(malformedYaml), 0644))

	badCollection := filepath.Join(dir, "Bad", "_collection.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(badCollection), 0755))
	require.NoError(t, os.WriteFile(badCollection, []byte(malformedYaml), 0644))

	collections, err := fs.LoadCollections()
	files, ok := SkippedFiles(err)
	require.True(t, ok, "expected a skipped-files error, got %v", err)
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	assert.ElementsMatch(t, []string{badRequest, badCollection}, paths)

	require.Len(t, collections, 1)
	assert.Equal(t, "Good", collections[0].MetaData.Name)
	require.Len(t, collections[0].Spec.Requests, 1)
	assert.Equal(t, "Kept", collections[0].Spec.Requests[0].MetaData.Name)
}

func TestFilesystemV2_LoadWorkspacesSkipsMalformedFile(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	require.NoError(t, fs.CreateWorkspace(domain.NewWorkspace("Good")))
	bad := filepath.Join(fs.dataDir, "Bad", "_workspace.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(bad), 0755))
	require.NoError(t, os.WriteFile(bad, []byte(malformedYaml), 0644))

	workspaces, err := fs.LoadWorkspaces()
	files, ok := SkippedFiles(err)
	require.True(t, ok, "expected a skipped-files error, got %v", err)
	require.Len(t, files, 1)
	assert.Equal(t, bad, files[0].Path)

	var names []string
	for _, w := range workspaces {
		names = append(names, w.MetaData.Name)
	}
	assert.Contains(t, names, "Good")
	assert.NotContains(t, names, "Bad")
}
