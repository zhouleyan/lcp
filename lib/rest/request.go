package rest

import (
	"context"
	"fmt"
	"net/http"

	"lcp.io/lcp/lib/runtime"
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

// DecodeBody decodes the request body into a runtime.Object using the
// Content-Type header to select the appropriate serializer.
//
// Parameters:
//   - ns: The negotiated serializer that provides all supported formats
//   - req: The HTTP request (used to read Content-Type header)
//   - body: The raw request body bytes
//   - into: Optional object to decode into (nil to create a new object)
//
// Returns the decoded object or an error if:
//   - The Content-Type is not supported
//   - Deserialization fails
func DecodeBody(
	ns runtime.NegotiatedSerializer,
	req *http.Request,
	body []byte,
	into runtime.Object,
) (runtime.Object, error) {
	// Get the Content-Type from the request header
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json" // default to JSON
	}

	// Find the serializer for this media type
	supportedTypes := ns.SupportedMediaTypes()
	info, ok := runtime.SerializerInfoForMediaType(supportedTypes, contentType)
	if !ok {
		return nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
	}

	// Decode the body using the matched serializer
	obj, err := info.Serializer.Decode(body, into)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request body: %w", err)
	}

	return obj, nil
}
