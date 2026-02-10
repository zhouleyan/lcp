package rest

import (
	"net/http"
	"strings"
)

// RouteFunction is a function that can be called when a route is matched
type RouteFunction func(w http.ResponseWriter, r *http.Request)

// Route binds an HTTP Method, Path, Consumes combination to a RouteFunction
type Route struct {
	Method   string
	Path     string // webservice root path + described path
	Produces []string
	Consumes []string
	Function RouteFunction

	// cached values for dispatching
	relativePath string
	pathParts    []string
	pathExpr     *pathExpression // cached compilation of relativePath as RegExp

	// indicate route path has custom verb
	hasCustomVerb bool

	paramCount  int
	staticCount int
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

// Return whether this Route can consume content with a type specified by mimeTypes (can be empty).
// If the route does not specify Consumes then return type (*/*)
// If no content type is set then return true for GET,HEAD,OPTIONS,DELETE and TRACE
func (r *Route) matchesContentType(mimeTypes string) bool {
	if len(r.Consumes) == 0 {
		// did not specify what it can consume; any media type ("*/*") is assumed
		return true
	}

	if len(mimeTypes) == 0 {
		// request did not specify its Content-Type
		// idempotent methods with (most-likely or guaranteed) empty content match missing Content-Type
		m := r.Method
		if m == "GET" || m == "HEAD" || m == "OPTIONS" || m == "DELETE" || m == "TRACE" {
			return true
		}
	}
	// proceed with default
	mimeTypes = MIME_OCTET

	remaining := mimeTypes
	for {
		var mimeType string
		mimeType, remaining = parseNextMimeType(remaining)
		
		for _, consumableType := range r.Consumes {
			if consumableType == "*/*" || consumableType == mimeType {
				return true
			}
		}
		if len(remaining) == 0 {
			return false
		}
	}
}

// Return whether the mimeType matches to what this Route can produce
func (r *Route) matchesAccept(mimeTypesWithQuality string) bool {
	remaining := mimeTypesWithQuality
	for {
		var mimeType string
		mimeType, remaining = parseNextMimeType(remaining)

		if mimeType == "*/*" {
			return true
		}
		for _, producibleType := range r.Produces {
			if producibleType == "*/*" || producibleType == mimeType {
				return true
			}
		}
		if len(remaining) == 0 {
			return false
		}
	}
}

func stringTrimSpaceCutset(r rune) bool {
	return r == ' '
}

func parseNextMimeType(remaining string) (mimeType string, nextRemaining string) {
	if end := strings.Index(remaining, ","); end == -1 {
		mimeType, nextRemaining = remaining, ""
	} else {
		mimeType, nextRemaining = remaining[:end], remaining[end+1:]
	}

	if quality := strings.Index(mimeType, ";"); quality != -1 {
		mimeType = mimeType[:quality]
	}

	mimeType = strings.TrimFunc(mimeType, stringTrimSpaceCutset)

	return mimeType, nextRemaining
}
