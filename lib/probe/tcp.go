package probe

import (
	"context"
	"net"
	"time"
)

// TCP probes whether the given addr (host:port) is reachable via TCP.
func TCP(ctx context.Context, addr string, opts *Options) *Result {
	timeout := opts.timeout()

	start := time.Now()
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	duration := time.Since(start)

	if err != nil {
		return &Result{
			Success:  false,
			Duration: duration,
			Phase:    classifyError(err),
			Message:  err.Error(),
		}
	}
	conn.Close()

	return &Result{
		Success:  true,
		Duration: duration,
	}
}
