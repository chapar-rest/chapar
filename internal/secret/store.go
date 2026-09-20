package secret

import (
	"errors"
	"runtime"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "chapar"
	keyringAccount = "master-key"
)

// Store is where the master key is kept between runs.
type Store interface {
	// Get returns the stored key, or ErrNoKey when nothing is stored.
	Get() (string, error)
	Set(encodedKey string) error
	Delete() error
	// Available reports whether the backing service can be reached.
	Available() bool
	// Name is how the service is called on this platform, for dialogs.
	Name() string
}

// OSStore keeps the key in the platform secret manager: Keychain on macOS,
// Credential Manager on Windows, Secret Service on Linux.
type OSStore struct{}

func NewOSStore() *OSStore { return &OSStore{} }

func (s *OSStore) Get() (string, error) {
	v, err := keyring.Get(keyringService, keyringAccount)
	switch {
	case errors.Is(err, keyring.ErrNotFound):
		return "", ErrNoKey
	case err != nil:
		return "", err
	}
	return v, nil
}

func (s *OSStore) Set(encodedKey string) error {
	return keyring.Set(keyringService, keyringAccount, encodedKey)
}

func (s *OSStore) Delete() error {
	err := keyring.Delete(keyringService, keyringAccount)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNoKey
	}
	return err
}

// Available probes the secret manager with a read. A missing entry still means
// the service works; only a transport error (no Secret Service on a headless
// Linux box, say) counts as unavailable.
func (s *OSStore) Available() bool {
	_, err := keyring.Get(keyringService, keyringAccount)
	return err == nil || errors.Is(err, keyring.ErrNotFound)
}

func (s *OSStore) Name() string {
	switch runtime.GOOS {
	case "darwin":
		return "Keychain"
	case "windows":
		return "Credential Manager"
	default:
		return "system keyring"
	}
}
