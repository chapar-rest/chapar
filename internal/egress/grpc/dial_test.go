package grpc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
)

// writeKeyPair writes a self-signed certificate and its key as PEM files.
func writeKeyPair(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certPath = filepath.Join(dir, "client.pem")
	keyPath = filepath.Join(dir, "client.key")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}

func tlsSpec(s domain.GRPCSettings) *domain.GRPCRequestSpec {
	return &domain.GRPCRequestSpec{ServerInfo: domain.ServerInfo{Address: "localhost:1"}, Settings: s}
}

func TestDialTLSNeedsNoFiles(t *testing.T) {
	conn, err := (&Service{}).Dial(tlsSpec(domain.GRPCSettings{}))
	if err != nil {
		t.Fatalf("TLS with no files: %v", err)
	}
	_ = conn.Close()
}

// The client key used to be read from the certificate's path, so mutual TLS
// always failed with a key-pair mismatch.
func TestDialMutualTLSReadsTheKeyFile(t *testing.T) {
	cert, key := writeKeyPair(t)
	conn, err := (&Service{}).Dial(tlsSpec(domain.GRPCSettings{ClientCertFile: cert, ClientKeyFile: key}))
	if err != nil {
		t.Fatalf("mutual TLS with a matching pair: %v", err)
	}
	_ = conn.Close()
}

func TestDialMutualTLSNeedsBothFiles(t *testing.T) {
	cert, key := writeKeyPair(t)
	for _, s := range []domain.GRPCSettings{{ClientCertFile: cert}, {ClientKeyFile: key}} {
		if _, err := (&Service{}).Dial(tlsSpec(s)); err == nil || !strings.Contains(err.Error(), "both") {
			t.Errorf("%+v: err = %v, want one asking for both files", s, err)
		}
	}
}

func TestDialRejectsRootWithoutCertificate(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(bad, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Service{}).Dial(tlsSpec(domain.GRPCSettings{RootCertFile: bad})); err == nil {
		t.Fatal("a root file with no certificate was accepted")
	}
}
