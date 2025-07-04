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

	assert.Contains(t, names, "TestCreate", "expected original proto file name")
	assert.Contains(t, names, "TestCreate_1", "expected unique name for duplicate proto file")
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
