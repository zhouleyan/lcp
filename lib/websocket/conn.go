package websocket

import (
	"context"
	"io"

	cws "github.com/coder/websocket"
)

// Re-export status codes so callers do not need to import coder/websocket.
const (
	StatusNormalClosure = cws.StatusNormalClosure
	StatusGoingAway     = cws.StatusGoingAway
	StatusInternalError = cws.StatusInternalError
)

// Conn wraps a coder/websocket.Conn to isolate the third-party dependency.
type Conn struct {
	inner *cws.Conn
}

// NewConn creates a Conn wrapping the given coder/websocket connection.
func NewConn(c *cws.Conn) *Conn {
	return &Conn{inner: c}
}

// ReadMessage reads a complete WebSocket message.
// It returns the message type and the raw bytes.
func (c *Conn) ReadMessage(ctx context.Context) (cws.MessageType, []byte, error) {
	msgType, reader, err := c.inner.Reader(ctx)
	if err != nil {
		return 0, nil, err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, err
	}
	return msgType, data, nil
}

// WriteMessage writes a complete WebSocket message of the given type.
func (c *Conn) WriteMessage(ctx context.Context, msgType cws.MessageType, data []byte) error {
	return c.inner.Write(ctx, msgType, data)
}

// WriteBinary writes a binary WebSocket message.
func (c *Conn) WriteBinary(ctx context.Context, data []byte) error {
	return c.WriteMessage(ctx, cws.MessageBinary, data)
}

// Close sends a close frame with the given status code and reason.
func (c *Conn) Close(code cws.StatusCode, reason string) error {
	return c.inner.Close(code, reason)
}

// Inner returns the underlying coder/websocket connection.
// Use sparingly; prefer the wrapper methods.
func (c *Conn) Inner() *cws.Conn {
	return c.inner
}
