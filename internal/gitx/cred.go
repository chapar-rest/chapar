package gitx

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/zalando/go-keyring"
)

// keyringService names Chapar's git credentials in the OS secret store.
const keyringService = "chapar-git"

// IsHTTP reports whether remote is an http or https URL, the kind that
// takes a user name and token. Other remotes use SSH or the file system.
func IsHTTP(remote string) bool {
	u, err := url.Parse(remote)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}

// credKey is the secret store account for remote: its scheme and host, so
// every repository on one server shares a token.
func credKey(remote string) (string, bool) {
	if !IsHTTP(remote) {
		return "", false
	}
	u, _ := url.Parse(remote)
	return u.Scheme + "://" + strings.ToLower(u.Host), true
}

// LoadAuth returns the credentials saved for remote's server, or the zero
// Auth when there are none or remote is not an HTTP URL.
func LoadAuth(remote string) Auth {
	key, ok := credKey(remote)
	if !ok {
		return Auth{}
	}
	v, err := keyring.Get(keyringService, key)
	if err != nil {
		return Auth{}
	}
	var a Auth
	if json.Unmarshal([]byte(v), &a) != nil {
		return Auth{}
	}
	return a
}

// SaveAuth keeps a for remote's server in the OS secret store.
func SaveAuth(remote string, a Auth) error {
	key, ok := credKey(remote)
	if !ok {
		return errors.New("credentials can only be saved for an http or https remote")
	}
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, key, string(data))
}

// ForgetAuth removes the credentials saved for remote's server.
func ForgetAuth(remote string) error {
	key, ok := credKey(remote)
	if !ok {
		return nil
	}
	err := keyring.Delete(keyringService, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
