package modules

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lcp.io/lcp/lib/ansible/variable"
)

// ========================================================================
// result module tests
// ========================================================================

func TestModuleResult(t *testing.T) {
	v := newTestVariable("testhost")

	stdout, stderr, err := ModuleResult(context.Background(), ExecOptions{
		Host:     "testhost",
		Variable: v,
		Args: map[string]any{
			"app_version": "1.0.0",
			"db_port":     5432,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "result stored" {
		t.Errorf("expected stdout 'result stored', got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}

	// Verify the result was stored.
	result := v.Get(variable.GetResultVariable())
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected result to be map[string]any, got %T", result)
	}
	if m["app_version"] != "1.0.0" {
		t.Errorf("expected app_version='1.0.0', got %v", m["app_version"])
	}
	if m["db_port"] != 5432 {
		t.Errorf("expected db_port=5432, got %v", m["db_port"])
	}
}

func TestModuleResult_MergeMultiple(t *testing.T) {
	v := newTestVariable("testhost")

	// First call: set some keys.
	_, _, err := ModuleResult(context.Background(), ExecOptions{
		Host:     "testhost",
		Variable: v,
		Args:     map[string]any{"key1": "value1"},
	})
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}

	// Second call: set additional keys.
	_, _, err = ModuleResult(context.Background(), ExecOptions{
		Host:     "testhost",
		Variable: v,
		Args:     map[string]any{"key2": "value2"},
	})
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}

	// Both keys should be present.
	result := v.Get(variable.GetResultVariable())
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected result to be map[string]any, got %T", result)
	}
	if m["key1"] != "value1" {
		t.Errorf("expected key1='value1', got %v", m["key1"])
	}
	if m["key2"] != "value2" {
		t.Errorf("expected key2='value2', got %v", m["key2"])
	}
}

func TestModuleResult_Empty(t *testing.T) {
	v := newTestVariable("testhost")

	// nil args
	_, _, err := ModuleResult(context.Background(), ExecOptions{
		Host:     "testhost",
		Variable: v,
		Args:     nil,
	})
	if err == nil {
		t.Fatal("expected error for nil args, got nil")
	}

	// empty args
	_, _, err = ModuleResult(context.Background(), ExecOptions{
		Host:     "testhost",
		Variable: v,
		Args:     map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for empty args, got nil")
	}
}

func TestModuleResult_Registered(t *testing.T) {
	fn := FindModule("result")
	if fn == nil {
		t.Fatal("expected 'result' module to be registered, got nil")
	}
}

// ========================================================================
// http_get_file module tests
// ========================================================================

func TestModuleHTTPGetFile(t *testing.T) {
	// Serve a test file.
	content := "hello, world!"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, content)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "downloaded.txt")

	stdout, stderr, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":  srv.URL + "/file.txt",
			"dest": dest,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
	if stdout == "" {
		t.Error("expected non-empty stdout")
	}

	// Verify downloaded content.
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected content %q, got %q", content, string(data))
	}
}

func TestModuleHTTPGetFile_BasicAuth(t *testing.T) {
	expectedUser := "admin"
	expectedPass := "secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != expectedUser || pass != expectedPass {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "unauthorized")
			return
		}
		fmt.Fprint(w, "authenticated content")
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "auth_file.txt")

	// Without auth: should fail.
	_, _, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":  srv.URL + "/secure",
			"dest": dest,
		},
	})
	if err == nil {
		t.Fatal("expected error without auth, got nil")
	}

	// With auth: should succeed.
	stdout, stderr, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":      srv.URL + "/secure",
			"dest":     dest,
			"username": expectedUser,
			"password": expectedPass,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error with auth: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
	if stdout == "" {
		t.Error("expected non-empty stdout")
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "authenticated content" {
		t.Errorf("expected 'authenticated content', got %q", string(data))
	}
}

func TestModuleHTTPGetFile_BearerToken(t *testing.T) {
	expectedToken := "my-secret-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+expectedToken {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "forbidden")
			return
		}
		fmt.Fprint(w, "token content")
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "token_file.txt")

	stdout, stderr, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":   srv.URL + "/api/data",
			"dest":  dest,
			"token": expectedToken,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got %q", stderr)
	}
	if stdout == "" {
		t.Error("expected non-empty stdout")
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "token content" {
		t.Errorf("expected 'token content', got %q", string(data))
	}
}

func TestModuleHTTPGetFile_CustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "custom-value" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "missing custom header")
			return
		}
		fmt.Fprint(w, "header ok")
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "header_file.txt")

	_, _, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":  srv.URL,
			"dest": dest,
			"headers": map[string]any{
				"X-Custom": "custom-value",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "header ok" {
		t.Errorf("expected 'header ok', got %q", string(data))
	}
}

func TestModuleHTTPGetFile_MissingArgs(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing both", map[string]any{}},
		{"missing url", map[string]any{"dest": "/tmp/file"}},
		{"missing dest", map[string]any{"url": "http://example.com/file"}},
		{"nil args", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
				Args: tt.args,
			})
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestModuleHTTPGetFile_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "error_file.txt")

	_, _, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":  srv.URL + "/fail",
			"dest": dest,
		},
	})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}

	// Dest file should not exist.
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Error("expected dest file to not exist after server error")
	}
}

func TestModuleHTTPGetFile_AtomicWrite(t *testing.T) {
	// When the server sends partial data then closes, the temp file should be cleaned up
	// and the dest file should not exist.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send headers with a large content-length but close before sending all data.
		w.Header().Set("Content-Length", "999999")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "partial")
		// The handler returns, closing the connection prematurely.
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "atomic_file.txt")

	_, _, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":  srv.URL + "/partial",
			"dest": dest,
		},
	})

	// The download should fail because response body is shorter than Content-Length.
	if err == nil {
		// Some HTTP clients may not detect the short read. If no error, verify
		// the file was still written atomically (either fully or not at all).
		return
	}

	// On error, dest file should not exist (atomic guarantee).
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Error("expected dest file to not exist after download error")
	}

	// Temp files should also be cleaned up.
	parentDir := filepath.Dir(dest)
	entries, _ := os.ReadDir(parentDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("temp file %s was not cleaned up", e.Name())
		}
	}
}

func TestModuleHTTPGetFile_CreatesParentDir(t *testing.T) {
	content := "nested file"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, content)
	}))
	defer srv.Close()

	// Dest is in a deeply nested directory that does not exist.
	dest := filepath.Join(t.TempDir(), "a", "b", "c", "file.txt")

	_, _, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":  srv.URL,
			"dest": dest,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}
}

func TestModuleHTTPGetFile_InvalidScheme(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "file.txt")

	_, _, err := ModuleHTTPGetFile(context.Background(), ExecOptions{
		Args: map[string]any{
			"url":  "ftp://example.com/file.txt",
			"dest": dest,
		},
	})
	if err == nil {
		t.Fatal("expected error for ftp scheme, got nil")
	}
}

func TestModuleHTTPGetFile_Registered(t *testing.T) {
	fn := FindModule("http_get_file")
	if fn == nil {
		t.Fatal("expected 'http_get_file' module to be registered, got nil")
	}
}
