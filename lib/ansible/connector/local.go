package connector

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Compile-time interface checks.
var _ Connector = &LocalConnector{}
var _ GatherFacts = &LocalConnector{}

// defaultShell is the fallback shell when $SHELL is not set.
const defaultShell = "/bin/bash"

// LocalConnector executes commands and file operations on the local host.
type LocalConnector struct {
	// password is the sudo password (optional). If empty, sudo commands run
	// without password input (assumes NOPASSWD or root user).
	password string
	// shell is the path to the shell binary, detected from $SHELL during Init.
	shell string
}

// NewLocalConnector creates a new LocalConnector with the given sudo password.
func NewLocalConnector(password string) *LocalConnector {
	return &LocalConnector{
		password: password,
	}
}

// Init detects the shell from the $SHELL environment variable.
// Falls back to /bin/bash if $SHELL is not set.
func (c *LocalConnector) Init(_ context.Context) error {
	c.shell = os.Getenv("SHELL")
	if c.shell == "" {
		c.shell = defaultShell
	}
	return nil
}

// Close is a no-op for the local connector.
func (c *LocalConnector) Close(_ context.Context) error {
	return nil
}

// ExecuteCommand executes a command on the local host using the detected shell.
// The command is run via: <shell> -c "<cmd>"
func (c *LocalConnector) ExecuteCommand(ctx context.Context, cmd string) ([]byte, []byte, error) {
	command := exec.CommandContext(ctx, c.shell, "-c", cmd)

	var stdoutBuf, stderrBuf bytes.Buffer
	command.Stdout = &stdoutBuf
	command.Stderr = &stderrBuf

	err := command.Run()
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

// PutFile writes src bytes to the file at dst with the given file mode.
// Parent directories are created automatically if they do not exist.
func (c *LocalConnector) PutFile(_ context.Context, src []byte, dst string, mode fs.FileMode) error {
	dir := filepath.Dir(dst)
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat directory %q: %w", dir, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
	}
	if err := os.WriteFile(dst, src, mode); err != nil {
		return fmt.Errorf("failed to write file %q: %w", dst, err)
	}
	return nil
}

// FetchFile reads the file at src and copies its contents to dst.
func (c *LocalConnector) FetchFile(_ context.Context, src string, dst io.Writer) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", src, err)
	}
	defer file.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return fmt.Errorf("failed to copy file %q: %w", src, err)
	}
	return nil
}

// HostInfo gathers local system information.
// On Linux, it reads /etc/os-release, runs uname/hostname/arch commands,
// and reads /proc/cpuinfo and /proc/meminfo.
// On non-Linux platforms, it returns an empty map.
func (c *LocalConnector) HostInfo(ctx context.Context) (map[string]any, error) {
	if runtime.GOOS != "linux" {
		return make(map[string]any), nil
	}

	// OS information
	osVars := make(map[string]any)

	var osRelease bytes.Buffer
	if err := c.FetchFile(ctx, "/etc/os-release", &osRelease); err != nil {
		return nil, fmt.Errorf("failed to read /etc/os-release: %w", err)
	}
	osVars["os_release"] = convertBytesToMap(osRelease.Bytes(), "=")

	kernel, stderr, err := c.ExecuteCommand(ctx, "uname -r")
	if err != nil {
		return nil, fmt.Errorf("failed to get kernel version: %w (stderr: %s)", err, string(stderr))
	}
	osVars["kernel_version"] = string(bytes.TrimSpace(kernel))

	hostname, stderr, err := c.ExecuteCommand(ctx, "hostname")
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w (stderr: %s)", err, string(stderr))
	}
	osVars["hostname"] = string(bytes.TrimSpace(hostname))

	arch, stderr, err := c.ExecuteCommand(ctx, "arch")
	if err != nil {
		return nil, fmt.Errorf("failed to get architecture: %w (stderr: %s)", err, string(stderr))
	}
	osVars["architecture"] = string(bytes.TrimSpace(arch))

	// Process information
	procVars := make(map[string]any)

	var cpuInfo bytes.Buffer
	if err := c.FetchFile(ctx, "/proc/cpuinfo", &cpuInfo); err != nil {
		return nil, fmt.Errorf("failed to read /proc/cpuinfo: %w", err)
	}
	procVars["cpu"] = convertBytesToSlice(cpuInfo.Bytes(), ":")

	var memInfo bytes.Buffer
	if err := c.FetchFile(ctx, "/proc/meminfo", &memInfo); err != nil {
		return nil, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}
	procVars["memory"] = convertBytesToMap(memInfo.Bytes(), ":")

	return map[string]any{
		"os":      osVars,
		"process": procVars,
	}, nil
}

// convertBytesToMap parses a byte slice into a map[string]string using the given
// delimiter. Only lines containing the delimiter are processed. Each line is
// split at the first occurrence; key and value are trimmed of whitespace.
func convertBytesToMap(bs []byte, split string) map[string]string {
	config := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(bs))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, split, 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}
	return config
}

// convertBytesToSlice parses a byte slice into a slice of map[string]string.
// Groups of key-value pairs are separated by empty lines. Each group is stored
// as a separate map.
func convertBytesToSlice(bs []byte, split string) []map[string]string {
	var config []map[string]string
	currentMap := make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewReader(bs))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			parts := strings.SplitN(line, split, 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				currentMap[key] = value
			}
		} else if len(currentMap) > 0 {
			config = append(config, currentMap)
			currentMap = make(map[string]string)
		}
	}
	if len(currentMap) > 0 {
		config = append(config, currentMap)
	}
	return config
}
