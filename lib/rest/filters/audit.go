package filters

import (
	"net/http"
	"strings"
	"time"

	"lcp.io/lcp/lib/audit"
	"lcp.io/lcp/lib/oidc"
)

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.code = code
	sw.ResponseWriter.WriteHeader(code)
}

// WithAudit returns middleware that logs API write operations (POST/PUT/PATCH/DELETE).
func WithAudit(logger audit.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isAuditableRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(sw, r)

			duration := time.Since(start)
			event := buildAuditEvent(r, sw.code, duration)
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
	event.ResourceID = extractResourceID(r.URL.Path, verb)

	return event
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
