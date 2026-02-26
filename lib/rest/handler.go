package rest

import (
	"context"
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
