package repository

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/secret"
)

// Environment values marked secret are encrypted here, at the edge of the
// filesystem, so the rest of the app only ever sees plain values. The
// environment file names a secret but leaves its value empty, since the file
// may be shared through git; the ciphertext is kept in the workspace's
// machine-local state (see localsecrets.go). A value that cannot be decrypted -
// no key, or the key is locked - stays as ciphertext and is marked Locked, and
// is written back exactly as it was read. A secret with no stored value, as in
// a freshly cloned workspace, is marked Unset.
//
// Older files hold the ciphertext inline. They load as before, and the next
// save moves the value out of the file.

// openEnvironment decrypts the secret values of a freshly loaded environment.
func (f *FilesystemV2) openEnvironment(env *domain.Environment) {
	if env == nil {
		return
	}
	var stored map[string]string
	for i, kv := range env.Spec.Values {
		v := &env.Spec.Values[i]
		v.Locked, v.Unset = false, false
		cipher := kv.Value
		if !secret.IsEncrypted(cipher) {
			// A plain value in the file is used as it is, even when marked
			// secret: someone wrote it there on purpose. Saving encrypts it.
			if !kv.Secret || kv.Value != "" {
				continue
			}
			if stored == nil {
				stored = f.readLocalSecrets(env.MetaData.ID)
			}
			c, ok := stored[kv.ID]
			if !ok {
				v.Unset = true
				continue
			}
			cipher = c
		}
		v.Secret = true
		v.Value = cipher
		if f.secrets == nil || !f.secrets.Unlocked() {
			v.Locked = true
			continue
		}
		plain, err := f.secrets.Decrypt(secret.AAD(env.MetaData.ID, kv.Key), cipher)
		if err != nil {
			v.Locked = true
			continue
		}
		v.Value = plain
	}
}

// readLocalSecrets is readLocalSecrets for the active workspace. A file that
// cannot be read leaves the values unset; saving then fails rather than
// overwrite it.
func (f *FilesystemV2) readLocalSecrets(envID string) map[string]string {
	dir, err := f.WorkspaceDir()
	if err != nil {
		return map[string]string{}
	}
	stored, err := readLocalSecrets(dir, envID)
	if err != nil {
		return map[string]string{}
	}
	return stored
}

// sealEnvironment stores the secret values of env, encrypted, in the local
// state, and returns a copy of env with those values left empty, ready to be
// written. The original is left untouched so the UI keeps showing plain
// values.
func (f *FilesystemV2) sealEnvironment(env *domain.Environment) (*domain.Environment, error) {
	sealed := *env
	sealed.Spec.Values = make([]domain.KeyValue, len(env.Spec.Values))
	copy(sealed.Spec.Values, env.Spec.Values)

	dir, err := f.WorkspaceDir()
	if err != nil {
		return nil, err
	}
	old, err := readLocalSecrets(dir, env.MetaData.ID)
	if err != nil {
		return nil, err
	}

	stored := map[string]string{}
	for i, kv := range sealed.Spec.Values {
		if !kv.Secret && !kv.Locked && !secret.IsEncrypted(kv.Value) {
			continue
		}
		if kv.ID == "" {
			// The stored value is found by ID, so every secret needs one.
			kv.ID = uuid.NewString()
			env.Spec.Values[i].ID = kv.ID
			sealed.Spec.Values[i].ID = kv.ID
		}
		sealed.Spec.Values[i].Secret = true
		sealed.Spec.Values[i].Value = ""

		switch {
		case kv.Unset:
			// Nothing to store, but keep a value saved since the load.
			if c, ok := old[kv.ID]; ok {
				stored[kv.ID] = c
			}
		case kv.Locked || secret.IsEncrypted(kv.Value):
			// A locked value was never decrypted; store the ciphertext as is.
			stored[kv.ID] = kv.Value
		default:
			if f.secrets == nil {
				return nil, fmt.Errorf("cannot save secret %q: %w", kv.Key, secret.ErrNoKey)
			}
			enc, err := f.secrets.Encrypt(secret.AAD(env.MetaData.ID, kv.Key), kv.Value)
			if err != nil {
				return nil, fmt.Errorf("cannot save secret %q: %w", kv.Key, err)
			}
			stored[kv.ID] = enc
		}
	}

	if err := writeLocalSecrets(dir, env.MetaData.ID, stored); err != nil {
		return nil, fmt.Errorf("cannot save secret values: %w", err)
	}
	return &sealed, nil
}
