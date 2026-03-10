package audit

import (
	"context"
	"time"
)

// Event represents a single audit log entry.
type Event struct {
	UserID       *int64
	Username     string
	EventType    string // "api_operation" | "authentication"
	Action       string
	ResourceType string
	ResourceID   string
	Module       string
	Scope        string
	WorkspaceID  *int64
	NamespaceID  *int64
	HTTPMethod   string
	HTTPPath     string
	StatusCode   int
	ClientIP     string
	UserAgent    string
	DurationMs   int
	Success      bool
	Detail       string
	CreatedAt    time.Time
}

// Logger is the interface for emitting audit events.
type Logger interface {
	Log(event Event)
}

// Sink persists audit events to storage.
type Sink interface {
	BatchCreate(ctx context.Context, events []Event) error
}
