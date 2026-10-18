package mitm

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func TestCurl(t *testing.T) {
	f := &Flow{
		Method: "POST", URL: "https://api.example.com/items?q=a b",
		ReqHeader: http.Header{
			"Content-Type":    {"application/json"},
			"Cookie":          {`sid=it's`},
			"Accept-Encoding": {"gzip, br"},
			"Content-Length":  {"20"},
		},
		ReqBody: []byte(`{"name":"O'Brien"}`),
	}
	got := Curl(f)
	want := `curl 'https://api.example.com/items?q=a b' \
  -H 'Content-Type: application/json' \
  -H 'Cookie: sid=it'\''s' \
  --compressed \
  --data-raw '{"name":"O'\''Brien"}'`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}

	if c := Curl(&Flow{Method: "DELETE", URL: "https://x.io/a"}); c != "curl -X DELETE https://x.io/a" {
		t.Fatalf("delete: %s", c)
	}
	bin := Curl(&Flow{Method: "PUT", URL: "https://x.io/b", ReqBody: []byte{0xff, 0x00}})
	if !strings.HasPrefix(bin, "printf %s /wA= | base64 -d | curl -X PUT") || !strings.Contains(bin, "--data-binary @-") {
		t.Fatalf("binary: %s", bin)
	}
}

// The quoting must survive a real shell: echo each argument back.
func TestCurlQuotingInShell(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	val := `a'b"c $HOME \n` + "`x`"
	out, err := exec.Command(sh, "-c", "printf %s "+shellQuote(val)).Output()
	if err != nil || string(out) != val {
		t.Fatalf("got %q, %v", out, err)
	}
}

// The command must reproduce the request when actually run.
func TestCurlReplays(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("no curl")
	}
	type got struct {
		method, query, cookie string
		body                  []byte
	}
	ch := make(chan got, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ch <- got{r.Method, r.URL.RawQuery, r.Header.Get("Cookie"), b}
	}))
	defer srv.Close()

	for _, body := range [][]byte{[]byte(`{"name":"O'Brien $x"}`), {0xff, 0x00, 0x7f}} {
		f := &Flow{
			Method: "PATCH", URL: srv.URL + "/p?a=1&b=%20",
			ReqHeader: http.Header{"Cookie": {`k="v w"`}},
			ReqBody:   body,
		}
		if out, err := exec.Command("sh", "-c", Curl(f)+" -s").CombinedOutput(); err != nil {
			t.Fatalf("%v: %s\n%s", err, out, Curl(f))
		}
		g := <-ch
		if g.method != "PATCH" || g.query != "a=1&b=%20" || g.cookie != `k="v w"` || string(g.body) != string(body) {
			t.Fatalf("server got %+v", g)
		}
	}
}

// shellPath must name the same file once a shell expands it.
func TestShellPathExpands(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, p := range []string{
		home + "/Library/Application Support/chapar/proxy/ca-cert.pem",
		home + `/odd "dir" $x/a.pem`,
		"/etc/ssl/ca.pem",
	} {
		out, err := exec.Command(sh, "-c", "printf %s "+shellPath(p)).Output()
		if err != nil || string(out) != p {
			t.Fatalf("%s -> %s -> %q (%v)", p, shellPath(p), out, err)
		}
	}
	if got := shellPath(home + "/a b.pem"); got != `"$HOME/a b.pem"` {
		t.Fatalf("got %s", got)
	}
}
