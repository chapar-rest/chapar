// Package secret encrypts environment values that the user marked as secret,
// so they never reach disk in plain text.
//
// One 32-byte master key encrypts every secret value with AES-256-GCM. The key
// lives in the OS secret manager (macOS Keychain, Windows Credential Manager,
// Linux Secret Service) or, where that is unavailable, is derived from a
// passphrase the user types once per session. Only non-secret metadata - key
// id, mode, salt and a verifier blob - is written to the config directory.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"crypto/pbkdf2"
)

const (
	// Prefix marks a value as ciphertext in the stored yaml.
	Prefix = "enc:v1:"

	// KeySize is the master key size in bytes (AES-256).
	KeySize = 32

	// canary is encrypted with a new key and kept in the metadata file, so a
	// key pasted on another machine can be checked before it is used.
	canary = "chapar-secret-verifier-v1"

	// pbkdf2Iter is the iteration count for passphrase-derived keys.
	pbkdf2Iter = 600_000
)

var (
	ErrNoKey       = errors.New("no secret key configured")
	ErrLocked      = errors.New("secret key is locked")
	ErrWrongKey    = errors.New("secret key does not match the one used for this data")
	ErrNotEncoded  = errors.New("value is not encrypted")
	ErrKeyExists   = errors.New("a secret key is already configured")
	ErrBadKey      = errors.New("invalid secret key: expected 32 bytes, base64 encoded")
	ErrPassphrase  = errors.New("wrong passphrase")
	ErrUnavailable = errors.New("no OS secret manager available on this system")
)

// Mode is where the master key comes from.
type Mode string

const (
	// ModeOS keeps the key in the OS secret manager.
	ModeOS Mode = "os"
	// ModePassphrase derives the key from a passphrase, kept in memory only
	// for the lifetime of the process.
	ModePassphrase Mode = "passphrase"
)

// Manager holds the master key for the running process and encrypts and
// decrypts individual values with it. The zero value is not usable; call New.
type Manager struct {
	mu    sync.RWMutex
	key   []byte
	meta  Meta
	store Store
	path  string
}

// New loads the key metadata and, in OS mode, unlocks the key from the OS
// secret manager. It returns a usable Manager even when nothing is configured
// yet or the key cannot be read; callers check Configured and Unlocked.
func New(store Store, metaPath string) *Manager {
	m := &Manager{store: store, path: metaPath}
	if meta, err := loadMeta(metaPath); err == nil {
		m.meta = meta
	}
	if m.meta.Spec.Mode == ModeOS {
		_ = m.unlockFromStore()
	}
	return m
}

// Configured reports whether a master key has been set up.
func (m *Manager) Configured() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.meta.Spec.KeyID != ""
}

// Unlocked reports whether the key is in memory and values can be read.
func (m *Manager) Unlocked() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.key) == KeySize
}

// Mode reports where the key is kept. It is empty until a key is configured.
func (m *Manager) Mode() Mode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.meta.Spec.Mode
}

// KeyID is a short public identifier for the configured key. It is derived
// from the key but reveals nothing about it.
func (m *Manager) KeyID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.meta.Spec.KeyID
}

// CreatedAt reports when the configured key was created.
func (m *Manager) CreatedAt() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.meta.Spec.CreatedAt
}

// StoreAvailable reports whether this machine has a usable OS secret manager.
func (m *Manager) StoreAvailable() bool { return m.store != nil && m.store.Available() }

// StoreName names the OS secret manager for use in dialogs ("Keychain", ...).
func (m *Manager) StoreName() string {
	if m.store == nil {
		return ""
	}
	return m.store.Name()
}

// Generate creates a new master key, stores it and returns it base64 encoded so
// it can be shown to the user once. It refuses to replace a configured key.
func (m *Manager) Generate(mode Mode) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.meta.Spec.KeyID != "" {
		return "", ErrKeyExists
	}
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("could not generate a key: %w", err)
	}
	if err := m.adoptLocked(key, mode, nil); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Import adopts a key the user pasted, for example one generated on another
// machine. When data was already encrypted with a different key, it fails with
// ErrWrongKey and the configured key is left alone.
func (m *Manager) Import(encoded string, mode Mode) error {
	key, err := decodeKey(encoded)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if v := m.meta.Spec.Verifier; v != "" {
		if err := verify(key, v); err != nil {
			return err
		}
	}
	return m.adoptLocked(key, mode, nil)
}

// SetPassphrase configures passphrase mode with a freshly derived key. Use it
// only when no key is configured yet; otherwise use UnlockWithPassphrase.
func (m *Manager) SetPassphrase(passphrase string) error {
	if passphrase == "" {
		return ErrPassphrase
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.meta.Spec.KeyID != "" {
		return ErrKeyExists
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("could not generate a salt: %w", err)
	}
	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return err
	}
	return m.adoptLocked(key, ModePassphrase, salt)
}

