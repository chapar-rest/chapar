package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chapar-rest/chapar/internal/domain"
)

func TestFilesystemV2_EntityPath(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Get the path for proto files
	path, err := fs.EntityPath(domain.KindProtoFile)
	assert.Nil(t, err, "expected no error getting proto file path")

	if path != filepath.Join(fs.dataDir, fs.workspaceName, "protofiles") {
		t.Errorf("expected proto file path '%s', got '%s'", filepath.Join(fs.dataDir, fs.workspaceName, "protofiles"), path)
	}

	// Get the path for collections
	path, err = fs.EntityPath(domain.KindCollection)
	assert.Nil(t, err, "expected no error getting collection path")
	if path != filepath.Join(fs.dataDir, fs.workspaceName, "collections") {
		t.Errorf("expected collection path '%s', got '%s'", filepath.Join(fs.dataDir, fs.workspaceName, "collections"), path)
	}

	// Get the path for environments
	path, err = fs.EntityPath(domain.KindEnv)
	assert.Nil(t, err, "expected no error getting environment path")
	if path != filepath.Join(fs.dataDir, fs.workspaceName, "environments") {
		t.Errorf("expected environment path '%s', got '%s'", filepath.Join(fs.dataDir, fs.workspaceName, "environments"), path)
	}

	// Get the path for requests
	path, err = fs.EntityPath(domain.KindRequest)
	assert.Nil(t, err, "expected no error getting request path")
	if path != filepath.Join(fs.dataDir, fs.workspaceName, "requests") {
		t.Errorf("expected request path '%s', got '%s'", filepath.Join(fs.dataDir, fs.workspaceName, "requests"), path)
	}

	// Get the path for workspaces
	path, err = fs.EntityPath(domain.KindWorkspace)
	assert.Nil(t, err, "expected no error getting workspace path")
	if path != fs.dataDir {
		t.Errorf("expected workspace path '%s', got '%s'", fs.dataDir, path)
	}
}

func TestFilesystemV2_LoadProtoFiles(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// create a temporary proto file for testing
	tempProtoFile := "test.yaml"
	dir, err := fs.EntityPath(domain.KindProtoFile)
	assert.Nil(t, err, "expected no error getting proto file path")
	tempFilePath := filepath.Join(dir, tempProtoFile)

	pf := domain.NewProtoFile("Test")
	assert.Nil(t, SaveToYaml(tempFilePath, pf), "expected no error saving proto file")

	protoFiles, err := fs.LoadProtoFiles()
	assert.Nil(t, err, "expected no error loading proto files")

	if len(protoFiles) == 0 {
		t.Error("expected at least one proto file, got none")
		return
	}

	if protoFiles[0].MetaData.Name != "Test" {
		t.Errorf("expected proto file name 'Test', got '%s'", protoFiles[0].MetaData.Name)
	}
}

// TestFilesystemV2_CreateProtoFile tests the creation of a proto file in the filesystem
func TestFilesystemV2_CreateProtoFile(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create a new proto file
	pf := domain.NewProtoFile("TestCreate")
	err := fs.CreateProtoFile(pf)
	assert.Nil(t, err, "expected no error creating proto file")

	// Verify the proto file was created
	protoFiles, err := fs.LoadProtoFiles()
	assert.Nil(t, err, "expected no error loading proto files")
	if len(protoFiles) == 0 {
		t.Error("expected at least one proto file after creation, got none")
		return
	}

	if protoFiles[0].MetaData.Name != "TestCreate" {
		t.Errorf("expected proto file name 'TestCreate', got '%s'", protoFiles[0].MetaData.Name)
	}

	// Create another proto file with the same name to test unique naming
	pfDuplicate := domain.NewProtoFile("TestCreate")
	err = fs.CreateProtoFile(pfDuplicate)
	assert.Nil(t, err, "expected no error creating duplicate proto file")

	// Verify the duplicate was created with a unique name
	protoFiles, err = fs.LoadProtoFiles()
	assert.Nil(t, err, "expected no error loading proto files after duplicate creation")

	if len(protoFiles) != 2 {
		t.Errorf("expected 2 proto files after creating duplicate, got %d", len(protoFiles))
		return
	}

	// we expect two files: "TestCreate.yaml" and "TestCreate_1.yaml"
	var names []string
	for _, pf := range protoFiles {
		names = append(names, pf.MetaData.Name)
	}

	assert.Subset(t, names, []string{"TestCreate", "TestCreate_1"}, "expected original proto file name")
}

func TestFilesystemV2_UpdateProtoFile(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create a new proto file
	pf := domain.NewProtoFile("TestUpdate")
	err := fs.CreateProtoFile(pf)
	assert.Nil(t, err, "expected no error creating proto file")

	// Update the proto file
	pf.Spec.Path = "updated/path/to/proto"

	err = fs.UpdateProtoFile(pf)
	assert.Nil(t, err, "expected no error updating proto file")

	// Load the proto files to verify the update
	protoFiles, err := fs.LoadProtoFiles()
	assert.Nil(t, err, "expected no error loading proto files after update")
	assert.Len(t, protoFiles, 1, "expected exactly one proto file after update")
	assert.Equal(t, pf.MetaData.Name, protoFiles[0].MetaData.Name, "expected proto file name")
}

