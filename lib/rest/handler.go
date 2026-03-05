package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"lcp.io/lcp/lib/runtime"
)

// handleError writes an error response using content negotiation.
func handleError(ns runtime.NegotiatedSerializer, err error, w http.ResponseWriter, r *http.Request) {
	ErrorNegotiated(w, r, ns, err)
}

// HandlerFunc is the unified function signature for all request handlers.
type HandlerFunc func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error)

// Handle returns an http.HandlerFunc that:
//  1. Extracts path params from context
//  2. Reads request body (if present)
//  3. Calls fn
//  4. Writes the response with the given statusCode (or 204 if result is nil)
func Handle(ns runtime.NegotiatedSerializer, statusCode int, fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := PathParams(req)

		var body []byte
		if req.Body != nil && req.ContentLength != 0 {
			var err error
			body, err = io.ReadAll(req.Body)
			if err != nil {
				handleError(ns, err, w, req)
				return
			}
			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(req.Body)
		}

		result, err := fn(ctx, params, body)
		if err != nil {
			handleError(ns, err, w, req)
			return
		}
		if result == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		transformResponseObject(ns, req, w, statusCode, result)
	}
}

func transformResponseObject(
	ns runtime.NegotiatedSerializer,
	req *http.Request,
	w http.ResponseWriter,
	statusCode int,
	result runtime.Object,
) {
	WriteObjectNegotiated(ns, w, req, statusCode, result)
}

// readBody reads the full request body.
func readBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body)
	return io.ReadAll(req.Body)
}

// jsonUnmarshal is a thin wrapper to avoid importing encoding/json in installer.go.
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func errNoIDs() error {
	return fmt.Errorf("no ids provided")
}
