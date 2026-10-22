package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/secret"
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

func localSecretsContents(t *testing.T, fs *FilesystemV2, envID string) string {
	t.Helper()
	dir, err := fs.WorkspaceDir()
	require.NoError(t, err)
	data, err := os.ReadFile(localSecretsFile(dir, envID))
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
	assert.NotContains(t, raw, secret.Prefix, "the shared file should not hold the ciphertext")
	assert.Contains(t, raw, "secret: true")

	local := localSecretsContents(t, fs, env.ID())
	assert.NotContains(t, local, "swordfish")
	assert.Contains(t, local, secret.Prefix, "the secret value should be stored encrypted on this machine")

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
	before := localSecretsContents(t, fs, env.ID())

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

	after := localSecretsContents(t, fs, env.ID())
	assert.Contains(t, after, locked.Spec.Values[0].Value, "the ciphertext should be written back untouched")
	assert.NotContains(t, after, "swordfish")
	assert.Equal(t, before, after)
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

func TestInlineCiphertextMovesOutOnSave(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()
	m, _ := setupSecrets(t, fs)

	env := domain.NewEnvironment("Legacy")
	enc, err := m.Encrypt(secret.AAD(env.ID(), "token"), "swordfish")
	require.NoError(t, err)
	env.Spec.Values = []domain.KeyValue{{ID: "1", Key: "token", Value: enc, Enable: true, Secret: true}}
	// Write the file the way older versions did, with the ciphertext inline.
	path, err := fs.EntityPath(domain.KindEnv)
	require.NoError(t, err)
	data, err := env.MarshalYaml()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(path, "Legacy.yaml"), data, 0o644))

	loaded, err := fs.LoadEnvironments()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, "swordfish", loaded[0].Spec.Values[0].Value, "an inline ciphertext should still load")

	require.NoError(t, fs.UpdateEnvironment(loaded[0]))
	raw := envFileContents(t, fs, "Legacy")
	assert.NotContains(t, raw, secret.Prefix, "saving should move the ciphertext out of the shared file")
	assert.Contains(t, localSecretsContents(t, fs, env.ID()), secret.Prefix)

	again, err := fs.LoadEnvironments()
	require.NoError(t, err)
	assert.Equal(t, "swordfish", again[0].Spec.Values[0].Value)
}

func TestSecretWithoutLocalValueIsUnset(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()
	setupSecrets(t, fs)

	env := domain.NewEnvironment("Shared")
	env.Spec.Values = []domain.KeyValue{
		{ID: "1", Key: "host", Value: "example.com", Enable: true},
		{ID: "2", Key: "token", Value: "swordfish", Enable: true, Secret: true},
	}
	require.NoError(t, fs.CreateEnvironment(env))
	shared := envFileContents(t, fs, "Shared")

	// A teammate clones the workspace: the shared file comes along, the
	// local state does not.
	dir, err := fs.WorkspaceDir()
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(localSecretsDir(dir)))

	loaded, err := fs.LoadEnvironments()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	token := loaded[0].Spec.Values[1]
	assert.True(t, token.Unset, "a secret with no local value should be unset")
	assert.True(t, token.Secret)
	assert.False(t, token.Locked)
	assert.Equal(t, "", token.Value)

	// It is not substituted into requests.
	req := &domain.HTTPRequestSpec{URL: "https://{{host}}/x?t={{token}}", Request: &domain.HTTPRequest{}}
	loaded[0].ApplyToHTTPRequest(req)
	assert.Equal(t, "https://example.com/x?t={{token}}", req.URL)

	// Saving without setting it leaves the shared file as it was.
	require.NoError(t, fs.UpdateEnvironment(loaded[0]))
	assert.Equal(t, shared, envFileContents(t, fs, "Shared"))

	// Typing the value in stores it on this machine only.
	loaded[0].Spec.Values[1].Value, loaded[0].Spec.Values[1].Unset = "hunter2", false
	require.NoError(t, fs.UpdateEnvironment(loaded[0]))
	assert.Equal(t, shared, envFileContents(t, fs, "Shared"))
	again, err := fs.LoadEnvironments()
	require.NoError(t, err)
	assert.Equal(t, "hunter2", again[0].Spec.Values[1].Value)
	assert.False(t, again[0].Spec.Values[1].Unset)
}

func TestSavingUnchangedEnvironmentIsStable(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()
	setupSecrets(t, fs)

	env := domain.NewEnvironment("Stable")
	env.Spec.Values = []domain.KeyValue{
		{ID: "1", Key: "host", Value: "example.com", Enable: true},
		{ID: "2", Key: "token", Value: "swordfish", Enable: true, Secret: true},
	}
	require.NoError(t, fs.CreateEnvironment(env))
	before := envFileContents(t, fs, "Stable")

	loaded, err := fs.LoadEnvironments()
	require.NoError(t, err)
	require.NoError(t, fs.UpdateEnvironment(loaded[0]))
	assert.Equal(t, before, envFileContents(t, fs, "Stable"), "saving an unchanged environment must not change its file")
}

func TestSecretWithoutIDGetsOne(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()
	setupSecrets(t, fs)

	env := domain.NewEnvironment("NoID")
	env.Spec.Values = []domain.KeyValue{{Key: "token", Value: "swordfish", Enable: true, Secret: true}}
	require.NoError(t, fs.CreateEnvironment(env))
	require.NotEmpty(t, env.Spec.Values[0].ID, "the stored value is found by ID")

	loaded, err := fs.LoadEnvironments()
	require.NoError(t, err)
	assert.Equal(t, "swordfish", loaded[0].Spec.Values[0].Value)
}

func TestDeletingEnvironmentRemovesSecretValues(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()
	setupSecrets(t, fs)

	env := domain.NewEnvironment("Gone")
	env.Spec.Values = []domain.KeyValue{{ID: "1", Key: "token", Value: "swordfish", Enable: true, Secret: true}}
	require.NoError(t, fs.CreateEnvironment(env))
	dir, err := fs.WorkspaceDir()
	require.NoError(t, err)
	require.FileExists(t, localSecretsFile(dir, env.ID()))

	require.NoError(t, fs.DeleteEnvironment(env))
	assert.NoFileExists(t, localSecretsFile(dir, env.ID()))
}

func TestDuplicatedEnvironmentKeepsSecretValues(t *testing.T) {
	fs, cleanup := setupTest(t)
	defer cleanup()
	setupSecrets(t, fs)

	env := domain.NewEnvironment("Orig")
	env.Spec.Values = []domain.KeyValue{{ID: "1", Key: "token", Value: "swordfish", Enable: true, Secret: true}}
	require.NoError(t, fs.CreateEnvironment(env))

	dup := env.Clone()
	require.NoError(t, fs.CreateEnvironment(dup))

	loaded, err := fs.LoadEnvironments()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	for _, e := range loaded {
		assert.Equal(t, "swordfish", e.Spec.Values[0].Value, "environment %s", e.GetName())
	}
}
