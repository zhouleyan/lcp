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

// maxRequestBodySize is the maximum allowed request body size (1 MB).
const maxRequestBodySize = 1 << 20

// Handle returns an http.HandlerFunc that:
//  1. Extracts path params and query params from the request
//  2. Reads request body (if present)
//  3. Calls fn
//  4. Writes the response with the given statusCode (or 204 if result is nil)
func Handle(ns runtime.NegotiatedSerializer, statusCode int, fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := mergeQueryParams(PathParams(req), req)

		var body []byte
		if req.Body != nil && req.ContentLength != 0 {
			req.Body = http.MaxBytesReader(w, req.Body, maxRequestBodySize)
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
		if fr, ok := result.(*FileResponse); ok {
			writeFileResponse(w, statusCode, fr)
			return
		}
		transformResponseObject(ns, req, w, statusCode, result)
	}
}

// HandleWithAPIVersion is like Handle but also sets the APIVersion on the result object.
func HandleWithAPIVersion(ns runtime.NegotiatedSerializer, statusCode int, fn HandlerFunc, apiVersion string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := mergeQueryParams(PathParams(req), req)

		var body []byte
		if req.Body != nil && req.ContentLength != 0 {
			req.Body = http.MaxBytesReader(w, req.Body, maxRequestBodySize)
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
		if fr, ok := result.(*FileResponse); ok {
			writeFileResponse(w, statusCode, fr)
			return
		}
		if tm := result.GetTypeMeta(); tm != nil {
			tm.APIVersion = apiVersion
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

// readBody reads the full request body with a size limit.
func readBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body)
	return io.ReadAll(io.LimitReader(req.Body, maxRequestBodySize+1))
}

// jsonUnmarshal is a thin wrapper to avoid importing encoding/json in installer.go.
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func errNoIDs() error {
	return fmt.Errorf("no ids provided")
}

// mergeQueryParams copies path params and adds query params that don't
// conflict with existing path params. This allows HandlerFunc to access
// query parameters (e.g. ?file=cert.pem) alongside path params.
func mergeQueryParams(pathParams map[string]string, req *http.Request) map[string]string {
	query := req.URL.Query()
	if len(query) == 0 {
		return pathParams
	}
	merged := make(map[string]string, len(pathParams)+len(query))
	for k, v := range pathParams {
		merged[k] = v
	}
	for k, vals := range query {
		if _, exists := merged[k]; !exists && len(vals) > 0 {
			merged[k] = vals[0]
		}
	}
	return merged
}
