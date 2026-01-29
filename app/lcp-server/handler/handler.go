package handler

import (
	"fmt"
	"net/http"

	"lcp.io/lcp/app/lcp-server/rest"
	"lcp.io/lcp/lib/httpserver/filters"
	"lcp.io/lcp/lib/logger"
)

// APIServerHandler holds the different http.Handlers used by the API server
type APIServerHandler struct {
	// FullHandlerChain is the one that is eventually served with. It should
	FullHandlerChain http.Handler

	// InstallAPIs use this
	GoRestfulContainer *rest.Container

	// Director is here so that we can properly handle fall through and proxy cases
	Director http.Handler
}

func NewAPIServerHandler(name string) (*APIServerHandler, error) {
	// create REST API container
	container := rest.NewContainer()

	director := director{
		name:      name,
		container: container,
	}
	a := &APIServerHandler{
		FullHandlerChain:   DefaultChainBuilder(director),
		GoRestfulContainer: container,
		Director:           director,
	}

	// Install APIs
	if err := a.InstallAPIs(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *APIServerHandler) RequestHandler(w http.ResponseWriter, r *http.Request) bool {
	a.ServeHTTP(w, r)
	return true
}

func (a *APIServerHandler) InstallAPIs() error {
	logger.Infof("installing lcp-server APIs...")
	return nil
}

// ChainBuilderFn is used to wrap the API handler using provided handler chain
// It is normally used to apply filtering like authentication and authorization
type ChainBuilderFn func(apiHandler http.Handler) http.Handler

// ServerHTTP makes it an http.Handler
func (a *APIServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "APIServerHandler")
	a.FullHandlerChain.ServeHTTP(w, r)
}

type director struct {
	name      string
	container *rest.Container
}

func (d director) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	logger.Infof("Directing: %s %s", d.name, path)
}

func DefaultChainBuilder(apiHandler http.Handler) http.Handler {
	handler := apiHandler

	// WithRequestInfo
	handler = filters.WithRequestInfo(handler)
	return handler
}
