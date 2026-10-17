package mitm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	caKeyFile  = "ca-key.pem"
	caCertFile = "ca-cert.pem"

	// leafCacheSize bounds the per-host certificates kept in memory.
	leafCacheSize = 512
	// leafValidity stays under the 398 days Apple platforms accept for
	// TLS server certificates.
	leafValidity = 397 * 24 * time.Hour
	caValidity   = 10 * 365 * 24 * time.Hour
)

// CA is the root certificate the proxy signs per-host certificates with.
// It satisfies go-mitmproxy's cert.CA interface.
type CA struct {
	dir  string
	key  *rsa.PrivateKey
	cert *x509.Certificate

	// leafKey is shared by every issued leaf: generating a key per host
	// would slow the first request to each host for no gain.
	leafKey *ecdsa.PrivateKey

	mu    sync.Mutex
	cache map[string]*tls.Certificate
	order []string
}

// CAExists reports whether dir holds a CA key and certificate.
func CAExists(dir string) bool {
	_, errK := os.Stat(filepath.Join(dir, caKeyFile))
	_, errC := os.Stat(filepath.Join(dir, caCertFile))
	return errK == nil && errC == nil
}

// LoadCA reads the CA stored in dir.
func LoadCA(dir string) (*CA, error) {
	keyPEM, err := os.ReadFile(filepath.Join(dir, caKeyFile))
	if err != nil {
		return nil, err
	}
	certPEM, err := os.ReadFile(filepath.Join(dir, caCertFile))
	if err != nil {
		return nil, err
	}
	kb, _ := pem.Decode(keyPEM)
	if kb == nil {
		return nil, errors.New("CA key file has no PEM block")
	}
	k, err := x509.ParsePKCS8PrivateKey(kb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("CA key is not RSA")
	}
	cb, _ := pem.Decode(certPEM)
	if cb == nil {
		return nil, errors.New("CA certificate file has no PEM block")
	}
	c, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}
	return newCA(dir, rk, c)
}

// LoadOrCreateCA loads the CA in dir, creating one when none exists.
func LoadOrCreateCA(dir string) (*CA, error) {
	if CAExists(dir) {
		return LoadCA(dir)
	}
	return CreateCA(dir)
}

// CreateCA generates a new CA and writes it to dir, replacing any CA there.
func CreateCA(dir string) (*CA, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serial, err := randSerial()
	if err != nil {
		return nil, err
	}
	name := "Chapar Proxy CA"
	if h, err := os.Hostname(); err == nil && h != "" {
		name += " (" + strings.TrimSuffix(h, ".local") + ")"
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: name, Organization: []string{"Chapar"}},
		NotBefore:             now.Add(-24 * time.Hour),
		NotAfter:              now.Add(caValidity),
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLenZero:        true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(filepath.Join(dir, caKeyFile), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return nil, err
	}
	if err := writeFileAtomic(filepath.Join(dir, caCertFile), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return nil, err
	}
	return newCA(dir, key, c)
}

// RemoveCA deletes the CA files in dir.
func RemoveCA(dir string) error {
	for _, f := range []string{caKeyFile, caCertFile} {
		if err := os.Remove(filepath.Join(dir, f)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func newCA(dir string, key *rsa.PrivateKey, c *x509.Certificate) (*CA, error) {
	lk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &CA{dir: dir, key: key, cert: c, leafKey: lk, cache: map[string]*tls.Certificate{}}, nil
}

// GetRootCA returns the CA certificate.
func (ca *CA) GetRootCA() *x509.Certificate { return ca.cert }

// CertPath is the PEM file holding only the CA certificate.
func (ca *CA) CertPath() string { return filepath.Join(ca.dir, caCertFile) }

// CertPEM returns the CA certificate in PEM form.
func (ca *CA) CertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw})
}

// CertDER returns the CA certificate in DER form.
func (ca *CA) CertDER() []byte { return ca.cert.Raw }

// Name is the CA's common name, as the OS trust store lists it.
func (ca *CA) Name() string { return ca.cert.Subject.CommonName }

// NotAfter is when the CA expires.
func (ca *CA) NotAfter() time.Time { return ca.cert.NotAfter }

// Fingerprint is the SHA-256 of the certificate as colon-separated hex.
func (ca *CA) Fingerprint() string {
	sum := sha256.Sum256(ca.cert.Raw)
	h := strings.ToUpper(hex.EncodeToString(sum[:]))
	var b strings.Builder
	for i := 0; i < len(h); i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(h[i : i+2])
	}
	return b.String()
}

// GetCert returns a certificate for host signed by the CA.
func (ca *CA) GetCert(host string) (*tls.Certificate, error) {
	ca.mu.Lock()
	if c, ok := ca.cache[host]; ok {
		ca.mu.Unlock()
		return c, nil
	}
	ca.mu.Unlock()

	c, err := ca.issue(host)
	if err != nil {
		return nil, err
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()
	if existing, ok := ca.cache[host]; ok {
		return existing, nil
	}
	if len(ca.order) >= leafCacheSize {
		delete(ca.cache, ca.order[0])
		ca.order = ca.order[1:]
	}
	ca.cache[host] = c
	ca.order = append(ca.order, host)
	return c, nil
}

func (ca *CA) issue(host string) (*tls.Certificate, error) {
	serial, err := randSerial()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host, Organization: []string{"Chapar"}},
		NotBefore:    now.Add(-24 * time.Hour),
		NotAfter:     now.Add(leafValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &ca.leafKey.PublicKey, ca.key)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{
		Certificate: [][]byte{der, ca.cert.Raw},
		PrivateKey:  ca.leafKey,
		Leaf:        leaf,
	}, nil
}

// TrustedByOS reports whether the OS trust store accepts certificates
// this CA issues. It verifies a fresh leaf against the system roots,
// which on macOS and Windows asks the platform verifier, so keychain
// trust settings count. Any verify error means "not trusted": the
// platform verifiers don't report an untrusted root as one error type.
func (ca *CA) TrustedByOS() (bool, error) {
	const probe = "chapar-trust-check.invalid"
	c, err := ca.issue(probe)
	if err != nil {
		return false, err
	}
	_, err = c.Leaf.Verify(x509.VerifyOptions{DNSName: probe})
	return err == nil, nil
}

func randSerial() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
