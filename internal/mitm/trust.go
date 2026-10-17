package mitm

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ErrManualTrust means this OS has no automatic install; show the
// steps from TrustSteps instead.
var ErrManualTrust = errors.New("install the certificate manually on this system")

// CanInstallTrust reports whether InstallTrust works on this OS.
func CanInstallTrust() bool { return runtime.GOOS == "darwin" }

// InstallTrust adds the CA to the user's trust store. On macOS it adds
// it to the login keychain as a trusted root, which makes macOS ask
// for the user's password. It blocks until the user answers.
func InstallTrust(ctx context.Context, ca *CA) error {
	switch runtime.GOOS {
	case "darwin":
		kc, err := loginKeychain()
		if err != nil {
			return err
		}
		return run(ctx, "security", "add-trusted-cert", "-r", "trustRoot", "-k", kc, ca.CertPath())
	}
	return ErrManualTrust
}

// RemoveTrust takes the CA out of the user's trust store.
func RemoveTrust(ctx context.Context, ca *CA) error {
	switch runtime.GOOS {
	case "darwin":
		kc, err := loginKeychain()
		if err != nil {
			return err
		}
		// Drop the trust setting, then the certificate. Either may be
		// missing already; only a failure to delete a present cert counts.
		_ = run(ctx, "security", "remove-trusted-cert", ca.CertPath())
		sum := sha1.Sum(ca.CertDER())
		err = run(ctx, "security", "delete-certificate", "-Z", strings.ToUpper(hex.EncodeToString(sum[:])), kc)
		if err != nil && strings.Contains(err.Error(), "could not be found") {
			return nil
		}
		return err
	}
	return ErrManualTrust
}

// TrustStep is one manual step to trust the CA.
type TrustStep struct {
	Text    string
	Command string // optional, ready to paste in a terminal
}

// TrustSteps lists how to trust the CA by hand on this OS.
func TrustSteps(ca *CA) []TrustStep {
	pem := ca.CertPath()
	switch runtime.GOOS {
	case "darwin":
		return []TrustStep{
			{Text: "Add the certificate to your login keychain as a trusted root (macOS asks for your password):",
				Command: fmt.Sprintf("security add-trusted-cert -r trustRoot -k ~/Library/Keychains/login.keychain-db %q", pem)},
			{Text: "Or open it in Keychain Access, double-click \"" + ca.Name() + "\" and set \"When using this certificate\" to \"Always Trust\"."},
		}
	case "windows":
		return []TrustStep{
			{Text: "Add the certificate to your user's trusted roots (Windows asks you to confirm):",
				Command: fmt.Sprintf("certutil -user -addstore Root %q", pem)},
		}
	default:
		return []TrustStep{
			{Text: "Debian / Ubuntu:",
				Command: fmt.Sprintf("sudo cp %q /usr/local/share/ca-certificates/chapar-ca.crt && sudo update-ca-certificates", pem)},
			{Text: "Fedora / RHEL / Arch:",
				Command: fmt.Sprintf("sudo trust anchor --store %q", pem)},
			{Text: "Firefox and Chromium on Linux keep their own store: import the certificate under Settings › Certificates › Authorities."},
		}
	}
}

// ClientEnv is a shell snippet that points CLI tools at the proxy and
// makes them trust the CA without touching the OS store.
func ClientEnv(addr string, ca *CA) string {
	pem := ca.CertPath()
	u := "http://" + addr
	if runtime.GOOS == "windows" {
		return strings.Join([]string{
			"$env:HTTP_PROXY=\"" + u + "\"",
			"$env:HTTPS_PROXY=\"" + u + "\"",
			"$env:SSL_CERT_FILE=\"" + pem + "\"",
			"$env:NODE_EXTRA_CA_CERTS=\"" + pem + "\"",
			"$env:REQUESTS_CA_BUNDLE=\"" + pem + "\"",
		}, "\n")
	}
	return strings.Join([]string{
		"export HTTP_PROXY=" + u + " HTTPS_PROXY=" + u,
		fmt.Sprintf("export SSL_CERT_FILE=%q NODE_EXTRA_CA_CERTS=%q REQUESTS_CA_BUNDLE=%q", pem, pem, pem),
	}, "\n")
}

func loginKeychain() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Keychains", "login.keychain-db"), nil
}

func run(ctx context.Context, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s: %s", name, msg)
	}
	return nil
}
