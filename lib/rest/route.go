package rest

import (
	"net/http"
	"strings"
)

// RouteFunction is a function that can be called when a route is matched
type RouteFunction func(w http.ResponseWriter, r *http.Request)

// Route binds a HTTP Method, Path, Consumes combination to a RouteFunction
type Route struct {
	Method   string
	Path     string // webservice root path + described path
	Function RouteFunction

	// cached values for dispatching
	relativePath string
	pathParts    []string
	pathExpr     *pathExpression // cached compilation of relativePath as RegExp

	// indicate route path has custom verb
	hasCustomVerb bool
}

func tokenizePath(path string) []string {
	if "/" == path {
		return nil
	}
	return strings.Split(strings.Trim(path, "/"), "/")
}

func (r *Route) postBuild() {
	r.pathParts = tokenizePath(r.Path)
	r.hasCustomVerb = hasCustomVerb(r.Path)
}

// for debugging
func (r *Route) String() string {
	return r.Method + " " + r.Path
}
