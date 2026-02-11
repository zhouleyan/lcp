package handler

import (
	"fmt"
	"net/http"
	"strings"

	"lcp.io/lcp/lib/httpserver/filters"
	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/rest"
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

	ws := new(rest.WebService)
	ws.Path("/apis/v1")
	ws.Route(ws.GET("/users").To(FakeHandle))
	ws.Route(ws.GET("/users/{userId}").To(FakeHandle))
	ws.Route(ws.POST("/users").To(FakeHandle))
	ws.Route(ws.GET("/users/{userId:[0-9]+}").To(FakeHandle))
	ws.Route(ws.DELETE("/users/{userId}").To(FakeHandle))
	ws.Route(ws.PUT("/users/{userId}").To(FakeHandle))

	a.GoRestfulContainer.Add(ws)
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

func FakeHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method)
	fmt.Println(r.URL.Path)
	for k, v := range r.Header {
		fmt.Printf("%s: %s\n", k, v)
	}

	params := rest.PathParams(r)
	for k, v := range params {
		fmt.Printf("%s: %s\n", k, v)
	}
	userID := rest.PathParam(r, "userId")
	fmt.Println(userID)
	bar := rest.QueryParams(r, "foo")
	fmt.Printf("Query Param foo: %s", bar)
	// TODO: Extract Body Parameters r.ParseForm()
	// TODO: Read Body
	// TODO: Response Write(json,xml,text)
}
