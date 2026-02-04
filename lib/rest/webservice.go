package rest

import (
	"net/http"
	"sync"

	"lcp.io/lcp/lib/logger"
)

// WebService holds a collection of Route values that bind a HTTP Method + URL Path to a function
// ws := new(WebService)
// ws.Path("/api/v1")
// ws.Consumes("*/*")
// ws.Produces("*/x")
// ws.APIVersion("v1")
// ws.Route(ws.GET("/").To(handle))
type WebService struct {
	rootPath   string
	pathExpr   *pathExpression // cached compilation of rootPath as RegExp
	routes     []Route
	produces   []string
	consumes   []string
	apiVersion string

	// protects `routes` if dynamic routes
	routesLock sync.RWMutex
}

// RootPath returns the RootPath associated with this WebService. Default "/"
func (w *WebService) RootPath() string {
	return w.rootPath
}

// Path specifies the root URL template path of the WebService
// All Routes will be relative to this path
func (w *WebService) Path(root string) *WebService {
	w.rootPath = root
	if len(w.rootPath) == 0 {
		w.rootPath = "/"
	}
	w.compilePathExpression()
	return w
}

// Route creates a new Route using the RouteBuilder and add to the ordered list of Routes
func (w *WebService) Route(builder *RouteBuilder) *WebService {
	w.routesLock.Lock()
	defer w.routesLock.Unlock()
	builder.copyDefaults(w.produces, w.consumes)
	w.routes = append(w.routes, builder.Build())
	return w
}

// Routes returns the Routes associated with this WebService
func (w *WebService) Routes() []Route {
	// Make a copy of the array to prevent concurrency problems
	w.routesLock.RLock()
	defer w.routesLock.RUnlock()
	result := make([]Route, len(w.routes))
	for ix := range w.routes {
		result[ix] = w.routes[ix]
	}
	return result
}

// RemoveRoute removes the specified route, looks for something that matches 'path' and 'method'
func (w *WebService) RemoveRoute(path, method string) error {
	w.routesLock.Lock()
	defer w.routesLock.Unlock()
	var newRoutes []Route
	for _, route := range w.routes {
		if route.Method == method && route.Path == path {
			continue
		}
		newRoutes = append(newRoutes, route)
	}
	w.routes = newRoutes
	return nil
}

// Produces specifies that this WebService can produce one or more MIME types.
// Http requests must have one of these values set for the Accept header.
func (w *WebService) Produces(contentTypes ...string) *WebService {
	w.produces = contentTypes
	return w
}

// Consumes specifies that this WebService can consume one or more MIME types.
// Http requests must have one of these values set for the Content-Type header.
func (w *WebService) Consumes(accepts ...string) *WebService {
	w.consumes = accepts
	return w
}

// SetAPIVersion sets the API version for documentation purposes.
func (w *WebService) SetAPIVersion(apiVersion string) *WebService {
	w.apiVersion = apiVersion
	return w
}

// Version returns the API version for documentation purposes.
func (w *WebService) Version() string { return w.apiVersion }

// compilePathExpression ensures that the path is compiled into a RegEx for those Routes that need it
func (w *WebService) compilePathExpression() {
	compiled, err := newPathExpression(w.rootPath)
	if err != nil {
		logger.Fatalf("invalid path: %s, %v", w.rootPath, err)
	}
	w.pathExpr = compiled
}

/*
	Convenience methods
*/

func (w *WebService) GET(subPath string) *RouteBuilder {
	return new(RouteBuilder).servicePath(w.rootPath).Method(http.MethodGet).Path(subPath)
}

func (w *WebService) POST(subPath string) *RouteBuilder {
	return new(RouteBuilder).servicePath(w.rootPath).Method(http.MethodPost).Path(subPath)
}

func (w *WebService) PUT(subPath string) *RouteBuilder {
	return new(RouteBuilder).servicePath(w.rootPath).Method(http.MethodPut).Path(subPath)
}

func (w *WebService) PATCH(subPath string) *RouteBuilder {
	return new(RouteBuilder).servicePath(w.rootPath).Method(http.MethodPatch).Path(subPath)
}

func (w *WebService) DELETE(subPath string) *RouteBuilder {
	return new(RouteBuilder).servicePath(w.rootPath).Method(http.MethodDelete).Path(subPath)
}

func (w *WebService) OPTIONS(subPath string) *RouteBuilder {
	return new(RouteBuilder).servicePath(w.rootPath).Method(http.MethodOptions).Path(subPath)
}
