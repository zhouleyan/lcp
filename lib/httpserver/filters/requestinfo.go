package filters

import (
	"net/http"

	"lcp.io/lcp/lib/logger"
)

// WithRequestInfo attaches a RequestInfo to the context
func WithRequestInfo(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Infof("Request Info: %s", r.RequestURI)
		handler.ServeHTTP(w, r)
	})
}
