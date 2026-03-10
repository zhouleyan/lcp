package handler

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"lcp.io/lcp/lib/audit"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/oidc"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/rest/filters"
	"lcp.io/lcp/lib/runtime"
)

// APIServerConfig holds the configuration for creating an API server handler.
type APIServerConfig struct {
	Name         string
	OIDCProvider *oidc.Provider
	Authorizer   *filters.Authorizer // nil = no authorization
	AuditLogger  audit.Logger        // nil = no audit logging
}

// APIServerHandler holds the different http.Handlers used by the API server.
type APIServerHandler struct {
	FullHandlerChain   http.Handler
	GoRestfulContainer *rest.Container
	Director           http.Handler

	serializer runtime.NegotiatedSerializer
	groups     []*rest.APIGroupInfo
}

func NewAPIServerHandler(
	cfg APIServerConfig,
	groups ...*rest.APIGroupInfo,
) (*APIServerHandler, error) {
	container := rest.NewContainer()

	director := director{
		name:      cfg.Name,
		container: container,
	}
	a := &APIServerHandler{
		FullHandlerChain:   buildChain(director, cfg),
		GoRestfulContainer: container,
		Director:           director,
		serializer:         runtime.NewCodecFactory(),
		groups:             groups,
	}

	if err := a.InstallAPIs(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *APIServerHandler) InstallAPIs() error {
	logger.Infof("installing lcp-server APIs...")
	return rest.InstallAPIGroups(a.GoRestfulContainer, a.serializer, a.groups...)
}

// ServeHTTP makes it an http.Handler.
func (a *APIServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.FullHandlerChain.ServeHTTP(w, r)
}

// RootHandlerConfig holds the components needed by the top-level request router.
type RootHandlerConfig struct {
	APIHandler  *APIServerHandler
	OIDCMux     http.Handler
	OpenAPISpec []byte
	FrontendFS  fs.FS
}

// NewRootHandler creates the top-level request handler that routes between
// OIDC public endpoints, OpenAPI spec, API handler, and frontend static files.
func NewRootHandler(cfg RootHandlerConfig) func(http.ResponseWriter, *http.Request) bool {
	oidcMux := cfg.OIDCMux

	var staticHandler http.Handler
	if cfg.FrontendFS != nil {
		staticHandler = http.FileServer(http.FS(cfg.FrontendFS))
	}

	return func(w http.ResponseWriter, r *http.Request) bool {
		urlPath := r.URL.Path

		// Route OIDC endpoints to public mux (no auth middleware)
		if oidcMux != nil && (strings.HasPrefix(urlPath, "/.well-known/") || strings.HasPrefix(urlPath, "/oidc/")) {
			oidcMux.ServeHTTP(w, r)
			return true
		}

		// Serve OpenAPI spec (no auth)
		if urlPath == "/docs/openapi.json" && cfg.OpenAPISpec != nil {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(cfg.OpenAPISpec)
			return true
		}

		// API requests go through the API handler (with auth middleware)
		if strings.HasPrefix(urlPath, "/api/") {
			cfg.APIHandler.ServeHTTP(w, r)
			return true
		}

		// Serve frontend static files; fallback to index.html for SPA routes
		if staticHandler != nil {
			serveFrontend(w, r, cfg.FrontendFS, staticHandler)
		}
		return true
	}
}

type director struct {
	name      string
	container *rest.Container
}

func (d director) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path

	for _, ws := range d.container.RegisteredWebServices() {
		switch {
		case ws.RootPath() == "/apis":
			if p == "/apis" || p == "/apis/" {
				d.container.Dispatch(w, r)
				return
			}
		case strings.HasPrefix(p, ws.RootPath()):
			if len(p) == len(ws.RootPath()) || p[len(ws.RootPath())] == '/' {
				d.container.Dispatch(w, r)
				return
			}
		}
	}
}

// buildChain assembles the middleware chain from APIServerConfig.
// Order (innermost → outermost): director → WithAuthorization → WithAudit → WithRequestInfo → WithAuthentication → WithRequestLog
func buildChain(apiHandler http.Handler, cfg APIServerConfig) http.Handler {
	handler := apiHandler
	if authz := cfg.Authorizer; authz != nil {
		if authz.Lookup != nil && authz.Checker != nil {
			handler = filters.WithAuthorization(authz.Lookup, authz.Checker)(handler)
		}
	}
	if cfg.AuditLogger != nil {
		handler = filters.WithAudit(cfg.AuditLogger)(handler)
	}
	if authz := cfg.Authorizer; authz != nil {
		if authz.NSResolver != nil {
			handler = filters.WithRequestInfo(authz.NSResolver)(handler)
		}
	}
	if cfg.OIDCProvider != nil {
		handler = filters.WithAuthentication(cfg.OIDCProvider)(handler)
	}
	handler = filters.WithRequestLog(handler)
	return handler
}

// serveFrontend serves static files from the embedded frontend.
// If the requested file exists, it is served directly.
// Otherwise, index.html is served to support SPA client-side routing.
func serveFrontend(w http.ResponseWriter, r *http.Request, distFS fs.FS, staticHandler http.Handler) {
	filePath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if filePath == "" {
		filePath = "index.html"
	}

	if f, err := distFS.Open(filePath); err == nil {
		_ = f.Close()
		staticHandler.ServeHTTP(w, r)
		return
	}

	// SPA fallback: serve index.html for all non-file routes
	r.URL.Path = "/"
	staticHandler.ServeHTTP(w, r)
}
