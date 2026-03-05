package handler

import (
	"net/http"
	"strings"

	"lcp.io/lcp/lib/httpserver/filters"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
)

// APIServerHandler holds the different http.Handlers used by the API server.
type APIServerHandler struct {
	FullHandlerChain   http.Handler
	GoRestfulContainer *rest.Container
	Director           http.Handler

	serializer runtime.NegotiatedSerializer
	groups     []*rest.APIGroupInfo
}

func NewAPIServerHandler(name string, groups ...*rest.APIGroupInfo) (*APIServerHandler, error) {
	container := rest.NewContainer()

	director := director{
		name:      name,
		container: container,
	}
	a := &APIServerHandler{
		FullHandlerChain:   DefaultChainBuilder(director),
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

func (a *APIServerHandler) RequestHandler(w http.ResponseWriter, r *http.Request) bool {
	a.ServeHTTP(w, r)
	return true
}

func (a *APIServerHandler) InstallAPIs() error {
	logger.Infof("installing lcp-server APIs...")
	return rest.InstallAPIGroups(a.GoRestfulContainer, a.serializer, a.groups...)
}

// ServeHTTP makes it an http.Handler.
func (a *APIServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.FullHandlerChain.ServeHTTP(w, r)
}

// ChainBuilderFn is used to wrap the API handler using provided handler chain.
type ChainBuilderFn func(apiHandler http.Handler) http.Handler

type director struct {
	name      string
	container *rest.Container
}

func (d director) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	for _, ws := range d.container.RegisteredWebServices() {
		switch {
		case ws.RootPath() == "/apis":
			if path == "/apis" || path == "/apis/" {
				d.container.Dispatch(w, r)
				return
			}
		case strings.HasPrefix(path, ws.RootPath()):
			if len(path) == len(ws.RootPath()) || path[len(ws.RootPath())] == '/' {
				d.container.Dispatch(w, r)
				return
			}
		}
	}
}

func DefaultChainBuilder(apiHandler http.Handler) http.Handler {
	handler := apiHandler
	handler = filters.WithRequestLog(handler)
	return handler
}
