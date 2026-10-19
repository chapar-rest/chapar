package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/cookies"
)

// The values of secret environment variables are kept out of the environment
// file, which a workspace may share through git, and stored encrypted in the
// workspace's machine-local state instead: one file per environment, mapping
// a value's ID to its ciphertext.

// localSecretsDir is the directory of the local secret files of a workspace.
func localSecretsDir(workspaceDir string) string {
	return filepath.Join(workspaceDir, cookies.StateDir, "secrets")
}

func localSecretsFile(workspaceDir, envID string) string {
	return filepath.Join(localSecretsDir(workspaceDir), envID+".yaml")
}

// readLocalSecrets returns the stored ciphertexts of an environment. A missing
// file is no secrets.
func readLocalSecrets(workspaceDir, envID string) (map[string]string, error) {
	data, err := os.ReadFile(localSecretsFile(workspaceDir, envID))
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("cannot read secret values of environment %s: %w", envID, err)
	}
	return out, nil
}

// writeLocalSecrets replaces the stored ciphertexts of an environment. An
// empty map removes the file.
func writeLocalSecrets(workspaceDir, envID string, values map[string]string) error {
	file := localSecretsFile(workspaceDir, envID)
	if len(values) == 0 {
		return removeLocalSecrets(workspaceDir, envID)
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return err
	}
	if err := cookies.EnsureStateDir(workspaceDir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	// Write a temporary file and rename it over the old one, so a crash never
	// leaves a half-written file behind.
	tmp, err := os.CreateTemp(filepath.Dir(file), ".secrets-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}

func removeLocalSecrets(workspaceDir, envID string) error {
	err := os.Remove(localSecretsFile(workspaceDir, envID))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
