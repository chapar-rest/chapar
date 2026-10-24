package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// AssetName is the release file for the app on goos/goarch, following the
// artifact pattern in yoga.toml.
func AssetName(version, goos, goarch string) (string, bool) {
	v := "v" + strings.TrimPrefix(version, "v")
	switch goos {
	case "darwin":
		return "chapar-macos-" + v + "-" + goarch + ".dmg", true
	case "windows":
		return "chapar-windows-" + v + "-" + goarch + ".zip", true
	case "linux":
		return "chapar-linux-" + v + "-" + goarch + ".tar.xz", true
	}
	return "", false
}

// Progress reports bytes downloaded so far out of total (0 when unknown).
type Progress func(done, total int64)

// Download fetches the named asset of r into a temp file and checks it
// against its signature, and GitHub's digest when there is one. The caller
// removes the returned file.
func (c Client) Download(ctx context.Context, r Release, name string, progress Progress) (string, error) {
	return c.download(ctx, r, name, releaseKey, progress)
}

func (c Client) download(ctx context.Context, r Release, name, pubKey string, progress Progress) (string, error) {
	pub, err := ParsePublicKey(pubKey)
	if err != nil {
		return "", err
	}
	a, ok := r.Asset(name)
	if !ok {
		return "", fmt.Errorf("%s has no %s", r.Version, name)
	}
	sigAsset, ok := r.Asset(name + SigSuffix)
	if !ok {
		return "", fmt.Errorf("%s is not signed", name)
	}
	sig, err := c.fetch(ctx, sigAsset.URL, io.Discard, nil)
	if err != nil {
		return "", err
	}

	f, err := os.CreateTemp("", "chapar-update-*-"+name)
	if err != nil {
		return "", err
	}
	path := f.Name()
	h := sha256.New()
	_, err = c.fetch(ctx, a.URL, io.MultiWriter(f, h), progress)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		sum := hex.EncodeToString(h.Sum(nil))
		if want, ok := strings.CutPrefix(a.Digest, "sha256:"); ok && !strings.EqualFold(want, sum) {
			err = fmt.Errorf("%s: download is corrupt", name)
		} else {
			err = verify(pub, name, sum, sig)
		}
	}
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

// fetch copies url's body to w. With a nil progress the body is small and
// returned as well.
func (c Client) fetch(ctx context.Context, url string, w io.Writer, progress Progress) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", url, resp.Status)
	}
	if progress == nil {
		return io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	}
	pw := &progressWriter{w: w, total: max(resp.ContentLength, 0), report: progress}
	_, err = io.Copy(pw, resp.Body)
	return nil, err
}

type progressWriter struct {
	w           io.Writer
	done, total int64
	report      Progress
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.done += int64(n)
	p.report(p.done, p.total)
	return n, err
}
