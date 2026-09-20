package secret

import (
	"errors"
	"path/filepath"
	"testing"
)

// memStore is an in-memory stand-in for the OS secret manager.
type memStore struct {
	value     string
	set       bool
	available bool
}

func newMemStore() *memStore { return &memStore{available: true} }

func (s *memStore) Get() (string, error) {
	if !s.set {
		return "", ErrNoKey
	}
	return s.value, nil
}

func (s *memStore) Set(v string) error {
	s.value, s.set = v, true
	return nil
}

func (s *memStore) Delete() error {
	if !s.set {
		return ErrNoKey
	}
	s.value, s.set = "", false
	return nil
}

func (s *memStore) Available() bool { return s.available }
func (s *memStore) Name() string    { return "test store" }

func newManager(t *testing.T, store Store) *Manager {
	t.Helper()
	return New(store, filepath.Join(t.TempDir(), MetaFileName))
}

func TestGenerateEncryptDecrypt(t *testing.T) {
	m := newManager(t, newMemStore())
	if m.Configured() || m.Unlocked() {
		t.Fatal("a fresh manager should have no key")
	}

	encoded, err := m.Generate(ModeOS)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !m.Configured() || !m.Unlocked() {
		t.Fatal("manager should be configured and unlocked after Generate")
	}
	if len(encoded) == 0 || m.KeyID() == "" {
		t.Fatalf("Generate returned %q, key id %q", encoded, m.KeyID())
	}

	aad := AAD("env-1", "token")
	enc, err := m.Encrypt(aad, "swordfish")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !IsEncrypted(enc) {
		t.Fatalf("ciphertext is not marked: %q", enc)
	}
	if got, err := m.Decrypt(aad, enc); err != nil || got != "swordfish" {
		t.Fatalf("Decrypt: %q %v", got, err)
	}
}

func TestEncryptIsRandomized(t *testing.T) {
	m := newManager(t, newMemStore())
	if _, err := m.Generate(ModeOS); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	aad := AAD("env-1", "token")
	a, err := m.Encrypt(aad, "same")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	b, err := m.Encrypt(aad, "same")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if a == b {
		t.Fatal("the same plaintext encrypted twice must not produce the same ciphertext")
	}
}

func TestDecryptRejectsOtherAAD(t *testing.T) {
	m := newManager(t, newMemStore())
	if _, err := m.Generate(ModeOS); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	enc, err := m.Encrypt(AAD("env-1", "token"), "swordfish")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := m.Decrypt(AAD("env-2", "token"), enc); !errors.Is(err, ErrWrongKey) {
		t.Fatalf("a value moved to another environment should not decrypt: %v", err)
	}
	if _, err := m.Decrypt(AAD("env-1", "other"), enc); !errors.Is(err, ErrWrongKey) {
		t.Fatalf("a value moved to another key should not decrypt: %v", err)
	}
}

func TestLockedManagerCannotRead(t *testing.T) {
	store := newMemStore()
	path := filepath.Join(t.TempDir(), MetaFileName)

	m := New(store, path)
	if _, err := m.Generate(ModeOS); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	enc, err := m.Encrypt(AAD("env-1", "token"), "swordfish")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Same config, but the OS secret manager no longer has the key.
	_ = store.Delete()
	locked := New(store, path)
	if !locked.Configured() {
		t.Fatal("metadata should still say a key is configured")
	}
	if locked.Unlocked() {
		t.Fatal("manager should be locked without the stored key")
	}
	if _, err := locked.Decrypt(AAD("env-1", "token"), enc); !errors.Is(err, ErrLocked) {
		t.Fatalf("Decrypt while locked: %v", err)
	}
	if _, err := locked.Encrypt(AAD("env-1", "token"), "x"); !errors.Is(err, ErrLocked) {
		t.Fatalf("Encrypt while locked: %v", err)
	}
}

func TestImportRoundTripAcrossMachines(t *testing.T) {
	store := newMemStore()
	m := New(store, filepath.Join(t.TempDir(), MetaFileName))
	encoded, err := m.Generate(ModeOS)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	enc, err := m.Encrypt(AAD("env-1", "token"), "swordfish")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Another machine: no key, but the same environment file.
	other := New(newMemStore(), filepath.Join(t.TempDir(), MetaFileName))
	if err := other.Import(encoded, ModeOS); err != nil {
		t.Fatalf("Import: %v", err)
	}
	if got, err := other.Decrypt(AAD("env-1", "token"), enc); err != nil || got != "swordfish" {
		t.Fatalf("Decrypt after import: %q %v", got, err)
	}
}

