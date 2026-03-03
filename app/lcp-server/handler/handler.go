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

	groups []*rest.APIGroupInfo
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

	scope := &rest.RequestScope{Serializer: runtime.NewCodecFactory()}

	// Install module-based API groups
	for _, group := range a.groups {
		prefix := "/apis/" + group.Version
		ws := a.findOrCreateWebService(prefix)
		installer := rest.NewAPIInstaller(group, ws, scope)
		installer.Install()
	}

	// Pod (legacy mock, not migrated)
	ws := a.findOrCreateWebService("/apis/v1")
	p := NewPod()
	ws.Route(ws.GET("/pods").To(rest.Handle(scope, http.StatusOK, p.Get)))
	logger.Infof("  GET    /apis/v1/pods (legacy)")

	return nil
}

// findOrCreateWebService returns the existing WebService for the given root path,
// or creates and registers a new one.
func (a *APIServerHandler) findOrCreateWebService(rootPath string) *rest.WebService {
	for _, ws := range a.GoRestfulContainer.RegisteredWebServices() {
		if ws.RootPath() == rootPath {
			return ws
		}
	}
	ws := new(rest.WebService)
	ws.Path(rootPath).
		Produces("application/json", "application/yaml").
		Consumes("application/json", "application/yaml")
	a.GoRestfulContainer.Add(ws)
	return ws
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
	logger.Infof("Directing: %s %s", d.name, path)

	for _, ws := range d.container.RegisteredWebServices() {
		switch {
		case ws.RootPath() == "/apis":
			if path == "/apis" || path == "/apis/" {
				logger.Infof("%v: %v %q satisfied by rest with web service %v", d.name, r.Method, path, ws.RootPath())
				d.container.Dispatch(w, r)
				return
			}
		case strings.HasPrefix(path, ws.RootPath()):
			if len(path) == len(ws.RootPath()) || path[len(ws.RootPath())] == '/' {
				logger.Infof("%v: %v %q satisfied by rest with web service %v", d.name, r.Method, path, ws.RootPath())
				d.container.Dispatch(w, r)
				return
			}
		}
	}
}

func DefaultChainBuilder(apiHandler http.Handler) http.Handler {
	handler := apiHandler
	handler = filters.WithRequestInfo(handler)
	return handler
}
