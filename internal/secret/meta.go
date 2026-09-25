package secret

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	metaAPIVersion = "v1"
	metaKind       = "SecretKey"

	// MetaFileName is the file, in the config directory, that holds the
	// non-secret half of the key setup.
	MetaFileName = "secret-key.yaml"
)

// Meta is what chapar knows about the master key without holding it: an id, how
// it is stored, and a verifier that tells whether a given key is the right one.
type Meta struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Spec       MetaSpec `yaml:"spec"`
}

type MetaSpec struct {
	KeyID     string    `yaml:"keyID"`
	Mode      Mode      `yaml:"mode"`
	CreatedAt time.Time `yaml:"createdAt"`
	// Verifier is a known string encrypted with the key, used to check a key
	// before it is trusted with real data.
	Verifier string `yaml:"verifier"`
	// Salt and Iterations are set in passphrase mode only.
	Salt       string `yaml:"salt,omitempty"`
	Iterations int    `yaml:"iterations,omitempty"`
}

func loadMeta(path string) (Meta, error) {
	var meta Meta
	data, err := os.ReadFile(path)
	if err != nil {
		return meta, err
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return Meta{}, fmt.Errorf("could not read %s: %w", path, err)
	}
	return meta, nil
}

func saveMeta(path string, meta Meta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o600)
}
