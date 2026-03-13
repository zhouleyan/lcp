package rest

import (
	"context"
	"net/http"

	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/websocket"
)

// WebSocketHandler handles an upgraded WebSocket connection.
// The framework performs the HTTP → WebSocket upgrade; the handler
// receives the ready-to-use connection with path/query params.
type WebSocketHandler func(ctx context.Context, params map[string]string, conn *websocket.Conn)

// HandleWebSocket returns an http.HandlerFunc that upgrades the
// connection to WebSocket, extracts path and query params, and
// delegates to the WebSocketHandler.
func HandleWebSocket(handler WebSocketHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// TODO: Origin checking is currently skipped because WebSocket
			// endpoints use Bearer token authentication (not cookie-based),
			// so CSRF is not a concern. If cookie-based auth is added in
			// the future, replace this with an explicit list of allowed origins.
			InsecureSkipVerify: true,
		})
		if err != nil {
			logger.Warnf("websocket upgrade failed for %s %s: %v", r.Method, r.URL.Path, err)
			return
		}

		params := mergeQueryParams(PathParams(r), r)
		handler(r.Context(), params, conn)
	}
}
