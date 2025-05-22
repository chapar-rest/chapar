package repository

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chapar-rest/chapar/internal/domain"
)

// setupTestFS creates a temporary directory for testing and returns a cleanup function
func setupTestFS(t *testing.T) (*Filesystem, func()) {
	tempDir, err := os.MkdirTemp("", "chapar-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	fs, err := NewFilesystem(tempDir, domain.AppStateSpec{})
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
			return
		}
	}

	return fs, cleanup
}

func TestNewFilesystem(t *testing.T) {
	fs, cleanup := setupTestFS(t)
	defer cleanup()

	assert.NotNil(t, fs)
	assert.NotNil(t, fs.ActiveWorkspace)
	assert.Equal(t, domain.DefaultWorkspaceName, fs.ActiveWorkspace.MetaData.Name)
}

func TestFilesystem_CreateAndLoadWorkspace(t *testing.T) {
	fs, cleanup := setupTestFS(t)
	defer cleanup()

	// Create a new workspace
	ws := domain.NewWorkspace("Test Workspace")
	err := fs.Create(ws)
	assert.NoError(t, err)

	// Load workspaces and verify
	workspaces, err := fs.LoadWorkspaces()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(workspaces), 1)

	var found bool
	for _, w := range workspaces {
		if w.MetaData.ID == ws.MetaData.ID {
			found = true
			assert.Equal(t, ws.MetaData.Name, w.MetaData.Name)
			break
		}
	}
	assert.True(t, found, "Created workspace not found in loaded workspaces")
}

func TestFilesystem_CreateAndLoadEnvironment(t *testing.T) {
	fs, cleanup := setupTestFS(t)
	defer cleanup()

	// Create a new environment
	env := domain.NewEnvironment("Test Environment")
	env.Spec.Values = []domain.KeyValue{
		{
			ID:     "1",
			Key:    "API_URL",
			Value:  "https://api.example.com",
			Enable: true,
		},
	}

	err := fs.Create(env)
	assert.NoError(t, err)

	// Load environments and verify
	environments, err := fs.LoadEnvironments()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(environments), 1)

	var found bool
	for _, e := range environments {
		if e.MetaData.ID == env.MetaData.ID {
			found = true
			assert.Equal(t, env.MetaData.Name, e.MetaData.Name)
			assert.Equal(t, len(env.Spec.Values), len(e.Spec.Values))
			assert.Equal(t, env.Spec.Values[0].Key, e.Spec.Values[0].Key)
			assert.Equal(t, env.Spec.Values[0].Value, e.Spec.Values[0].Value)
			break
		}
	}
	assert.True(t, found, "Created environment not found in loaded environments")
}

func TestFilesystem_CreateAndLoadRequest(t *testing.T) {
	fs, cleanup := setupTestFS(t)
	defer cleanup()

	// Create a new HTTP request
	req := domain.NewHTTPRequest("Test Request")
	req.Spec.HTTP.URL = "https://api.example.com/test"
	req.Spec.HTTP.Method = "GET"
	req.Spec.HTTP.Request.Headers = []domain.KeyValue{
		{
			ID:     "1",
			Key:    "Content-Type",
			Value:  "application/json",
			Enable: true,
		},
	}

	err := fs.Create(req)
	assert.NoError(t, err)

	// Load requests and verify
	requests, err := fs.LoadRequests()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(requests), 1)

	var found bool
	for _, r := range requests {
		if r.MetaData.ID == req.MetaData.ID {
			found = true
			assert.Equal(t, req.MetaData.Name, r.MetaData.Name)
			assert.Equal(t, req.Spec.HTTP.URL, r.Spec.HTTP.URL)
			assert.Equal(t, req.Spec.HTTP.Method, r.Spec.HTTP.Method)
			assert.Equal(t, len(req.Spec.HTTP.Request.Headers), len(r.Spec.HTTP.Request.Headers))
			break
		}
	}
	assert.True(t, found, "Created request not found in loaded requests")
}

func TestFilesystem_CreateAndLoadCollection(t *testing.T) {
	fs, cleanup := setupTestFS(t)
	defer cleanup()

	// Create a new collection
	collection := domain.NewCollection("Test Collection")
	err := fs.Create(collection)
	assert.NoError(t, err)

	// Add a request to the collection
	req := domain.NewHTTPRequest("Collection Request")
	req.Spec.HTTP.URL = "https://api.example.com/test"

	err = fs.CreateRequestInCollection(collection, req)
	assert.NoError(t, err)

	// Load collections and verify
	collections, err := fs.LoadCollections()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(collections), 1)

	var found bool
	for _, c := range collections {
		if c.MetaData.ID == collection.MetaData.ID {
			found = true
			assert.Equal(t, collection.MetaData.Name, c.MetaData.Name)
			assert.Equal(t, 1, len(c.Spec.Requests))
			if len(c.Spec.Requests) > 0 {
				assert.Equal(t, "Collection Request", c.Spec.Requests[0].MetaData.Name)
			}
			break
		}
	}
	assert.True(t, found, "Created collection not found in loaded collections")
}

func TestFilesystem_UpdateAndDeleteEntities(t *testing.T) {
	fs, cleanup := setupTestFS(t)
	defer cleanup()

	// Create and update a request
	t.Run("Update Request", func(t *testing.T) {
		req := domain.NewHTTPRequest("Test Request")
		err := fs.Create(req)
		assert.NoError(t, err)

		// Update request
		req.MetaData.Name = "Updated Request"
		err = fs.Update(req)
		assert.NoError(t, err)

		// Verify update
		loadedReq, err := fs.GetRequest(req.MetaData.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Request", loadedReq.MetaData.Name)

		// Delete request
		err = fs.Delete(req)
		assert.NoError(t, err)

		// Verify deletion
		requests, err := fs.LoadRequests()
		assert.NoError(t, err)
		for _, r := range requests {
			assert.NotEqual(t, req.MetaData.ID, r.MetaData.ID)
		}
	})

	// Create and update an environment
	t.Run("Update Environment", func(t *testing.T) {
		env := domain.NewEnvironment("Test Environment")
		err := fs.Create(env)
		assert.NoError(t, err)

		// Update environment
		env.MetaData.Name = "Updated Environment"
		err = fs.Update(env)
		assert.NoError(t, err)

		// Verify update
		loadedEnv, err := fs.GetEnvironment(env.MetaData.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Environment", loadedEnv.MetaData.Name)

		// Delete environment
		err = fs.Delete(env)
		assert.NoError(t, err)

		// Verify deletion
		environments, err := fs.LoadEnvironments()
		assert.NoError(t, err)
		for _, e := range environments {
			assert.NotEqual(t, env.MetaData.ID, e.MetaData.ID)
		}
	})
}
