// Command updatesign makes and uses the key that signs release assets, so
// Chapar's in-app updater can tell a real release from a tampered one.
//
//	go run ./build/updatesign genkey -out signing.key
//	UPDATE_SIGNING_KEY=$(cat signing.key) go run ./build/updatesign sign dist/darwin/*.dmg
//
// genkey writes the private seed to -out and the public key to
// internal/update/release_key.pub, which the app embeds. The seed goes into
// the UPDATE_SIGNING_KEY repository secret and nowhere else.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/update"
)

const pubKeyPath = "internal/update/release_key.pub"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "genkey":
		fs := flag.NewFlagSet("genkey", flag.ExitOnError)
		out := fs.String("out", "", "file to write the private seed to")
		_ = fs.Parse(os.Args[2:])
		if *out == "" {
			usage()
		}
		err = genkey(*out)
	case "sign":
		err = sign(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "updatesign:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: updatesign genkey -out FILE | updatesign sign ASSET...")
	os.Exit(2)
}

func genkey(out string) error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	seed := base64.StdEncoding.EncodeToString(priv.Seed()) + "\n"
	if err := os.WriteFile(out, []byte(seed), 0o600); err != nil {
		return err
	}
	return os.WriteFile(pubKeyPath, []byte(base64.StdEncoding.EncodeToString(pub)+"\n"), 0o644)
}

// sign writes ASSET.sig next to each asset, then checks it against the
// embedded public key so a release never ships signatures the app rejects.
func sign(paths []string) error {
	if len(paths) == 0 {
		usage()
	}
	key, err := update.ParsePrivateKey(os.Getenv("UPDATE_SIGNING_KEY"))
	if err != nil {
		return err
	}
	pubFile, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return err
	}
	pub, err := update.ParsePublicKey(string(pubFile))
	if err != nil {
		return err
	}
	if !key.Public().(ed25519.PublicKey).Equal(pub) {
		return fmt.Errorf("UPDATE_SIGNING_KEY does not match %s", pubKeyPath)
	}
	for _, p := range paths {
		sum, err := fileSHA256(p)
		if err != nil {
			return err
		}
		sig := update.Sign(key, filepath.Base(p), sum)
		if err := os.WriteFile(p+update.SigSuffix, []byte(sig), 0o644); err != nil {
			return err
		}
		fmt.Println(p + update.SigSuffix)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