func TestFilesystemV2_DeleteProtoFile(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create two new proto file
	pf := domain.NewProtoFile("TestDelete")
	err := fs.CreateProtoFile(pf)
	assert.Nil(t, err, "expected no error creating proto file")

	pf1 := domain.NewProtoFile("NotToDelete1")
	err = fs.CreateProtoFile(pf1)
	assert.Nil(t, err, "expected no error creating second proto file")

	// Delete the proto file
	err = fs.DeleteProtoFile(pf)
	assert.Nil(t, err, "expected no error deleting proto file")

	// Load the proto files to verify deletion
	protoFiles, err := fs.LoadProtoFiles()
	assert.Nil(t, err, "expected no error loading proto files after deletion")
	assert.Len(t, protoFiles, 1, "expected exactly one proto file after deletion")
	assert.Equal(t, pf1.MetaData.Name, protoFiles[0].MetaData.Name, "expected remaining proto file name to be 'NotToDelete1'")
}

func TestFilesystemV2_LoadRequests(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create a temporary request file for testing
	tempRequestFile := "test_request.yaml"
	dir, err := fs.EntityPath(domain.KindRequest)
	assert.Nil(t, err, "expected no error getting request path")
	tempFilePath := filepath.Join(dir, tempRequestFile)

	req := domain.NewHTTPRequest("TestRequest")
	assert.Nil(t, SaveToYaml(tempFilePath, req), "expected no error saving request")

	requests, err := fs.LoadRequests()
	assert.Nil(t, err, "expected no error loading requests")

	assert.Len(t, requests, 1, "expected exactly one request")
	assert.Equal(t, req.MetaData.Name, requests[0].MetaData.Name, "expected request name")
}

func TestFilesystemV2_CreateRequest(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create a new request
	req := domain.NewHTTPRequest("TestCreateRequest")
	err := fs.CreateRequest(req, nil)
	assert.Nil(t, err, "expected no error creating request")

	// Verify the request was created
	requests, err := fs.LoadRequests()
	assert.Nil(t, err, "expected no error loading requests")
	assert.Len(t, requests, 1, "expected exactly one request after creation")
	assert.Equal(t, req.MetaData.Name, requests[0].MetaData.Name, "expected request name to match")

	// Create another request with the same name to test unique naming
	reqDuplicate := domain.NewHTTPRequest("TestCreateRequest")
	err = fs.CreateRequest(reqDuplicate, nil)
	assert.Nil(t, err, "expected no error creating duplicate request")
	// Verify the duplicate was created with a unique name
	requests, err = fs.LoadRequests()
	assert.Nil(t, err, "expected no error loading requests after duplicate creation")
	assert.Len(t, requests, 2, "expected exactly two requests after creating duplicate")
	assert.Contains(t, []string{"TestCreateRequest", "TestCreateRequest_1"}, requests[1].MetaData.Name, "expected one of the request names to be 'TestCreateRequest' or 'TestCreateRequest_1'")
}

func TestFilesystemV2_UpdateRequest(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create a new request
	req := domain.NewHTTPRequest("TestUpdateRequest")
	err := fs.CreateRequest(req, nil)
	assert.Nil(t, err, "expected no error creating request")

	// Update the request
	req.Spec.HTTP.URL = "https://updated.url"
	err = fs.UpdateRequest(req, nil)
	assert.Nil(t, err, "expected no error updating request")

	// Load the requests to verify the update
	requests, err := fs.LoadRequests()
	assert.Nil(t, err, "expected no error loading requests after update")
	assert.Len(t, requests, 1, "expected exactly one request after update")
	assert.Equal(t, req.MetaData.Name, requests[0].MetaData.Name, "expected request name to match")
	assert.Equal(t, "https://updated.url", requests[0].Spec.HTTP.URL, "expected request URL to be updated")

	// Update the request name to test renaming the file
	req.MetaData.Name = "UpdatedRequestName"
	err = fs.UpdateRequest(req, nil)
	assert.Nil(t, err, "expected no error updating request name")

	// Load the requests to verify the renaming
	requests, err = fs.LoadRequests()
	assert.Nil(t, err, "expected no error loading requests after renaming")
	assert.Len(t, requests, 1, "expected exactly one request after renaming")
	assert.Equal(t, "UpdatedRequestName", requests[0].MetaData.Name, "expected request name to be 'UpdatedRequestName'")

	// Verify the file was renamed
	requestPath, err := fs.EntityPath(domain.KindRequest)
	assert.Nil(t, err, "expected no error getting request path")
	renamedFilePath := filepath.Join(requestPath, "UpdatedRequestName.yaml")
	_, err = os.Stat(renamedFilePath)
	assert.Nil(t, err, "expected no error checking renamed request file existence")

	// Check that the old file name does not exist
	oldFilePath := filepath.Join(requestPath, "TestUpdateRequest.yaml")
	_, err = os.Stat(oldFilePath)
	assert.True(t, os.IsNotExist(err), "expected old request file to not exist after renaming")
}

