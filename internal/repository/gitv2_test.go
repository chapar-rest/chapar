package repository

import (
	"fmt"
	"os"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
)

func TestGitRepositoryV2_BasicOperations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "chapar-git-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize Git repository
	gitConfig := &GitConfig{
		RemoteURL: "", // No remote for testing
		Username:  "test-user",
		Token:     "",
		Branch:    "main",
	}

	repo, err := NewGitRepositoryV2(tempDir, "test-workspace", gitConfig)
	if err != nil {
		t.Fatalf("Failed to create Git repository: %v", err)
	}

	// Test workspace creation
	workspace := domain.NewWorkspace("Test Workspace")
	err = repo.CreateWorkspace(workspace)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Test environment creation
	env := domain.NewEnvironment("Test Environment")
	env.SetKey("API_URL", "https://api.example.com")
	err = repo.CreateEnvironment(env)
	if err != nil {
		t.Fatalf("Failed to create environment: %v", err)
	}

	// Test proto file creation
	protoFile := domain.NewProtoFile("test.proto")
	protoFile.Spec.Path = "/path/to/test.proto"
	err = repo.CreateProtoFile(protoFile)
	if err != nil {
		t.Fatalf("Failed to create proto file: %v", err)
	}

	// Test loading entities
	workspaces, err := repo.LoadWorkspaces()
	if err != nil {
		t.Fatalf("Failed to load workspaces: %v", err)
	}
	if len(workspaces) != 1 {
		t.Errorf("Expected 1 workspace, got %d", len(workspaces))
	}

	environments, err := repo.LoadEnvironments()
	if err != nil {
		t.Fatalf("Failed to load environments: %v", err)
	}
	if len(environments) != 1 {
		t.Errorf("Expected 1 environment, got %d", len(environments))
	}

	protoFiles, err := repo.LoadProtoFiles()
	if err != nil {
		t.Fatalf("Failed to load proto files: %v", err)
	}
	if len(protoFiles) != 1 {
		t.Errorf("Expected 1 proto file, got %d", len(protoFiles))
	}

	// Test commit history
	commits, err := repo.GetCommitHistory()
	if err != nil {
		t.Fatalf("Failed to get commit history: %v", err)
	}
	if len(commits) == 0 {
		t.Error("Expected at least one commit")
	}

	fmt.Printf("Successfully created %d commits\n", len(commits))
	for i, commit := range commits {
		fmt.Printf("Commit %d: %s\n", i+1, commit)
	}
}

func TestGitRepositoryV2_WithRemote(t *testing.T) {
	// This test demonstrates how to use Git repository with remote
	// Note: This test requires actual Git credentials and will be skipped in CI
	
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping remote Git test in CI")
	}

	tempDir, err := os.MkdirTemp("", "chapar-git-remote-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Example configuration for remote Git repository
	gitConfig := &GitConfig{
		RemoteURL: "https://github.com/username/repository.git",
		Username:  "username",
		Token:     "your-github-token",
		Branch:    "main",
	}

	repo, err := NewGitRepositoryV2(tempDir, "test-workspace", gitConfig)
	if err != nil {
		t.Fatalf("Failed to create Git repository: %v", err)
	}

	// Create a test workspace
	workspace := domain.NewWorkspace("Remote Test Workspace")
	err = repo.CreateWorkspace(workspace)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Test push (this will fail without valid credentials, but demonstrates usage)
	err = repo.PushChanges()
	if err != nil {
		fmt.Printf("Push failed (expected without valid credentials): %v\n", err)
	}

	fmt.Println("Git repository with remote configuration created successfully")
}
