package connector

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"

	"lcp.io/lcp/lib/clients/sshclient"
)

// Compile-time interface checks.
var _ Connector = &SSHConnector{}
var _ GatherFacts = &SSHConnector{}

// SSHConnector implements Connector and GatherFacts using SSH.
// It delegates connection management, command execution, and file transfer
// to the underlying sshclient.Client.
type SSHConnector struct {
	client     *sshclient.Client
	host       string
	password   string // for sudo
	become     bool
	becomeUser string
}

// NewSSHConnector creates an SSH connector from the given config.
// The connection is not established until Init is called.
func NewSSHConnector(config sshclient.Config, become bool, becomeUser string) *SSHConnector {
	return &SSHConnector{
		client:     sshclient.New(config),
		host:       config.Host,
		password:   config.Password,
		become:     become,
		becomeUser: becomeUser,
	}
}

// Init establishes the SSH connection.
func (s *SSHConnector) Init(ctx context.Context) error {
	return s.client.Connect(ctx)
}

// Close closes the SSH connection and releases resources.
func (s *SSHConnector) Close(_ context.Context) error {
	return s.client.Close()
}

// ExecuteCommand executes a command on the remote host via SSH.
// If become is true, the command is wrapped with sudo. If becomeUser is
// set, "sudo -u <user>" is used. The sudo password is supplied via the
// sshclient.ExecWithSudo mechanism when available.
func (s *SSHConnector) ExecuteCommand(ctx context.Context, cmd string) ([]byte, []byte, error) {
	if s.become {
		actualCmd := cmd
		if s.becomeUser != "" && s.becomeUser != "root" {
			// sudo -u <user> to run as a specific non-root user
			actualCmd = fmt.Sprintf("-u %s %s", s.becomeUser, cmd)
		}
		result, err := s.client.ExecWithSudo(ctx, actualCmd, s.password)
		if result != nil {
			return result.Stdout, result.Stderr, err
		}
		return nil, nil, err
	}

	result, err := s.client.Exec(ctx, cmd)
	if result != nil {
		return result.Stdout, result.Stderr, err
	}
	return nil, nil, err
}

// PutFile uploads content to a remote path via SFTP.
func (s *SSHConnector) PutFile(_ context.Context, src []byte, dst string, mode fs.FileMode) error {
	return s.client.PutFile(src, dst, mode)
}

// FetchFile downloads a remote file via SFTP.
func (s *SSHConnector) FetchFile(_ context.Context, src string, dst io.Writer) error {
	return s.client.FetchFile(src, dst)
}

// HostInfo gathers remote system information by executing commands and
// reading files over the SSH connection. It collects OS release info,
// kernel version, hostname, architecture, CPU info, and memory info.
func (s *SSHConnector) HostInfo(ctx context.Context) (map[string]any, error) {
	// OS information
	osVars := make(map[string]any)

	var osRelease bytes.Buffer
	if err := s.FetchFile(ctx, "/etc/os-release", &osRelease); err != nil {
		return nil, fmt.Errorf("failed to read /etc/os-release: %w", err)
	}
	osVars["os_release"] = convertBytesToMap(osRelease.Bytes(), "=")

	kernel, stderr, err := s.ExecuteCommand(ctx, "uname -r")
	if err != nil {
		return nil, fmt.Errorf("failed to get kernel version: %w (stderr: %s)", err, string(stderr))
	}
	osVars["kernel_version"] = string(bytes.TrimSpace(kernel))

	hostname, stderr, err := s.ExecuteCommand(ctx, "hostname")
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w (stderr: %s)", err, string(stderr))
	}
	osVars["hostname"] = string(bytes.TrimSpace(hostname))

	arch, stderr, err := s.ExecuteCommand(ctx, "arch")
	if err != nil {
		return nil, fmt.Errorf("failed to get architecture: %w (stderr: %s)", err, string(stderr))
	}
	osVars["architecture"] = string(bytes.TrimSpace(arch))

	// Process information
	procVars := make(map[string]any)

	var cpuInfo bytes.Buffer
	if err := s.FetchFile(ctx, "/proc/cpuinfo", &cpuInfo); err != nil {
		return nil, fmt.Errorf("failed to read /proc/cpuinfo: %w", err)
	}
	procVars["cpu"] = convertBytesToSlice(cpuInfo.Bytes(), ":")

	var memInfo bytes.Buffer
	if err := s.FetchFile(ctx, "/proc/meminfo", &memInfo); err != nil {
		return nil, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}
	procVars["memory"] = convertBytesToMap(memInfo.Bytes(), ":")

	return map[string]any{
		"os":      osVars,
		"process": procVars,
	}, nil
}