func TestFilesystemV2_DeleteRequest(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create two new requests
	req := domain.NewHTTPRequest("TestDeleteRequest")
	err := fs.CreateRequest(req, nil)
	assert.Nil(t, err, "expected no error creating request")

	req1 := domain.NewHTTPRequest("NotToDeleteRequest")
	err = fs.CreateRequest(req1, nil)
	assert.Nil(t, err, "expected no error creating second request")

	// Delete the request
	err = fs.DeleteRequest(req, nil)
	assert.Nil(t, err, "expected no error deleting request")
	// Load the requests to verify deletion
	requests, err := fs.LoadRequests()
	assert.Nil(t, err, "expected no error loading requests after deletion")
	assert.Len(t, requests, 1, "expected exactly one request after deletion")
	assert.Equal(t, req1.MetaData.Name, requests[0].MetaData.Name, "expected remaining request name to be 'NotToDeleteRequest'")
}

func TestFilesystemV2_LoadCollections(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create a temporary collection file for testing
	dir, err := fs.EntityPath(domain.KindCollection)
	assert.Nil(t, err, "expected no error getting collection path")
	collectionMetaPath := filepath.Join(dir, "TestCollection", "_collection.yaml")

	// make sure the directory exists
	err = os.MkdirAll(filepath.Dir(collectionMetaPath), 0755)
	assert.Nil(t, err, "expected no error creating collection directory")

	// create an empty folder in collections directory
	// this folder will be skipped by LoadCollections as it does not contain a "_collection.yaml" file
	noneCollectionDir := filepath.Join(dir, "NoneCollection")
	err = os.MkdirAll(noneCollectionDir, 0755)
	assert.Nil(t, err, "expected no error creating none collection directory")

	// create a random file in the collections directory
	// this file will not be loaded as a collection
	randomFilePath := filepath.Join(dir, "random.txt")
	err = os.WriteFile(randomFilePath, []byte("This is a random file"), 0644)

	col := domain.NewCollection("TestCollection")
	assert.Nil(t, SaveToYaml(collectionMetaPath, col), "expected no error saving collection")

	collections, err := fs.LoadCollections()
	assert.Nil(t, err, "expected no error loading collections")
	assert.Len(t, collections, 1, "expected exactly one collection")
	assert.Equal(t, col.MetaData.Name, collections[0].MetaData.Name, "expected collection name to match")
}

func TestFilesystemV2_CreateCollection(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create a new collection
	col := domain.NewCollection("TestCreateCollection")
	err := fs.CreateCollection(col)
	assert.Nil(t, err, "expected no error creating collection")

	// Verify the collection was created
	collections, err := fs.LoadCollections()
	assert.Nil(t, err, "expected no error loading collections")
	assert.Len(t, collections, 1, "expected exactly one collection after creation")
	assert.Equal(t, col.MetaData.Name, collections[0].MetaData.Name, "expected collection name to match")
}

func TestFilesystemV2_UpdateCollection(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	// Create a new collection
	col := domain.NewCollection("TestUpdateCollection")
	err := fs.CreateCollection(col)
	assert.Nil(t, err, "expected no error creating collection")

	// Update the collection
	col.MetaData.Name = "UpdatedCollectionName"
	err = fs.UpdateCollection(col)
	assert.Nil(t, err, "expected no error updating collection")

	// Load the collections to verify the update
	collections, err := fs.LoadCollections()
	assert.Nil(t, err, "expected no error loading collections after update")
	assert.Len(t, collections, 1, "expected exactly one collection after update")

	assert.Equal(t, "UpdatedCollectionName", collections[0].MetaData.Name, "expected collection name to be 'UpdatedCollectionName'")
	// Verify the collection file was renamed
	collectionPath, err := fs.EntityPath(domain.KindCollection)
	assert.Nil(t, err, "expected no error getting collection path")
	renamedDirPath := filepath.Join(collectionPath, "UpdatedCollectionName")
	_, err = os.Stat(renamedDirPath)
	assert.Nil(t, err, "expected no error checking renamed collection file existence")

	// Check that the old file name does not exist
	oldDirPath := filepath.Join(collectionPath, "TestUpdateCollection")
	_, err = os.Stat(oldDirPath)
	assert.True(t, os.IsNotExist(err), "expected old collection file to not exist after renaming")
}

// setupTest creates a temporary directory for testing and returns a cleanup function
func setupTest(t *testing.T) (*FilesystemV2, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "chapar-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	fs := NewFilesystemV2(tempDir, "Default")
	cleanup := func() {
		if t.Failed() {
			t.Logf("Test failed, keeping temp dir: %s", tempDir)
			return
		}

		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
			return
		}
	}

	return fs, cleanup
}
