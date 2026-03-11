package connector

import (
	"context"
	"io"
	"io/fs"
)

// Connector is the interface for connecting to a host.
// It abstracts the operations required to interact with different types of hosts
// (e.g., local, SSH). Implementations provide mechanisms for initialization,
// cleanup, file transfer, and command execution.
type Connector interface {
	// Init initializes the connection.
	Init(ctx context.Context) error
	// Close closes the connection and releases any resources.
	Close(ctx context.Context) error
	// ExecuteCommand executes a command on the host.
	// Returns stdout, stderr, and error (if any).
	ExecuteCommand(ctx context.Context, cmd string) (stdout, stderr []byte, err error)
	// PutFile copies content from src (as bytes) to dst (path on host) with the specified file mode.
	PutFile(ctx context.Context, src []byte, dst string, mode fs.FileMode) error
	// FetchFile copies a file from src (path on host) to dst (writer).
	FetchFile(ctx context.Context, src string, dst io.Writer) error
}

// GatherFacts defines an interface for retrieving host information.
type GatherFacts interface {
	// HostInfo returns a map of host facts gathered from the system.
	HostInfo(ctx context.Context) (map[string]any, error)
}
