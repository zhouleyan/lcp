package http_get_file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"lcp.io/lcp/lib/ansible/modules/internal"
)

// ModuleHTTPGetFile downloads a file from HTTP/HTTPS to a local path.
//
// Args:
//
//	url:      download URL (required)
//	dest:     local save path (required)
//	username: basic auth username (optional)
//	password: basic auth password (optional)
//	token:    bearer token (optional)
//	timeout:  request timeout duration string (optional, default "30s")
//	headers:  custom headers map[string]any (optional)
func ModuleHTTPGetFile(ctx context.Context, opts internal.ExecOptions) (string, string, error) {
	urlStr := internal.StringArg(opts.Args, "url")
	dest := internal.StringArg(opts.Args, "dest")
	if urlStr == "" || dest == "" {
		return "", "", fmt.Errorf("http_get_file: url and dest are required")
	}

	// Validate URL scheme.
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", "", fmt.Errorf("http_get_file: invalid url: %w", err)
	}
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
		urlStr = parsedURL.String()
	} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", "", fmt.Errorf("http_get_file: unsupported url scheme %q, only http and https are supported", parsedURL.Scheme)
	}

	// Parse timeout.
	timeout := 30 * time.Second
	if t := internal.StringArg(opts.Args, "timeout"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	// Create HTTP client.
	client := &http.Client{Timeout: timeout}

	// Build request.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, http.NoBody)
	if err != nil {
		return "", "", fmt.Errorf("http_get_file: create request: %w", err)
	}

	// Add basic auth.
	username := internal.StringArg(opts.Args, "username")
	password := internal.StringArg(opts.Args, "password")
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Add bearer token.
	if token := internal.StringArg(opts.Args, "token"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Add custom headers.
	if headers, ok := opts.Args["headers"].(map[string]any); ok {
		for k, v := range headers {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}

	// Execute request.
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("http_get_file: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("http_get_file: server returned %d: %s", resp.StatusCode, string(body))
	}

	// Ensure parent directory exists.
	parentDir := filepath.Dir(dest)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", "", fmt.Errorf("http_get_file: create directory %s: %w", parentDir, err)
	}

	// Download to temp file then atomic rename.
	tmpFile, err := os.CreateTemp(parentDir, ".http_get_file_*.tmp")
	if err != nil {
		return "", "", fmt.Errorf("http_get_file: create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		// Clean up temp file on any error path.
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return "", "", fmt.Errorf("http_get_file: download to temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", "", fmt.Errorf("http_get_file: close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return "", "", fmt.Errorf("http_get_file: rename temp to dest: %w", err)
	}

	return fmt.Sprintf("downloaded %s -> %s", urlStr, dest), "", nil
}
