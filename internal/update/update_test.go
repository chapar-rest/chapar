package update

import (
	"archive/zip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"v0.11.0", "v0.12.0", true},
		{"v0.11.0", "v0.11.1", true},
		{"v0.11.9", "v0.12.0", true},
		{"v0.9.1", "v0.10.0", true},
		{"v1.0.0", "v0.99.0", false},
		{"v0.11.0", "v0.11.0", false},
		{"v0.12.0-rc.1", "v0.12.0", true},
		{"v0.12.0", "v0.12.0-rc.2", false},
		{"0.1.0", "v0.12.0", false}, // dev build default
		{"dev", "v0.12.0", false},
		{"v0.11.0", "nightly", false},
		{"v0.11", "v0.12.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.cur, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.cur, c.latest, got, c.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	cases := map[[3]string]string{
		{"v0.12.0", "darwin", "arm64"}:  "chapar-macos-v0.12.0-arm64.dmg",
		{"0.12.0", "darwin", "amd64"}:   "chapar-macos-v0.12.0-amd64.dmg",
		{"v0.12.0", "windows", "amd64"}: "chapar-windows-v0.12.0-amd64.zip",
		{"v0.12.0", "linux", "arm64"}:   "chapar-linux-v0.12.0-arm64.tar.xz",
	}
	for in, want := range cases {
		if got, _ := AssetName(in[0], in[1], in[2]); got != want {
			t.Errorf("AssetName%v = %q, want %q", in, got, want)
		}
	}
	if _, ok := AssetName("v1.0.0", "plan9", "amd64"); ok {
		t.Error("plan9 has no asset")
	}
}

func TestEmbeddedKeyParses(t *testing.T) {
	if _, err := ParsePublicKey(releaseKey); err != nil {
		t.Fatal(err)
	}
}

type fakeRelease struct {
	pub     string
	payload []byte
	files   map[string][]byte
	srv     *httptest.Server
}

func newFakeRelease(t *testing.T, name string, payload []byte) *fakeRelease {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	f := &fakeRelease{
		pub:     base64.StdEncoding.EncodeToString(pub),
		payload: payload,
		files: map[string][]byte{
			name:             payload,
			name + SigSuffix: []byte(Sign(priv, name, hex.EncodeToString(sum[:]))),
		},
	}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := f.files[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeRelease) release(digest string) Release {
	r := Release{Version: "v9.9.9"}
	for name := range f.files {
		a := Asset{Name: name, URL: f.srv.URL + "/" + name}
		if !strings.HasSuffix(name, SigSuffix) {
			a.Digest = digest
		}
		r.Assets = append(r.Assets, a)
	}
	return r
}

func TestDownloadVerifies(t *testing.T) {
	const name = "chapar-macos-v9.9.9-arm64.dmg"
	payload := []byte(strings.Repeat("chapar", 1000))
	sum := sha256.Sum256(payload)
	goodDigest := "sha256:" + hex.EncodeToString(sum[:])

	t.Run("ok", func(t *testing.T) {
		f := newFakeRelease(t, name, payload)
		var last int64
		path, err := Client{}.download(context.Background(), f.release(goodDigest), name, f.pub, func(done, _ int64) { last = done })
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(path) }()
		got, _ := os.ReadFile(path)
		if string(got) != string(payload) || last != int64(len(payload)) {
			t.Fatalf("got %d bytes, progress %d", len(got), last)
		}
	})

	t.Run("no digest is fine", func(t *testing.T) {
		f := newFakeRelease(t, name, payload)
		path, err := Client{}.download(context.Background(), f.release(""), name, f.pub, func(int64, int64) {})
		if err != nil {
			t.Fatal(err)
		}
		_ = os.Remove(path)
	})

	fail := func(t *testing.T, f *fakeRelease, r Release, pub, want string) {
		t.Helper()
		path, err := Client{}.download(context.Background(), r, name, pub, func(int64, int64) {})
		if err == nil {
			_ = os.Remove(path)
			t.Fatal("download succeeded")
		}
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %v, want %q", err, want)
		}
		if path != "" {
			t.Fatalf("left %s behind", path)
		}
	}

	t.Run("tampered", func(t *testing.T) {
		f := newFakeRelease(t, name, payload)
		f.files[name] = append([]byte("evil"), payload...)
		fail(t, f, f.release(""), f.pub, "signature does not match")
	})
	t.Run("wrong key", func(t *testing.T) {
		f := newFakeRelease(t, name, payload)
		other := newFakeRelease(t, name, payload)
		fail(t, f, f.release(goodDigest), other.pub, "signature does not match")
	})
	t.Run("signature for another asset", func(t *testing.T) {
		f := newFakeRelease(t, name, payload)
		g := newFakeRelease(t, "chapar-macos-v9.9.9-amd64.dmg", payload)
		g.pub = f.pub
		f.files[name+SigSuffix] = g.files["chapar-macos-v9.9.9-amd64.dmg"+SigSuffix]
		fail(t, f, f.release(goodDigest), f.pub, "signature does not match")
	})
	t.Run("digest mismatch", func(t *testing.T) {
		f := newFakeRelease(t, name, payload)
		fail(t, f, f.release("sha256:"+strings.Repeat("0", 64)), f.pub, "corrupt")
	})
	t.Run("unsigned", func(t *testing.T) {
		f := newFakeRelease(t, name, payload)
		delete(f.files, name+SigSuffix)
		fail(t, f, f.release(goodDigest), f.pub, "not signed")
	})
}

func TestLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Chapar/test" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte(`{"tag_name":"v0.12.0","html_url":"https://x/r","body":"notes",
			"assets":[{"name":"a.dmg","browser_download_url":"https://x/a.dmg","size":3,"digest":"sha256:ab"}]}`))
	}))
	defer srv.Close()
	r, err := Client{LatestURL: srv.URL, UserAgent: "Chapar/test"}.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a, ok := r.Asset("a.dmg")
	if r.Version != "v0.12.0" || r.URL != "https://x/r" || r.Notes != "notes" || !ok || a.Digest != "sha256:ab" || a.Size != 3 {
		t.Fatalf("got %+v", r)
	}
}

func TestSwap(t *testing.T) {
	dir := t.TempDir()
	cur, next, old := filepath.Join(dir, "Chapar.exe"), filepath.Join(dir, "next"), filepath.Join(dir, "Chapar.exe.old")
	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(p string) string { b, _ := os.ReadFile(p); return string(b) }

	write(cur, "v1")
	write(next, "v2")
	write(old, "stale")
	if err := swap(cur, next, old); err != nil {
		t.Fatal(err)
	}
	if read(cur) != "v2" || read(old) != "v1" {
		t.Fatalf("cur=%q old=%q", read(cur), read(old))
	}

	// A missing replacement leaves the current file where it was.
	if err := swap(cur, filepath.Join(dir, "missing"), old); err == nil {
		t.Fatal("swap succeeded")
	}
	if read(cur) != "v2" {
		t.Fatalf("cur=%q after failed swap", read(cur))
	}
}

func TestExtractExe(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "r.zip")
	makeZip := func(files map[string]string) {
		f, _ := os.Create(zipPath)
		zw := zip.NewWriter(f)
		for name, body := range files {
			w, _ := zw.Create(name)
			_, _ = w.Write([]byte(body))
		}
		_ = zw.Close()
		_ = f.Close()
	}
	dest := filepath.Join(dir, "out.exe")

	makeZip(map[string]string{"Chapar.exe": "binary", "README.txt": "hi"})
	if err := extractExe(zipPath, dest); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dest); string(b) != "binary" {
		t.Fatalf("got %q", b)
	}

	makeZip(map[string]string{"README.txt": "hi"})
	if err := extractExe(zipPath, dest); err == nil {
		t.Fatal("extracted from a zip with no exe")
	}
	makeZip(map[string]string{"a.exe": "1", "b.EXE": "2"})
	if err := extractExe(zipPath, dest); err == nil {
		t.Fatal("extracted from a zip with two exes")
	}
}
