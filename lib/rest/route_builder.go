package rest

import (
	"strings"

	"lcp.io/lcp/lib/logger"
)

// RouteBuilder is a helper to construct Route
type RouteBuilder struct {
	rootPath    string
	currentPath string
	produces    []string
	consumes    []string
	httpMethod  string
	function    RouteFunction
}

// To bind the route to a function
// If this route is matched with the incoming HTTP request then call this function with the ResponseWriter, *Request pair
// Required
func (b *RouteBuilder) To(function RouteFunction) *RouteBuilder {
	b.function = function
	return b
}

// Method specifies what HTTP method to match
// Required
func (b *RouteBuilder) Method(method string) *RouteBuilder {
	b.httpMethod = method
	return b
}

func (b *RouteBuilder) servicePath(path string) *RouteBuilder {
	b.rootPath = path
	return b
}

// Path specifies the relative (w.r.t WebService root path) URL path to match
func (b *RouteBuilder) Path(path string) *RouteBuilder {
	b.currentPath = path
	return b
}

func (b *RouteBuilder) copyDefaults(rootProduces, rootConsumes []string) {
	if len(b.produces) == 0 {
		b.produces = rootProduces
	}
	if len(b.consumes) == 0 {
		b.consumes = rootConsumes
	}
}

// Produces specifies what MIME types can be produced ; the matched one will appear in the Content-Type Http header
func (b *RouteBuilder) Produces(mimeTypes ...string) *RouteBuilder {
	b.produces = mimeTypes
	return b
}

// Consumes specifies what MIME types can be consumed ; the Accept Http header must match any of these
func (b *RouteBuilder) Consumes(mimeTypes ...string) *RouteBuilder {
	b.consumes = mimeTypes
	return b
}

// Build creates a new Route using the specification details collected by the RouteBuilder
func (b *RouteBuilder) Build() Route {
	pathExpr, err := newPathExpression(b.currentPath)
	if err != nil {
		logger.Fatalf("invalid path: %s, error: %v", b.currentPath, err)
	}
	if b.function == nil {
		logger.Fatalf("no function specified for route: %s", b.currentPath)
	}
	route := Route{
		Method:       b.httpMethod,
		Path:         concatPath(b.rootPath, b.currentPath),
		Function:     b.function,
		relativePath: b.currentPath,
		pathExpr:     pathExpr,
	}
	route.postBuild()
	return route
}

// merge two paths using the current (package global) merge path strategy.
func concatPath(rootPath, routePath string) string {
	return strings.TrimRight(rootPath, "/") + "/" + strings.TrimLeft(routePath, "/")
}