func TestImportWrongKeyRejected(t *testing.T) {
	m := newManager(t, newMemStore())
	if _, err := m.Generate(ModeOS); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	keyID := m.KeyID()

	// A key generated elsewhere does not match this config's verifier.
	other := newManager(t, newMemStore())
	stranger, err := other.Generate(ModeOS)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	m2 := New(newMemStore(), m.path)
	if err := m2.Import(stranger, ModeOS); !errors.Is(err, ErrWrongKey) {
		t.Fatalf("Import of a stranger key: %v", err)
	}
	if m2.KeyID() != keyID {
		t.Fatalf("a rejected import must not replace the configured key: %q", m2.KeyID())
	}
}

func TestImportRejectsMalformedKey(t *testing.T) {
	m := newManager(t, newMemStore())
	for _, bad := range []string{"", "not base64!", "c2hvcnQ="} {
		if err := m.Import(bad, ModeOS); !errors.Is(err, ErrBadKey) {
			t.Fatalf("Import(%q): %v", bad, err)
		}
	}
}

func TestPassphraseMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), MetaFileName)
	m := New(nil, path)
	if err := m.SetPassphrase("correct horse battery staple"); err != nil {
		t.Fatalf("SetPassphrase: %v", err)
	}
	if m.Mode() != ModePassphrase {
		t.Fatalf("mode: %q", m.Mode())
	}
	enc, err := m.Encrypt(AAD("env-1", "token"), "swordfish")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Next run: metadata is there, the key is not.
	next := New(nil, path)
	if next.Unlocked() {
		t.Fatal("passphrase mode should start locked")
	}
	if err := next.UnlockWithPassphrase("wrong"); !errors.Is(err, ErrPassphrase) {
		t.Fatalf("wrong passphrase: %v", err)
	}
	if err := next.UnlockWithPassphrase("correct horse battery staple"); err != nil {
		t.Fatalf("UnlockWithPassphrase: %v", err)
	}
	if got, err := next.Decrypt(AAD("env-1", "token"), enc); err != nil || got != "swordfish" {
		t.Fatalf("Decrypt: %q %v", got, err)
	}
}

func TestGenerateRefusesToReplaceKey(t *testing.T) {
	m := newManager(t, newMemStore())
	if _, err := m.Generate(ModeOS); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := m.Generate(ModeOS); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("second Generate: %v", err)
	}
}

func TestForgetLocksButKeepsMetadata(t *testing.T) {
	store := newMemStore()
	m := New(store, filepath.Join(t.TempDir(), MetaFileName))
	encoded, err := m.Generate(ModeOS)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := m.Forget(); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if m.Unlocked() {
		t.Fatal("Forget should drop the key from memory")
	}
	if !m.Configured() {
		t.Fatal("Forget should keep the metadata so the key can come back")
	}
	if store.set {
		t.Fatal("Forget should remove the key from the OS secret manager")
	}
	if err := m.Import(encoded, ModeOS); err != nil {
		t.Fatalf("Import after Forget: %v", err)
	}
}

func TestRevealAndVerifierStates(t *testing.T) {
	m := newManager(t, newMemStore())
	if _, err := m.Reveal(); !errors.Is(err, ErrNoKey) {
		t.Fatalf("Reveal without a key: %v", err)
	}
	encoded, err := m.Generate(ModeOS)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got, err := m.Reveal(); err != nil || got != encoded {
		t.Fatalf("Reveal: %q %v", got, err)
	}
	if err := m.Forget(); err != nil {
		t.Fatalf("Forget: %v", err)
	}
	if _, err := m.Reveal(); !errors.Is(err, ErrLocked) {
		t.Fatalf("Reveal while locked: %v", err)
	}
}

func TestDecryptRejectsPlainValue(t *testing.T) {
	m := newManager(t, newMemStore())
	if _, err := m.Generate(ModeOS); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := m.Decrypt(AAD("env-1", "token"), "plain"); !errors.Is(err, ErrNotEncoded) {
		t.Fatalf("Decrypt of a plain value: %v", err)
	}
}
