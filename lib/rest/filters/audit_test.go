package filters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"lcp.io/lcp/lib/audit"
)

type captureSink struct {
	events []audit.Event
}

func (c *captureSink) Log(event audit.Event) {
	c.events = append(c.events, event)
}

func TestIsAuditableRequest(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{"GET", "/api/iam/v1/users", false},
		{"POST", "/api/iam/v1/users", true},
		{"PUT", "/api/iam/v1/users/1", true},
		{"PATCH", "/api/iam/v1/users/1", true},
		{"DELETE", "/api/iam/v1/users/1", true},
		{"DELETE", "/api/iam/v1/users", true},
		{"GET", "/oidc/login", false},
		{"POST", "/oidc/login", false},
	}

	for _, tt := range tests {
		r := httptest.NewRequest(tt.method, tt.path, nil)
		got := isAuditableRequest(r)
		if got != tt.want {
			t.Errorf("isAuditableRequest(%s %s) = %v, want %v", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestStatusWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec, code: http.StatusOK}

	sw.WriteHeader(http.StatusForbidden)
	if sw.code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", sw.code)
	}
}

func TestExtractResourceID(t *testing.T) {
	tests := []struct {
		path string
		verb string
		want string
	}{
		{"/api/iam/v1/users/123", "update", "123"},
		{"/api/iam/v1/users/123", "delete", "123"},
		{"/api/iam/v1/users", "create", ""},
		{"/api/iam/v1/users", "deleteCollection", ""},
		{"/api/iam/v1/workspaces/1/namespaces/2", "update", "2"},
		{"/api/iam/v1/users/1/change-password", "change-password", ""},
	}

	for _, tt := range tests {
		got := extractResourceID(tt.path, tt.verb)
		if got != tt.want {
			t.Errorf("extractResourceID(%s, %s) = %q, want %q", tt.path, tt.verb, got, tt.want)
		}
	}
}

func TestWithAudit_Integration(t *testing.T) {
	sink := &captureSink{}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := WithAudit(sink)(inner)

	// POST should be audited
	r := httptest.NewRequest("POST", "/api/iam/v1/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if len(sink.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(sink.events))
	}

	event := sink.events[0]
	if event.EventType != "api_operation" {
		t.Errorf("expected event_type=api_operation, got %s", event.EventType)
	}
	if event.Action != "create" {
		t.Errorf("expected action=create, got %s", event.Action)
	}
	if event.StatusCode != http.StatusCreated {
		t.Errorf("expected status=201, got %d", event.StatusCode)
	}
	if event.Module != "iam" {
		t.Errorf("expected module=iam, got %s", event.Module)
	}
	if !event.Success {
		t.Error("expected success=true")
	}

	// GET should NOT be audited
	r = httptest.NewRequest("GET", "/api/iam/v1/users", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if len(sink.events) != 1 {
		t.Errorf("expected still 1 audit event after GET, got %d", len(sink.events))
	}
}

func TestWithAudit_FailedRequest(t *testing.T) {
	sink := &captureSink{}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	handler := WithAudit(sink)(inner)

	r := httptest.NewRequest("DELETE", "/api/iam/v1/users/5", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if len(sink.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(sink.events))
	}
	if sink.events[0].Success {
		t.Error("expected success=false for 403 response")
	}
	if sink.events[0].ResourceID != "5" {
		t.Errorf("expected resourceID=5, got %s", sink.events[0].ResourceID)
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name string
		xff  string
		xri  string
		addr string
		want string
	}{
		{"xff single", "1.2.3.4", "", "5.6.7.8:1234", "1.2.3.4"},
		{"xff multiple", "1.2.3.4, 5.6.7.8", "", "9.0.0.1:1234", "1.2.3.4"},
		{"xri", "", "10.0.0.1", "5.6.7.8:1234", "10.0.0.1"},
		{"remoteaddr", "", "", "192.168.1.1:9090", "192.168.1.1"},
	}

	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = tt.addr
		if tt.xff != "" {
			r.Header.Set("X-Forwarded-For", tt.xff)
		}
		if tt.xri != "" {
			r.Header.Set("X-Real-IP", tt.xri)
		}
		got := audit.ClientIP(r)
		if got != tt.want {
			t.Errorf("%s: ClientIP = %q, want %q", tt.name, got, tt.want)
		}
	}
}
