package egress

import (
	"crypto/tls"
	"fmt"
	"net/http/httptrace"
	"sync"
	"time"
)

// HTTPTraceCollector records httptrace.ClientTrace phases into TimelineSteps.
type HTTPTraceCollector struct {
	mu sync.Mutex

	dnsStart, dnsDone         time.Time
	connectStart, connectDone time.Time
	tlsStart, tlsDone         time.Time
	wroteRequest              time.Time
	gotFirstByte              time.Time
	start                     time.Time

	dnsHost  string
	connAddr string
	reused   bool
	tlsVers  string
	tlsErr   string
	dnsErr   string
	connErr  string
	writeErr string
}

// NewHTTPTraceCollector starts timing from now.
func NewHTTPTraceCollector() *HTTPTraceCollector {
	return &HTTPTraceCollector{start: time.Now()}
}

// ClientTrace returns a ClientTrace wired to this collector.
func (c *HTTPTraceCollector) ClientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.dnsStart = time.Now()
			c.dnsHost = info.Host
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.dnsDone = time.Now()
			if info.Err != nil {
				c.dnsErr = info.Err.Error()
			}
		},
		ConnectStart: func(_, addr string) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.connectStart = time.Now()
			c.connAddr = addr
		},
		ConnectDone: func(_, addr string, err error) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.connectDone = time.Now()
			c.connAddr = addr
			if err != nil {
				c.connErr = err.Error()
			}
		},
		TLSHandshakeStart: func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.tlsDone = time.Now()
			c.tlsVers = tlsVersionName(state.Version)
			if err != nil {
				c.tlsErr = err.Error()
			}
		},
		GotConn: func(info httptrace.GotConnInfo) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.reused = info.Reused
			if info.Conn != nil {
				c.connAddr = info.Conn.RemoteAddr().String()
			}
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.wroteRequest = time.Now()
			if info.Err != nil {
				c.writeErr = info.Err.Error()
			}
		},
		GotFirstResponseByte: func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.gotFirstByte = time.Now()
		},
	}
}

// Steps returns network timeline steps observed so far. downloadEnd marks body read completion.
func (c *HTTPTraceCollector) Steps(downloadEnd time.Time) []TimelineStep {
	c.mu.Lock()
	defer c.mu.Unlock()

	var steps []TimelineStep
	add := func(name string, start, end time.Time, detail, errStr string) {
		if start.IsZero() {
			return
		}
		if end.IsZero() {
			end = start
		}
		steps = append(steps, TimelineStep{
			Name:     name,
			Phase:    TimelinePhaseNetwork,
			Duration: end.Sub(start),
			Detail:   detail,
			Err:      errStr,
		})
	}

	if !c.dnsStart.IsZero() {
		detail := c.dnsHost
		add("DNS Lookup", c.dnsStart, c.dnsDone, detail, c.dnsErr)
	}
	if !c.connectStart.IsZero() {
		detail := c.connAddr
		if c.reused {
			detail = fmt.Sprintf("%s (connection reused)", detail)
		}
		add("TCP Connect", c.connectStart, c.connectDone, detail, c.connErr)
	}
	if !c.tlsStart.IsZero() {
		detail := c.tlsVers
		add("TLS Handshake", c.tlsStart, c.tlsDone, detail, c.tlsErr)
	}
	if !c.wroteRequest.IsZero() {
		start := c.start
		if !c.tlsDone.IsZero() {
			start = c.tlsDone
		} else if !c.connectDone.IsZero() {
			start = c.connectDone
		} else if !c.dnsDone.IsZero() {
			start = c.dnsDone
		}
		add("Request Sent", start, c.wroteRequest, "Request headers and body written", c.writeErr)
	}
	if !c.gotFirstByte.IsZero() && !c.wroteRequest.IsZero() {
		add("Waiting (TTFB)", c.wroteRequest, c.gotFirstByte, "Time to first response byte", "")
	}
	if !c.gotFirstByte.IsZero() && !downloadEnd.IsZero() {
		add("Download", c.gotFirstByte, downloadEnd, "Response body transfer", "")
	}
	return steps
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		if v == 0 {
			return ""
		}
		return fmt.Sprintf("0x%04x", v)
	}
}
