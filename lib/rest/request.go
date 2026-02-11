package rest

import (
	"context"
	"net/http"
)

type key int

const (
	PathParamsKey key = iota
)

// WithPathParams add path params to request context (r = WithPathParams(r, pathParams))
func WithPathParams(r *http.Request, pathParams map[string]string) *http.Request {
	ctx := context.WithValue(r.Context(), PathParamsKey, pathParams)
	return r.WithContext(ctx)
}

func PathParams(r *http.Request) map[string]string {
	return r.Context().Value(PathParamsKey).(map[string]string)
}

func PathParam(r *http.Request, name string) string {
	return r.Context().Value(PathParamsKey).(map[string]string)[name]
}

// QueryParams returns all the query parameters values by name
func QueryParams(r *http.Request, name string) []string {
	return r.URL.Query()[name]
}

// QueryParam returns the (first) Query parameter value by its name
func QueryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

// BodyParam returns the body of the request
// (once for typically a POST or a PUT)
// and returns the value of the given name or an error
func BodyParam(r *http.Request, name string) (string, error) {
	// Parse the form data
	if err := r.ParseForm(); err != nil {
		return "", err
	}

	// Return the value of the given name
	return r.PostFormValue(name), nil
}

func HeaderParam(r *http.Request, name string) string {
	return r.Header.Get(name)
}
