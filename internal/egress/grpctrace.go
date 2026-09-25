package egress

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/stats"
)

// GRPCTraceCollector implements stats.Handler and records network timeline steps.
type GRPCTraceCollector struct {
	mu sync.Mutex

	addr     string
	insecure bool

	connBegin, connEnd time.Time
	rpcBegin           time.Time
	outHeader          time.Time
	inHeader           time.Time
	inPayload          time.Time
	rpcEnd             time.Time

	rpcErr string
	method string
}

// NewGRPCTraceCollector builds a collector for the given target.
func NewGRPCTraceCollector(addr string, insecure bool) *GRPCTraceCollector {
	return &GRPCTraceCollector{addr: addr, insecure: insecure}
}

func (c *GRPCTraceCollector) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.method = info.FullMethodName
	return ctx
}

func (c *GRPCTraceCollector) HandleRPC(_ context.Context, s stats.RPCStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	switch st := s.(type) {
	case *stats.Begin:
		c.rpcBegin = now
	case *stats.OutHeader:
		c.outHeader = now
	case *stats.InHeader:
		c.inHeader = now
	case *stats.InPayload:
		if c.inPayload.IsZero() {
			c.inPayload = now
		}
	case *stats.End:
		c.rpcEnd = now
		if st.Error != nil {
			c.rpcErr = st.Error.Error()
		}
	}
}

func (c *GRPCTraceCollector) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (c *GRPCTraceCollector) HandleConn(_ context.Context, s stats.ConnStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	switch s.(type) {
	case *stats.ConnBegin:
		c.connBegin = now
	case *stats.ConnEnd:
		c.connEnd = now
	}
}

// Steps returns network timeline steps collected during the RPC.
func (c *GRPCTraceCollector) Steps() []TimelineStep {
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

	sec := "TLS"
	if c.insecure {
		sec = "insecure"
	}
	connDetail := fmt.Sprintf("%s (%s)", c.addr, sec)
	if !c.connBegin.IsZero() {
		end := c.connEnd
		if end.IsZero() {
			// Conn still open; use RPC begin or out header as connect complete.
			end = c.rpcBegin
			if end.IsZero() {
				end = c.outHeader
			}
			if end.IsZero() {
				end = c.connBegin
			}
		}
		add("Connect", c.connBegin, end, connDetail, "")
	} else if !c.rpcBegin.IsZero() {
		// Connection may have been reused before this collector was attached.
		add("Connect", c.rpcBegin, c.rpcBegin, connDetail+" (ready)", "")
	}

	sentStart := c.rpcBegin
	sentEnd := c.outHeader
	if sentEnd.IsZero() {
		sentEnd = c.rpcBegin
	}
	if !sentStart.IsZero() {
		detail := c.method
		add("Request Sent", sentStart, sentEnd, detail, "")
	}

	waitStart := c.outHeader
	if waitStart.IsZero() {
		waitStart = c.rpcBegin
	}
	waitEnd := c.inHeader
	if waitEnd.IsZero() {
		waitEnd = c.inPayload
	}
	if !waitStart.IsZero() && !waitEnd.IsZero() {
		add("Waiting", waitStart, waitEnd, "Waiting for response headers/payload", "")
	}

	recvStart := c.inHeader
	if recvStart.IsZero() {
		recvStart = c.inPayload
	}
	if !recvStart.IsZero() && !c.rpcEnd.IsZero() {
		add("Receive", recvStart, c.rpcEnd, "Response messages and trailers", c.rpcErr)
	} else if !c.rpcBegin.IsZero() && !c.rpcEnd.IsZero() {
		add("RPC", c.rpcBegin, c.rpcEnd, c.method, c.rpcErr)
	}

	return steps
}

var _ stats.Handler = (*GRPCTraceCollector)(nil)
