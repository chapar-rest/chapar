package repository

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/secret"
)

// Environment values marked secret are encrypted here, at the edge of the
// filesystem, so the rest of the app only ever sees plain values. A value that
// cannot be decrypted - no key, or the key is locked - stays as ciphertext and
// is marked Locked, and is written back exactly as it was read.

// openEnvironment decrypts the secret values of a freshly loaded environment.
func (f *FilesystemV2) openEnvironment(env *domain.Environment) {
	if env == nil {
		return
	}
	for i, kv := range env.Spec.Values {
		if !secret.IsEncrypted(kv.Value) {
			env.Spec.Values[i].Locked = false
			continue
		}
		env.Spec.Values[i].Secret = true
		if f.secrets == nil || !f.secrets.Unlocked() {
			env.Spec.Values[i].Locked = true
			continue
		}
		plain, err := f.secrets.Decrypt(secret.AAD(env.MetaData.ID, kv.Key), kv.Value)
		if err != nil {
			env.Spec.Values[i].Locked = true
			continue
		}
		env.Spec.Values[i].Value = plain
		env.Spec.Values[i].Locked = false
	}
}

// sealEnvironment returns a copy of env with its secret values encrypted, ready
// to be written. The original is left untouched so the UI keeps showing plain
// values.
func (f *FilesystemV2) sealEnvironment(env *domain.Environment) (*domain.Environment, error) {
	sealed := *env
	sealed.Spec.Values = make([]domain.KeyValue, len(env.Spec.Values))
	copy(sealed.Spec.Values, env.Spec.Values)

	for i, kv := range sealed.Spec.Values {
		// A locked value was never decrypted; write the ciphertext back as is.
		if kv.Locked || secret.IsEncrypted(kv.Value) {
			sealed.Spec.Values[i].Secret = true
			continue
		}
		if !kv.Secret {
			continue
		}
		if f.secrets == nil {
			return nil, fmt.Errorf("cannot save secret %q: %w", kv.Key, secret.ErrNoKey)
		}
		enc, err := f.secrets.Encrypt(secret.AAD(env.MetaData.ID, kv.Key), kv.Value)
		if err != nil {
			return nil, fmt.Errorf("cannot save secret %q: %w", kv.Key, err)
		}
		sealed.Spec.Values[i].Value = enc
	}
	return &sealed, nil
}
