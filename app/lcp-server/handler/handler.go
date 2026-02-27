package handler

import (
	"net/http"
	"strings"

	"lcp.io/lcp/lib/httpserver/filters"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/service"
)

// APIServerHandler holds the different http.Handlers used by the API server
type APIServerHandler struct {
	// FullHandlerChain is the one that is eventually served with. It should
	FullHandlerChain http.Handler

	// InstallAPIs use this
	GoRestfulContainer *rest.Container

	// Director is here so that we can properly handle fall through and proxy cases
	Director http.Handler

	svc *service.Service
}

func NewAPIServerHandler(name string, svc *service.Service) (*APIServerHandler, error) {
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
		svc:                svc,
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

	codec := runtime.NewCodecFactory()

	ws := new(rest.WebService)
	ws.Path("/apis/v1").
		Produces("application/json", "application/yaml").
		Consumes("application/json", "application/yaml")

	// User routes
	u := newUserHandler(a.svc)
	userScope := &rest.RequestScope{Serializer: codec}
	ws.Route(ws.POST("/users").To(rest.CreateResource(userScope, u.Create)))
	ws.Route(ws.GET("/users/{userId}").To(rest.GetResourceWithID(userScope, "userId", u.Get)))

	// Namespace routes
	ns := newNamespaceHandler(a.svc)
	nsScope := &rest.RequestScope{Serializer: codec}
	ws.Route(ws.POST("/namespaces").To(rest.CreateResource(nsScope, ns.Create)))
	ws.Route(ws.GET("/namespaces/{namespaceId}").To(rest.GetResourceWithID(nsScope, "namespaceId", ns.Get)))
	ws.Route(ws.POST("/namespaces/{namespaceId}/members").To(
		rest.CreateResourceWithID(nsScope, "namespaceId", ns.AddMemberCreator),
	))

	// Keep existing pod route
	p := NewPod()
	podScope := &rest.RequestScope{
		Name:       "pod",
		Serializer: codec,
	}
	ws.Route(ws.GET("/pods").To(rest.GetResource(podScope, p.Get)))

	a.GoRestfulContainer.Add(ws)
	return nil
}

// ChainBuilderFn is used to wrap the API handler using provided handler chain
// It is normally used to apply filtering like authentication and authorization
type ChainBuilderFn func(apiHandler http.Handler) http.Handler

// ServerHTTP makes it an http.Handler
func (a *APIServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.FullHandlerChain.ServeHTTP(w, r)
}

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
			// ensure an exact match or a path boundary match
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

	// WithRequestInfo
	handler = filters.WithRequestInfo(handler)
	return handler
}
