package filters

import (
	"net/http"

	"lcp.io/lcp/lib/logger"
)

// WithRequestLog logs each incoming request URI.
func WithRequestLog(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Infof("%s %s", r.Method, r.RequestURI)
		handler.ServeHTTP(w, r)
	})
}
