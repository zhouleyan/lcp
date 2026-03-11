package modules

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	RegisterModule("fetch", ModuleFetch)
}

// ModuleFetch downloads a file from a remote host to the local filesystem.
//
// Args:
//
//	src:  remote file path (required)
//	dest: local destination path (required)
func ModuleFetch(ctx context.Context, opts ExecOptions) (string, string, error) {
	src := stringArg(opts.Args, "src")
	if src == "" {
		return "", "", fmt.Errorf("fetch: src is required")
	}

	dest := stringArg(opts.Args, "dest")
	if dest == "" {
		return "", "", fmt.Errorf("fetch: dest is required")
	}

	// Ensure the destination directory exists.
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", "", fmt.Errorf("fetch: create dest dir %s: %w", destDir, err)
	}

	// Fetch file content from remote into a buffer.
	var buf bytes.Buffer
	if err := opts.Connector.FetchFile(ctx, src, &buf); err != nil {
		return "", "", fmt.Errorf("fetch: fetch file %s: %w", src, err)
	}

	// Write fetched content to the destination file.
	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		return "", "", fmt.Errorf("fetch: write file %s: %w", dest, err)
	}

	return fmt.Sprintf("fetched %s to %s", src, dest), "", nil
}
