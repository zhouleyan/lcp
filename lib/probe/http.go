package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

// HTTP probes whether the given URL is reachable and returns a non-5xx status.
func HTTP(ctx context.Context, url string, opts *Options) *Result {
	timeout := opts.timeout()

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout: timeout,
			}).DialContext,
		},
		// Do not follow redirects — just record the first response.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &Result{
			Success: false,
			Phase:   PhaseTCP,
			Message: fmt.Sprintf("invalid request: %v", err),
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return &Result{
			Success:  false,
			Duration: duration,
			Phase:    classifyError(err),
			Message:  err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return &Result{
			Success:    false,
			Duration:   duration,
			Phase:      PhaseHTTP,
			Message:    fmt.Sprintf("server error: %s", resp.Status),
			StatusCode: resp.StatusCode,
		}
	}

	return &Result{
		Success:    true,
		Duration:   duration,
		StatusCode: resp.StatusCode,
	}
}
