package mitm

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// appResolver names the app behind a client connection. For local
// clients it asks the OS which process owns the connection's source
// port; for other devices it uses the device's address. The
// User-Agent's product token is the fallback.
type appResolver struct {
	mu     sync.Mutex
	byAddr map[string]string // client addr -> app, per connection
	byPID  map[int]string
}

func newAppResolver() *appResolver {
	return &appResolver{byAddr: map[string]string{}, byPID: map[int]string{}}
}

func (r *appResolver) lookup(clientAddr, ua string) string {
	if clientAddr == "" {
		return uaProduct(ua)
	}
	r.mu.Lock()
	if app, ok := r.byAddr[clientAddr]; ok {
		r.mu.Unlock()
		return app
	}
	r.mu.Unlock()

	app := r.resolve(clientAddr)
	if app == "" {
		app = uaProduct(ua)
	}

	r.mu.Lock()
	// A client port is reused by later connections; keep the cache small
	// so stale entries go before that matters.
	if len(r.byAddr) > 4096 {
		r.byAddr = map[string]string{}
	}
	r.byAddr[clientAddr] = app
	r.mu.Unlock()
	return app
}

func (r *appResolver) resolve(clientAddr string) string {
	host, port, err := net.SplitHostPort(clientAddr)
	if err != nil {
		return ""
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	if !ip.IsLoopback() {
		return "Device " + host
	}
	pid := pidForPort(port)
	if pid <= 0 {
		return ""
	}
	r.mu.Lock()
	if app, ok := r.byPID[pid]; ok {
		r.mu.Unlock()
		return app
	}
	r.mu.Unlock()
	app := appForPID(pid)
	if app != "" {
		r.mu.Lock()
		r.byPID[pid] = app
		r.mu.Unlock()
	}
	return app
}

// pidForPort finds the process with an open TCP socket whose local port
// is port, other than this one.
func pidForPort(port string) int {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "lsof", "-nP", "-iTCP@127.0.0.1:"+port, "-iTCP@[::1]:"+port, "-sTCP:ESTABLISHED", "-Fpn").Output()
	if err != nil && len(out) == 0 {
		return 0
	}
	self := os.Getpid()
	pid := 0
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(line[1:])
		case 'n':
			// "n127.0.0.1:52344->127.0.0.1:9090": the client's socket has
			// the port on the left.
			local, _, _ := strings.Cut(line[1:], "->")
			if pid != self && strings.HasSuffix(local, ":"+port) {
				return pid
			}
		}
	}
	return 0
}

// appForPID names the app a process belongs to: the outermost .app
// bundle on macOS, else the executable's name.
func appForPID(pid int) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	exe := strings.TrimSpace(string(out))
	if exe == "" {
		return ""
	}
	if i := strings.Index(exe, ".app/"); i >= 0 {
		return filepath.Base(exe[:i])
	}
	return filepath.Base(exe)
}

// uaProduct returns the first product token of a User-Agent, such as
// "curl" for "curl/8.4.0". Browser UAs all start with "Mozilla", so
// they are named by their engine marker instead.
func uaProduct(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return ""
	}
	if strings.HasPrefix(ua, "Mozilla/") {
		for _, b := range []struct{ marker, name string }{
			{"Edg/", "Edge"}, {"OPR/", "Opera"}, {"Firefox/", "Firefox"},
			{"Chrome/", "Chrome"}, {"Safari/", "Safari"},
		} {
			if strings.Contains(ua, b.marker) {
				return b.name
			}
		}
		return "Browser"
	}
	tok, _, _ := strings.Cut(ua, " ")
	name, _, _ := strings.Cut(tok, "/")
	return name
}
