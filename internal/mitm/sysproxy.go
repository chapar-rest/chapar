package mitm

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SystemProxy is the OS-wide HTTPS proxy setting. Chapar only reads it.
type SystemProxy struct {
	Enabled bool
	Host    string
	Port    int
}

// ErrSystemProxyUnknown means this OS's setting can't be read.
var ErrSystemProxyUnknown = errors.New("system proxy setting can't be read on this system")

// ReadSystemProxy returns the OS-wide HTTPS proxy setting.
func ReadSystemProxy() (SystemProxy, error) {
	if runtime.GOOS != "darwin" {
		return SystemProxy{}, ErrSystemProxyUnknown
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "scutil", "--proxy").Output()
	if err != nil {
		return SystemProxy{}, err
	}
	return parseScutilProxy(out), nil
}

// parseScutilProxy reads the HTTPS entries of `scutil --proxy` output.
func parseScutilProxy(out []byte) SystemProxy {
	var sp SystemProxy
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "HTTPSEnable":
			sp.Enabled = v == "1"
		case "HTTPSProxy":
			sp.Host = v
		case "HTTPSPort":
			sp.Port, _ = strconv.Atoi(v)
		}
	}
	return sp
}

// PointsTo reports whether the setting sends traffic to a proxy on this
// machine at port.
func (sp SystemProxy) PointsTo(port int) bool {
	if !sp.Enabled || sp.Port != port {
		return false
	}
	switch sp.Host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	for _, a := range LANAddrs() {
		if a == sp.Host {
			return true
		}
	}
	return false
}

// LANAddrs lists this machine's IPv4 addresses on active non-loopback
// interfaces, the ones other devices can reach.
func LANAddrs() []string {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipn.IP.To4()
			if ip4 == nil || ip4.IsLinkLocalUnicast() {
				continue
			}
			out = append(out, ip4.String())
		}
	}
	return out
}
