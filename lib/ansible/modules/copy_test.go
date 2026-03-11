package modules

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- copy tests ---

func TestModuleCopy_Content(t *testing.T) {
	conn := &mockConnector{}
	opts := ExecOptions{
		Args: map[string]any{
			"content": "hello world",
			"dest":    "/tmp/hello.txt",
		},
		Connector: conn,
	}

	stdout, stderr, err := ModuleCopy(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "/tmp/hello.txt") {
		t.Errorf("expected stdout to mention dest, got %q", stdout)
	}
	if string(conn.putFileData) != "hello world" {
		t.Errorf("expected PutFile data %q, got %q", "hello world", string(conn.putFileData))
	}
	if conn.putFileDest != "/tmp/hello.txt" {
		t.Errorf("expected PutFile dest %q, got %q", "/tmp/hello.txt", conn.putFileDest)
	}
	if conn.putFileMode != 0644 {
		t.Errorf("expected default mode 0644, got %04o", conn.putFileMode)
	}
}

func TestModuleCopy_ContentWithMode(t *testing.T) {
	conn := &mockConnector{}
	opts := ExecOptions{
		Args: map[string]any{
			"content": "#!/bin/bash\necho hi",
			"dest":    "/usr/local/bin/greet.sh",
			"mode":    "0755",
		},
		Connector: conn,
	}

	_, _, err := ModuleCopy(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.putFileMode != 0755 {
		t.Errorf("expected mode 0755, got %04o", conn.putFileMode)
	}
}

func TestModuleCopy_ContentWithIntMode(t *testing.T) {
	conn := &mockConnector{}
	opts := ExecOptions{
		Args: map[string]any{
			"content": "data",
			"dest":    "/tmp/f",
			"mode":    0600,
		},
		Connector: conn,
	}

	_, _, err := ModuleCopy(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.putFileMode != 0600 {
		t.Errorf("expected mode 0600, got %04o", conn.putFileMode)
	}
}

func TestModuleCopy_Src(t *testing.T) {
	conn := &mockConnector{}
	source := &mockSource{
		files: map[string][]byte{
			"configs/app.yaml": []byte("key: value"),
		},
	}
	opts := ExecOptions{
		Args: map[string]any{
			"src":  "configs/app.yaml",
			"dest": "/etc/app/config.yaml",
		},
		Connector: conn,
		Source:    source,
	}

	stdout, stderr, err := ModuleCopy(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "/etc/app/config.yaml") {
		t.Errorf("expected stdout to mention dest, got %q", stdout)
	}
	if string(conn.putFileData) != "key: value" {
		t.Errorf("expected PutFile data %q, got %q", "key: value", string(conn.putFileData))
	}
}

func TestModuleCopy_SrcNotFound(t *testing.T) {
	conn := &mockConnector{}
	source := &mockSource{files: map[string][]byte{}}
	opts := ExecOptions{
		Args: map[string]any{
			"src":  "missing.txt",
			"dest": "/tmp/missing.txt",
		},
		Connector: conn,
		Source:    source,
	}

	_, _, err := ModuleCopy(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for missing source file")
	}
	if !strings.Contains(err.Error(), "read source") {
		t.Errorf("expected error to mention 'read source', got %q", err.Error())
	}
}

func TestModuleCopy_NoDest(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"content": "test",
		},
	}

	_, _, err := ModuleCopy(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when dest is missing")
	}
	if !strings.Contains(err.Error(), "dest is required") {
		t.Errorf("expected 'dest is required' error, got %q", err.Error())
	}
}

func TestModuleCopy_NoSrcOrContent(t *testing.T) {
	conn := &mockConnector{}
	opts := ExecOptions{
		Args: map[string]any{
			"dest": "/tmp/test.txt",
		},
		Connector: conn,
	}

	_, _, err := ModuleCopy(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when neither src nor content is provided")
	}
	if !strings.Contains(err.Error(), "either src or content") {
		t.Errorf("expected 'either src or content' error, got %q", err.Error())
	}
}

func TestModuleCopy_PutFileError(t *testing.T) {
	conn := &mockConnector{putFileErr: fmt.Errorf("connection refused")}
	opts := ExecOptions{
		Args: map[string]any{
			"content": "data",
			"dest":    "/tmp/f.txt",
		},
		Connector: conn,
	}

	_, _, err := ModuleCopy(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error from PutFile failure")
	}
	if !strings.Contains(err.Error(), "put file") {
		t.Errorf("expected 'put file' error, got %q", err.Error())
	}
}

func TestModuleCopy_NoSource(t *testing.T) {
	conn := &mockConnector{}
	opts := ExecOptions{
		Args: map[string]any{
			"src":  "relative/path.txt",
			"dest": "/tmp/path.txt",
		},
		Connector: conn,
		Source:    nil,
	}

	_, _, err := ModuleCopy(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when source is nil for relative path")
	}
}

