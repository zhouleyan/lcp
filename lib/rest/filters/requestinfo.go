package filters

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

type contextKey int

const requestInfoKey contextKey = iota

// RequestInfo holds scope information extracted from the URL path.
type RequestInfo struct {
	WorkspaceID int64
	NamespaceID int64
	Scope       string // "platform" | "workspace" | "namespace"
}

// NamespaceResolver resolves the parent workspace ID for a given namespace ID.
type NamespaceResolver interface {
	GetWorkspaceID(ctx context.Context, namespaceID int64) (int64, bool)
}

// WithRequestInfo extracts workspaceID/namespaceID/scope from URL and stores in context.
func WithRequestInfo(nsResolver NamespaceResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := resolveRequestInfo(r.Context(), r.URL.Path, nsResolver)
			ctx := context.WithValue(r.Context(), requestInfoKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestInfoFromContext retrieves the RequestInfo from context.
func RequestInfoFromContext(ctx context.Context) *RequestInfo {
	if info, ok := ctx.Value(requestInfoKey).(*RequestInfo); ok {
		return info
	}
	return &RequestInfo{Scope: "platform"}
}

func resolveRequestInfo(ctx context.Context, path string, nsResolver NamespaceResolver) *RequestInfo {
	segments := strings.Split(path, "/")
	info := &RequestInfo{Scope: "platform"}

	for i := 0; i < len(segments)-1; i++ {
		seg := segments[i]
		nextSeg := segments[i+1]

		// Handle custom verb in next segment (e.g. "123:workspaces")
		if idx := strings.IndexByte(nextSeg, ':'); idx > 0 {
			nextSeg = nextSeg[:idx]
		}

		switch seg {
		case "workspaces":
			if id, err := strconv.ParseInt(nextSeg, 10, 64); err == nil {
				info.WorkspaceID = id
			}
		case "namespaces":
			if id, err := strconv.ParseInt(nextSeg, 10, 64); err == nil {
				info.NamespaceID = id
			}
		}
	}

	// If we have a namespaceID but no workspaceID, resolve via NamespaceResolver
	if info.NamespaceID > 0 && info.WorkspaceID == 0 && nsResolver != nil {
		if wsID, ok := nsResolver.GetWorkspaceID(ctx, info.NamespaceID); ok {
			info.WorkspaceID = wsID
		}
	}

	if info.NamespaceID > 0 {
		info.Scope = "namespace"
	} else if info.WorkspaceID > 0 {
		info.Scope = "workspace"
	}

	return info
}
