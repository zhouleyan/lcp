package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"lcp.io/lcp/lib/runtime"
)

// WriteObjectNegotiated serializes obj using a content-type negotiated from the
// request's Accept header, and writes it to the HTTP response
//
// This is the top-level entry point
//
//  1. Negotiate the output media type from the Accept header
//  2. If negotiation fails for an error response, fall back to raw JSON
//  3. Create the encoder
//  4. Call SerializeObject to encode and write
func WriteObjectNegotiated(
	ns runtime.NegotiatedSerializer,
	w http.ResponseWriter,
	req *http.Request,
	statusCode int,
	obj runtime.Object,
) {
	result, err := runtime.NegotiateOutputMediaType(req, ns)
	if err != nil {
		// if negotiation fails
		// and the status was already an error, fall back to raw JSON
		if isErrorStatusCode(statusCode) {
			WriteRawJSON(w, statusCode, obj)
			return
		}
		ErrorNegotiated(w, req, ns, fmt.Errorf("not acceptable: %w", err))
		return
	}

	SerializeObject(result.MediaType, result.Serializer, w, statusCode, obj)
}

// ErrorNegotiated writes an error response through the same negotiation path
func ErrorNegotiated(
	w http.ResponseWriter,
	req *http.Request,
	ns runtime.NegotiatedSerializer,
	err error,
) {
	// Build a simple error object
	errObj := &ErrorResponse{
		TypeMeta: runtime.TypeMeta{
			APIVersion: "v1",
			Kind:       "Status",
		},
		Status:  "Failure",
		Message: err.Error(),
		Code:    http.StatusNotAcceptable,
	}

	result, negErr := runtime.NegotiateOutputMediaType(req, ns)
	if negErr != nil {
		// Ultimate fallback: raw JSON
		WriteRawJSON(w, http.StatusNotAcceptable, errObj)
		return
	}

	SerializeObject(result.MediaType, result.Serializer, w, http.StatusNotAcceptable, errObj)
}

// SerializeObject encodes the object and writes the HTTP response
//
//  1. Create a buffer for encoding
//  2. Encode the object via the encoder
//  3. Set Content-Type header
//  4. Write status code and body
func SerializeObject(
	mediaType string,
	encoder runtime.Encoder,
	w http.ResponseWriter,
	statusCode int,
	obj runtime.Object,
) {
	// Buffer the encoded output
	buf := &bytes.Buffer{}
	if err := encoder.Encode(obj, buf); err != nil {
		// If encoding fails, try to serialize an error status
		internalError(w, fmt.Errorf("encoding response: %w", err))
		return
	}

	// Set headers and write response
	w.Header().Set("Content-Type", mediaType)
	w.WriteHeader(statusCode)
	_, _ = w.Write(buf.Bytes())
}

// WriteRawJSON writes a non-API object in JSON
func WriteRawJSON(w http.ResponseWriter, statusCode int, object any) {
	output, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(output)
}

// ErrorResponse is a simplified API error status object
type ErrorResponse struct {
	runtime.TypeMeta `json:",inline" yaml:",inline"`
	Status           string `json:"status" yaml:"status"`
	Message          string `json:"message" yaml:"message"`
	Code             int    `json:"code" yaml:"code"`
}

// GetTypeMeta implements runtime.Object.
func (e *ErrorResponse) GetTypeMeta() *runtime.TypeMeta {
	return &e.TypeMeta
}

func isErrorStatusCode(code int) bool {
	return code >= 400
}

func internalError(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
}
