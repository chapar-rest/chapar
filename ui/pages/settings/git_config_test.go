package settings

import (
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
)

func TestGitConfigurationIntegration(t *testing.T) {
	// Test that Git configuration is properly integrated
	config := prefs.GetGlobalConfig()

	// Verify Git configuration exists
	if config.Spec.Data.Git.Enabled != false {
		t.Errorf("Expected Git to be disabled by default, got %v", config.Spec.Data.Git.Enabled)
	}

	if config.Spec.Data.Git.Branch != "main" {
		t.Errorf("Expected default branch to be 'main', got %s", config.Spec.Data.Git.Branch)
	}

	// Test updating Git configuration
	config.Spec.Data.Git.Enabled = true
	config.Spec.Data.Git.RemoteURL = "https://github.com/test/repo.git"
	config.Spec.Data.Git.Username = "testuser"
	config.Spec.Data.Git.Token = "testtoken"
	config.Spec.Data.Git.Branch = "develop"

	// Verify the changes
	if !config.Spec.Data.Git.Enabled {
		t.Error("Expected Git to be enabled")
	}

	if config.Spec.Data.Git.RemoteURL != "https://github.com/test/repo.git" {
		t.Errorf("Expected remote URL to be 'https://github.com/test/repo.git', got %s", config.Spec.Data.Git.RemoteURL)
	}

	if config.Spec.Data.Git.Username != "testuser" {
		t.Errorf("Expected username to be 'testuser', got %s", config.Spec.Data.Git.Username)
	}

	if config.Spec.Data.Git.Token != "testtoken" {
		t.Errorf("Expected token to be 'testtoken', got %s", config.Spec.Data.Git.Token)
	}

	if config.Spec.Data.Git.Branch != "develop" {
		t.Errorf("Expected branch to be 'develop', got %s", config.Spec.Data.Git.Branch)
	}
}

func TestGitConfigurationChanged(t *testing.T) {
	// Test the Changed method
	config1 := domain.GitConfig{
		Enabled:   false,
		RemoteURL: "",
		Username:  "",
		Token:     "",
		Branch:    "main",
	}

	config2 := domain.GitConfig{
		Enabled:   true,
		RemoteURL: "https://github.com/test/repo.git",
		Username:  "testuser",
		Token:     "testtoken",
		Branch:    "main",
	}

	if !config2.Changed(config1) {
		t.Error("Expected config2 to be different from config1")
	}

	if config1.Changed(config1) {
		t.Error("Expected config1 to be the same as itself")
	}
}
