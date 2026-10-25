package update

import (
	"crypto/ed25519"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// SigSuffix is appended to an asset's name to get its signature asset.
const SigSuffix = ".sig"

// releaseKey is the public half of the key the release workflows sign assets
// with (build/updatesign). Rotating it means older apps reject new releases
// and fall back to a manual download.
//
//go:embed release_key.pub
var releaseKey string

// signedMessage is what a signature covers: the asset's name, so one asset's
// signature can't vouch for another, and its SHA-256.
func signedMessage(name, sha256Hex string) []byte {
	return []byte("chapar-update-v1\n" + name + "\n" + strings.ToLower(sha256Hex) + "\n")
}

// Sign returns the base64 signature file contents for an asset.
func Sign(key ed25519.PrivateKey, name, sha256Hex string) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(key, signedMessage(name, sha256Hex))) + "\n"
}

// ParsePublicKey decodes a base64 ed25519 public key.
func ParsePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, errors.New("invalid update public key")
	}
	return ed25519.PublicKey(b), nil
}

// ParsePrivateKey decodes a base64 ed25519 seed.
func ParsePrivateKey(s string) (ed25519.PrivateKey, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil || len(b) != ed25519.SeedSize {
		return nil, errors.New("invalid update signing key: want a base64 ed25519 seed")
	}
	return ed25519.NewKeyFromSeed(b), nil
}

// verify checks sig (a signature file's contents) for the asset.
func verify(pub ed25519.PublicKey, name, sha256Hex string, sig []byte) error {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil {
		return fmt.Errorf("%s: malformed signature", name)
	}
	if !ed25519.Verify(pub, signedMessage(name, sha256Hex), raw) {
		return fmt.Errorf("%s: signature does not match", name)
	}
	return nil
}
