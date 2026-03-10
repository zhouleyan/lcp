package filters

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"lcp.io/lcp/lib/audit"
	"lcp.io/lcp/lib/oidc"
)

// maxBodyCapture is the maximum request body size captured for audit (64 KB).
const maxBodyCapture = 64 * 1024

// statusWriter wraps http.ResponseWriter to capture the status code
// and optionally the response body (for extracting created resource IDs).
type statusWriter struct {
	http.ResponseWriter
	code    int
	buf     bytes.Buffer
	capture bool
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.code = code
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if sw.capture && sw.buf.Len() < maxBodyCapture {
		sw.buf.Write(b)
	}
	return sw.ResponseWriter.Write(b)
}

// WithAudit returns middleware that logs API write operations (POST/PUT/PATCH/DELETE).
func WithAudit(logger audit.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isAuditableRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Capture request body before downstream consumes it.
			// Read the full body (downstream json.Decode would do the same),
			// restore it for the handler, and only truncate when writing to audit detail.
			var bodyDetail json.RawMessage
			if r.Body != nil && hasRequestBody(r.Method) {
				buf, err := io.ReadAll(r.Body)
				if err == nil && len(buf) > 0 {
					// Only store if it's valid JSON.
					if json.Valid(buf) {
						if len(buf) > maxBodyCapture {
							bodyDetail = buf[:maxBodyCapture]
						} else {
							bodyDetail = buf
						}
					}
				}
				// Restore the full body so downstream handlers can read it.
				r.Body = io.NopCloser(bytes.NewReader(buf))
			}

			_, _, verb := ResolveResourceAndVerb(r.Method, r.URL.Path)
			sw := &statusWriter{
				ResponseWriter: w,
				code:           http.StatusOK,
				capture:        verb == "create",
			}
			start := time.Now()

			next.ServeHTTP(sw, r)

			duration := time.Since(start)
			event := buildAuditEvent(r, sw.code, duration)
			event.Detail = bodyDetail
			if verb == "create" && sw.code >= 200 && sw.code < 300 {
				event.ResourceID = extractResourceIDFromResponse(sw.buf.Bytes())
			}
			logger.Log(event)
		})
	}
}

// isAuditableRequest returns true for API write operations.
func isAuditableRequest(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		return false
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// hasRequestBody returns true for HTTP methods that typically carry a request body.
func hasRequestBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	}
	return false
}

func buildAuditEvent(r *http.Request, statusCode int, duration time.Duration) audit.Event {
	module, resourceChain, verb := ResolveResourceAndVerb(r.Method, r.URL.Path)
	reqInfo := RequestInfoFromContext(r.Context())

	event := audit.Event{
		EventType:    "api_operation",
		Action:       verb,
		ResourceType: resourceChain,
		Module:       module,
		Scope:        reqInfo.Scope,
		HTTPMethod:   r.Method,
		HTTPPath:     r.URL.Path,
		StatusCode:   statusCode,
		ClientIP:     audit.ClientIP(r),
		UserAgent:    r.UserAgent(),
		DurationMs:   int(duration.Milliseconds()),
		Success:      statusCode >= 200 && statusCode < 400,
		CreatedAt:    time.Now(),
	}

	if reqInfo.WorkspaceID > 0 {
		wsID := reqInfo.WorkspaceID
		event.WorkspaceID = &wsID
	}
	if reqInfo.NamespaceID > 0 {
		nsID := reqInfo.NamespaceID
		event.NamespaceID = &nsID
	}

	if userID, ok := oidc.UserIDFromContext(r.Context()); ok {
		event.UserID = &userID
	}
	event.Username = oidc.UsernameFromContext(r.Context())
	event.ResourceID = extractResourceID(r.URL.Path, verb)

	return event
}

// extractResourceIDFromResponse parses the JSON response body to find the created resource ID.
func extractResourceIDFromResponse(body []byte) string {
	var resp struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if json.Unmarshal(body, &resp) == nil && resp.Metadata.ID != "" {
		return resp.Metadata.ID
	}
	return ""
}

// extractResourceID extracts the target resource ID from the URL path for single-resource operations.
func extractResourceID(path, verb string) string {
	switch verb {
	case "get", "update", "patch", "delete":
		// These verbs target a specific resource by ID
	default:
		return ""
	}

	segments := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
	// segments: module/version/resource/id[/sub/id...]
	if len(segments) < 4 {
		return ""
	}

	// Return the last numeric segment
	for i := len(segments) - 1; i >= 3; i-- {
		seg := segments[i]
		// Handle custom verb: "123:workspaces"
		if idx := strings.IndexByte(seg, ':'); idx > 0 {
			seg = seg[:idx]
		}
		// Check if numeric
		isNumeric := true
		for _, c := range seg {
			if c < '0' || c > '9' {
				isNumeric = false
				break
			}
		}
		if isNumeric && seg != "" {
			return seg
		}
	}
	return ""
}
