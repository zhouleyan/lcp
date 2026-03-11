package connector

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"

	"lcp.io/lcp/lib/clients/sshclient"
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

// NewConnector creates a connector based on host and variables.
// It selects the connector type using the "connection" variable:
//   - "local": returns a LocalConnector
//   - "ssh" or "" (default): returns an SSHConnector, unless the host
//     resolves to a local address, in which case a LocalConnector is returned
//
// Recognised variables:
//
//	connection, become, become_user, password,
//	port, remote_user, private_key, private_key_content
func NewConnector(host string, vars map[string]any) (Connector, error) {
	connType, _ := vars["connection"].(string)
	become, _ := vars["become"].(bool)
	becomeUser, _ := vars["become_user"].(string)
	password, _ := vars["password"].(string)

	switch connType {
	case "local":
		return NewLocalConnector(password), nil
	case "ssh", "":
		if isLocal(host) {
			return NewLocalConnector(password), nil
		}
		config := sshclient.Config{
			Host:     host,
			Password: password,
		}
		if port, ok := vars["port"].(int); ok {
			config.Port = port
		}
		if user, ok := vars["remote_user"].(string); ok {
			config.User = user
		}
		if key, ok := vars["private_key"].(string); ok {
			config.PrivateKey = key
		}
		if keyContent, ok := vars["private_key_content"].(string); ok {
			config.PrivateKeyContent = keyContent
		}
		return NewSSHConnector(config, become, becomeUser), nil
	default:
		return nil, fmt.Errorf("unsupported connection type: %s", connType)
	}
}

// isLocal checks if host refers to the local machine. It matches
// "localhost", well-known loopback addresses, the OS hostname, and any
// IP address assigned to a local network interface.
func isLocal(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Check against the OS hostname.
	if hostname, err := os.Hostname(); err == nil && host == hostname {
		return true
	}

	// Check against local network interface addresses.
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			continue
		}
		if ip.String() == host {
			return true
		}
	}

	return false
}
