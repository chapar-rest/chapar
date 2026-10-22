package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chapar-rest/chapar/internal/domain"
)

// A workspace may be kept in git, so loading an item and saving it unchanged
// must leave its file byte for byte as it was.
func TestSavingUnchangedItemsIsStable(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	for _, r := range []*domain.Request{
		domain.NewHTTPRequest("http"),
		domain.NewGRPCRequest("grpc"),
		domain.NewGraphQLRequest("graphql"),
		domain.NewWebSocketRequest("ws"),
	} {
		require.NoError(t, fs.CreateRequest(r, nil))
	}
	col := domain.NewCollection("col")
	require.NoError(t, fs.CreateCollection(col))
	inCol := domain.NewHTTPRequest("inside")
	require.NoError(t, fs.CreateRequest(inCol, col))
	require.NoError(t, fs.CreateTestCase(domain.NewTestCase("case")))

	dir, err := fs.WorkspaceDir()
	require.NoError(t, err)

	// Loading fills in defaults the first save writes out, so compare the
	// second save with the first.
	saveAll(t, fs)
	before := snapshot(t, dir)
	saveAll(t, fs)
	assert.Equal(t, before, snapshot(t, dir))

	cols, err := fs.LoadCollections()
	require.NoError(t, err)
	require.Len(t, cols, 1)
	assert.Contains(t, before["collections/col/_collection.yaml"], "requests: []",
		"a collection's file should not hold copies of its requests")
	require.Len(t, cols[0].Spec.Requests, 1, "the requests still load from their own files")
}

// saveAll loads every request, collection and test case and saves it as it is.
func saveAll(t *testing.T, fs *FilesystemV2) {
	t.Helper()
	reqs, err := fs.LoadRequests()
	require.NoError(t, err)
	for _, r := range reqs {
		require.NoError(t, fs.UpdateRequest(r, nil))
	}
	cols, err := fs.LoadCollections()
	require.NoError(t, err)
	for _, c := range cols {
		require.NoError(t, fs.UpdateCollection(c))
		for _, r := range c.Spec.Requests {
			require.NoError(t, fs.UpdateRequest(r, c))
		}
	}
	cases, err := fs.LoadTestCases()
	require.NoError(t, err)
	for _, tc := range cases {
		require.NoError(t, fs.UpdateTestCase(tc))
	}
}

// snapshot reads every file under dir, keyed by its path relative to dir.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		out[rel] = string(data)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, out)
	return out
}