// --- fetch tests ---

func TestModuleFetch(t *testing.T) {
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "fetched.txt")

	conn := &mockConnector{
		fetchFileContent: []byte("remote file content"),
	}
	opts := ExecOptions{
		Args: map[string]any{
			"src":  "/remote/file.txt",
			"dest": destPath,
		},
		Connector: conn,
	}

	stdout, stderr, err := ModuleFetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "/remote/file.txt") {
		t.Errorf("expected stdout to mention src, got %q", stdout)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(data) != "remote file content" {
		t.Errorf("expected file content %q, got %q", "remote file content", string(data))
	}
}

func TestModuleFetch_NoSrc(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"dest": "/tmp/dest.txt",
		},
	}

	_, _, err := ModuleFetch(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when src is missing")
	}
	if !strings.Contains(err.Error(), "src is required") {
		t.Errorf("expected 'src is required' error, got %q", err.Error())
	}
}

func TestModuleFetch_NoDest(t *testing.T) {
	opts := ExecOptions{
		Args: map[string]any{
			"src": "/remote/file.txt",
		},
	}

	_, _, err := ModuleFetch(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when dest is missing")
	}
	if !strings.Contains(err.Error(), "dest is required") {
		t.Errorf("expected 'dest is required' error, got %q", err.Error())
	}
}

func TestModuleFetch_ConnectorError(t *testing.T) {
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "out.txt")

	conn := &mockConnector{
		fetchFileErr: fmt.Errorf("network error"),
	}
	opts := ExecOptions{
		Args: map[string]any{
			"src":  "/remote/file.txt",
			"dest": destPath,
		},
		Connector: conn,
	}

	_, _, err := ModuleFetch(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error from FetchFile failure")
	}
	if !strings.Contains(err.Error(), "fetch file") {
		t.Errorf("expected 'fetch file' error, got %q", err.Error())
	}
}

func TestModuleFetch_CreatesDestDir(t *testing.T) {
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "sub", "dir", "fetched.txt")

	conn := &mockConnector{
		fetchFileContent: []byte("data"),
	}
	opts := ExecOptions{
		Args: map[string]any{
			"src":  "/remote/file.txt",
			"dest": destPath,
		},
		Connector: conn,
	}

	_, _, err := ModuleFetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(data) != "data" {
		t.Errorf("expected %q, got %q", "data", string(data))
	}
}

// --- helper function tests ---

func TestStringArg(t *testing.T) {
	args := map[string]any{
		"name":  "hello",
		"count": 42,
		"empty": "",
	}

	if got := stringArg(args, "name"); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	if got := stringArg(args, "count"); got != "" {
		t.Errorf("expected empty string for non-string value, got %q", got)
	}
	if got := stringArg(args, "missing"); got != "" {
		t.Errorf("expected empty string for missing key, got %q", got)
	}
	if got := stringArg(args, "empty"); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFileModeArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		key      string
		def      fs.FileMode
		expected fs.FileMode
	}{
		{"missing key", map[string]any{}, "mode", 0644, 0644},
		{"int value", map[string]any{"mode": 0755}, "mode", 0644, 0755},
		{"int64 value", map[string]any{"mode": int64(0600)}, "mode", 0644, 0600},
		{"float64 value", map[string]any{"mode": float64(0644)}, "mode", 0755, 0644},
		{"string octal", map[string]any{"mode": "0755"}, "mode", 0644, 0755},
		{"string octal 644", map[string]any{"mode": "644"}, "mode", 0755, 0644},
		{"invalid string", map[string]any{"mode": "notanumber"}, "mode", 0644, 0644},
		{"unsupported type", map[string]any{"mode": true}, "mode", 0644, 0644},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileModeArg(tt.args, tt.key, tt.def)
			if got != tt.expected {
				t.Errorf("expected %04o, got %04o", tt.expected, got)
			}
		})
	}
}

func TestReadSource_Relative(t *testing.T) {
	source := &mockSource{
		files: map[string][]byte{
			"templates/config.yaml": []byte("key: val"),
		},
	}

	data, err := readSource(source, "templates/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "key: val" {
		t.Errorf("expected %q, got %q", "key: val", string(data))
	}
}

func TestReadSource_NilSource(t *testing.T) {
	_, err := readSource(nil, "relative/path.txt")
	if err == nil {
		t.Fatal("expected error with nil source")
	}
}

// --- registration tests ---

func TestCopyModuleRegistered(t *testing.T) {
	if fn := FindModule("copy"); fn == nil {
		t.Error("expected 'copy' module to be registered")
	}
}

func TestFetchModuleRegistered(t *testing.T) {
	if fn := FindModule("fetch"); fn == nil {
		t.Error("expected 'fetch' module to be registered")
	}
}
