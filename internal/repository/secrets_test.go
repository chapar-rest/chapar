package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keyStore is an in-memory stand-in for the OS secret manager.
type keyStore struct {
	value string
	set   bool
}

func (s *keyStore) Get() (string, error) {
	if !s.set {
		return "", secret.ErrNoKey
	}
	return s.value, nil
}

func (s *keyStore) Set(v string) error {
	s.value, s.set = v, true
	return nil
}

func (s *keyStore) Delete() error {
	s.value, s.set = "", false
	return nil
}

func (s *keyStore) Available() bool { return true }
func (s *keyStore) Name() string    { return "test store" }

func setupSecrets(t *testing.T, fs *FilesystemV2) (*secret.Manager, *keyStore) {
	t.Helper()
	store := &keyStore{}
	m := secret.New(store, filepath.Join(t.TempDir(), secret.MetaFileName))
	_, err := m.Generate(secret.ModeOS)
	require.NoError(t, err)
	fs.SetSecrets(m)
	return m, store
}

func envFileContents(t *testing.T, fs *FilesystemV2, name string) string {
	t.Helper()
	path, err := fs.EntityPath(domain.KindEnv)
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(path, name+".yaml"))
	require.NoError(t, err)
	return string(data)
}

func TestSecretValueIsEncryptedOnDisk(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()
	setupSecrets(t, fs)

	env := domain.NewEnvironment("Secrets")
	env.Spec.Values = []domain.KeyValue{
		{ID: "1", Key: "host", Value: "example.com", Enable: true},
		{ID: "2", Key: "token", Value: "swordfish", Enable: true, Secret: true},
	}
	require.NoError(t, fs.CreateEnvironment(env))

	raw := envFileContents(t, fs, "Secrets")
	assert.NotContains(t, raw, "swordfish", "a secret value must not be written in plain text")
	assert.Contains(t, raw, "example.com", "a plain value should stay readable")
	assert.Contains(t, raw, secret.Prefix, "the secret value should be marked as encrypted")

	// The in-memory environment keeps its plain value.
	assert.Equal(t, "swordfish", env.Spec.Values[1].Value)

	loaded, err := fs.LoadEnvironments()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	require.Len(t, loaded[0].Spec.Values, 2)
	assert.Equal(t, "swordfish", loaded[0].Spec.Values[1].Value, "a secret value should come back decrypted")
	assert.True(t, loaded[0].Spec.Values[1].Secret)
	assert.False(t, loaded[0].Spec.Values[1].Locked)
}

func TestLockedSecretSurvivesSaveWithoutKey(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()
	m, store := setupSecrets(t, fs)

	env := domain.NewEnvironment("Secrets")
	env.Spec.Values = []domain.KeyValue{
		{ID: "1", Key: "token", Value: "swordfish", Enable: true, Secret: true},
	}
	require.NoError(t, fs.CreateEnvironment(env))
	before := envFileContents(t, fs, "Secrets")

	// The key is gone: chapar can no longer read the value.
	require.NoError(t, m.Forget())
	_ = store

	loaded, err := fs.LoadEnvironments()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	locked := loaded[0]
	assert.True(t, locked.Spec.Values[0].Locked, "an unreadable value should be marked locked")
	assert.True(t, strings.HasPrefix(locked.Spec.Values[0].Value, secret.Prefix))

	// Saving the environment again must not destroy the value.
	locked.Spec.Values = append(locked.Spec.Values, domain.KeyValue{ID: "2", Key: "host", Value: "example.com", Enable: true})
	require.NoError(t, fs.UpdateEnvironment(locked))

	after := envFileContents(t, fs, "Secrets")
	assert.Contains(t, after, locked.Spec.Values[0].Value, "the ciphertext should be written back untouched")
	assert.NotContains(t, after, "swordfish")
	_ = before
}

func TestSavingSecretWithoutKeyFails(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()

	env := domain.NewEnvironment("Secrets")
	env.Spec.Values = []domain.KeyValue{
		{ID: "1", Key: "token", Value: "swordfish", Enable: true, Secret: true},
	}
	err := fs.CreateEnvironment(env)
	require.Error(t, err, "saving a secret without a key must fail rather than write plain text")
	assert.ErrorIs(t, err, secret.ErrNoKey)

	path, perr := fs.EntityPath(domain.KindEnv)
	require.NoError(t, perr)
	data, rerr := os.ReadFile(filepath.Join(path, "Secrets.yaml"))
	if rerr == nil {
		assert.NotContains(t, string(data), "swordfish")
	}
}

func TestLockedValueIsNotSubstituted(t *testing.T) {
	env := domain.NewEnvironment("Secrets")
	env.Spec.Values = []domain.KeyValue{
		{ID: "1", Key: "token", Value: secret.Prefix + "ZmFrZQ==", Enable: true, Secret: true, Locked: true},
		{ID: "2", Key: "host", Value: "example.com", Enable: true},
	}

	req := &domain.HTTPRequestSpec{URL: "https://{{host}}/x?t={{token}}", Request: &domain.HTTPRequest{}}
	env.ApplyToHTTPRequest(req)
	assert.Equal(t, "https://example.com/x?t={{token}}", req.URL, "a locked value must not leak ciphertext into a request")

	values := env.GetKeyValues()
	assert.NotContains(t, values, "token")
	assert.Equal(t, "example.com", values["host"])
}
