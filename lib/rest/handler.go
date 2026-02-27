package rest

import (
	"context"
	"io"
	"net/http"

	"lcp.io/lcp/lib/runtime"
)

// RequestScope encapsulates common fields across all RESTful handler methods
type RequestScope struct {
	Name       string
	Serializer runtime.NegotiatedSerializer
}

func (scope *RequestScope) err(err error, w http.ResponseWriter, r *http.Request) {
	ErrorNegotiated(w, r, scope.Serializer, err)
}

type getterFunc func(ctx context.Context, name string) (runtime.Object, error)

// CreatorFunc is the function signature for handlers that create a resource from a request body.
type CreatorFunc func(ctx context.Context, body []byte) (runtime.Object, error)

func GetResource(scope *RequestScope, getter getterFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		name := scope.Name

		result, err := getter(ctx, name)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		transformResponseObject(scope, req, w, http.StatusOK, result)
	}
}

func transformResponseObject(
	scope *RequestScope,
	req *http.Request,
	w http.ResponseWriter,
	statusCode int,
	result runtime.Object,
) {
	WriteObjectNegotiated(scope.Serializer, w, req, statusCode, result)
}

// CreateResource returns an http.HandlerFunc that reads the request body and
// delegates to the given CreatorFunc to create a resource.
func CreateResource(scope *RequestScope, creator CreatorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		defer req.Body.Close()
		result, err := creator(ctx, body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		transformResponseObject(scope, req, w, http.StatusCreated, result)
	}
}

// GetResourceWithID returns an http.HandlerFunc that extracts a path parameter
// by name and delegates to the getter.
func GetResourceWithID(scope *RequestScope, paramName string, getter getterFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		id := PathParam(req, paramName)
		result, err := getter(ctx, id)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		transformResponseObject(scope, req, w, http.StatusOK, result)
	}
}

// CreateResourceWithID returns an http.HandlerFunc for nested resource creation
// (e.g. POST /namespaces/{id}/members). It extracts the parent ID from the path
// and uses creatorFactory to build a CreatorFunc scoped to that parent.
func CreateResourceWithID(scope *RequestScope, paramName string, creatorFactory func(id string) CreatorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		id := PathParam(req, paramName)
		body, err := io.ReadAll(req.Body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		defer req.Body.Close()
		result, err := creatorFactory(id)(ctx, body)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		transformResponseObject(scope, req, w, http.StatusCreated, result)
	}
}
