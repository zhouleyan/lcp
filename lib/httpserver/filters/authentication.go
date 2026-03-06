package filters

import (
	"fmt"
	"net/http"
	"strings"

	"lcp.io/lcp/lib/oidc"
)

// WithAuthentication returns middleware that validates Bearer tokens.
// Requests without a valid token receive 401 Unauthorized.
func WithAuthentication(provider *oidc.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				authError(w, "missing authorization header")
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				authError(w, "invalid authorization header format")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			userID, err := provider.VerifyBearerToken(tokenStr)
			if err != nil {
				authError(w, "invalid or expired token")
				return
			}

			if err := provider.CheckUserActive(r.Context(), userID); err != nil {
				authError(w, "account is not active")
				return
			}

			r = oidc.WithUserID(r, userID)
			next.ServeHTTP(w, r)
		})
	}
}

func authError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = fmt.Fprintf(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Unauthorized","message":"%s"}`, message)
}
