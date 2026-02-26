package rest

import (
	"encoding/json"
	"net/http"
)

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
