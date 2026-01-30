package rest

import "strings"

// Route binds a HTTP Method, Path, Consumes combination to a RouteFunction
type Route struct {
	Method string
	Path   string // webservice root path + described path

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
	return strings.Split(strings.TrimLeft(path, "/"), "/")
}