// UnlockWithPassphrase derives the configured key from a passphrase and keeps
// it in memory for this process.
func (m *Manager) UnlockWithPassphrase(passphrase string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.meta.Spec.Mode != ModePassphrase || m.meta.Spec.Salt == "" {
		return ErrNoKey
	}
	salt, err := base64.StdEncoding.DecodeString(m.meta.Spec.Salt)
	if err != nil {
		return fmt.Errorf("corrupt key metadata: %w", err)
	}
	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return err
	}
	if err := verify(key, m.meta.Spec.Verifier); err != nil {
		if errors.Is(err, ErrWrongKey) {
			return ErrPassphrase
		}
		return err
	}
	m.key = key
	return nil
}

// Reveal returns the master key base64 encoded, so the user can write it down
// or copy it to another machine.
func (m *Manager) Reveal() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.key) != KeySize {
		if m.meta.Spec.KeyID == "" {
			return "", ErrNoKey
		}
		return "", ErrLocked
	}
	return base64.StdEncoding.EncodeToString(m.key), nil
}

// Forget drops the key from memory and, in OS mode, from the OS secret
// manager. The metadata stays, so the same key can be imported again and
// already encrypted values remain readable once it is.
func (m *Manager) Forget() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.key = nil
	if m.meta.Spec.Mode == ModeOS && m.store != nil {
		if err := m.store.Delete(); err != nil && !errors.Is(err, ErrNoKey) {
			return err
		}
	}
	return nil
}

// Encrypt returns the ciphertext form of plain, bound to aad so a value cannot
// be moved to another environment or key by editing the file.
func (m *Manager) Encrypt(aad, plain string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gcm, err := m.gcmLocked()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("could not generate a nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), []byte(aad))
	return Prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. The aad must match the one used to encrypt.
func (m *Manager) Decrypt(aad, encoded string) (string, error) {
	if !IsEncrypted(encoded) {
		return "", ErrNotEncoded
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	gcm, err := m.gcmLocked()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, Prefix))
	if err != nil {
		return "", fmt.Errorf("corrupt encrypted value: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("corrupt encrypted value: too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, []byte(aad))
	if err != nil {
		return "", ErrWrongKey
	}
	return string(plain), nil
}

// IsEncrypted reports whether a stored value is ciphertext.
func IsEncrypted(value string) bool { return strings.HasPrefix(value, Prefix) }

// AAD binds a ciphertext to the environment and key it belongs to.
func AAD(envID, key string) string { return envID + "\x00" + key }

// adoptLocked installs key, persists the metadata and, in OS mode, writes the
// key to the OS secret manager. Callers hold m.mu.
func (m *Manager) adoptLocked(key []byte, mode Mode, salt []byte) error {
	verifier, err := sealCanary(key)
	if err != nil {
		return err
	}
	meta := m.meta
	meta.ApiVersion = metaAPIVersion
	meta.Kind = metaKind
	meta.Spec.KeyID = keyID(key)
	meta.Spec.Mode = mode
	meta.Spec.Verifier = verifier
	if meta.Spec.CreatedAt.IsZero() {
		meta.Spec.CreatedAt = time.Now().UTC()
	}
	if mode == ModePassphrase {
		if salt != nil {
			meta.Spec.Salt = base64.StdEncoding.EncodeToString(salt)
			meta.Spec.Iterations = pbkdf2Iter
		}
	} else {
		meta.Spec.Salt = ""
		meta.Spec.Iterations = 0
		if m.store == nil {
			return ErrUnavailable
		}
		if err := m.store.Set(base64.StdEncoding.EncodeToString(key)); err != nil {
			return err
		}
	}
	if err := saveMeta(m.path, meta); err != nil {
		return err
	}
	m.meta = meta
	m.key = key
	return nil
}

// unlockFromStore reads the key from the OS secret manager. Callers do not
// hold m.mu.
func (m *Manager) unlockFromStore() error {
	if m.store == nil {
		return ErrUnavailable
	}
	encoded, err := m.store.Get()
	if err != nil {
		return err
	}
	key, err := decodeKey(encoded)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if v := m.meta.Spec.Verifier; v != "" {
		if err := verify(key, v); err != nil {
			return err
		}
	}
	m.key = key
	return nil
}

// gcmLocked builds the AEAD for the in-memory key. Callers hold m.mu.
func (m *Manager) gcmLocked() (cipher.AEAD, error) {
	if len(m.key) != KeySize {
		if m.meta.Spec.KeyID == "" {
			return nil, ErrNoKey
		}
		return nil, ErrLocked
	}
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func decodeKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(key) != KeySize {
		return nil, ErrBadKey
	}
	return key, nil
}

func deriveKey(passphrase string, salt []byte) ([]byte, error) {
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, pbkdf2Iter, KeySize)
	if err != nil {
		return nil, fmt.Errorf("could not derive a key: %w", err)
	}
	return key, nil
}

// keyID is a public fingerprint of the key: the first bytes of its hash.
func keyID(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:6])
}

func sealCanary(key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(canary), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// verify checks a key against the stored canary.
func verify(key []byte, verifier string) error {
	raw, err := base64.StdEncoding.DecodeString(verifier)
	if err != nil {
		return fmt.Errorf("corrupt key metadata: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	if len(raw) < gcm.NonceSize() {
		return fmt.Errorf("corrupt key metadata: verifier too short")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil || string(plain) != canary {
		return ErrWrongKey
	}
	return nil
}
