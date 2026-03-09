package filters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockNamespaceResolver implements NamespaceResolver for testing.
type mockNamespaceResolver struct {
	mapping map[int64]int64
}

func (m *mockNamespaceResolver) GetWorkspaceID(namespaceID int64) (int64, bool) {
	wsID, ok := m.mapping[namespaceID]
	return wsID, ok
}

func TestResolveRequestInfo(t *testing.T) {
	resolver := &mockNamespaceResolver{
		mapping: map[int64]int64{
			456: 789,
			100: 200,
		},
	}

	tests := []struct {
		name      string
		path      string
		resolver  NamespaceResolver
		wantScope string
		wantWsID  int64
		wantNsID  int64
	}{
		{
			name:      "platform - users list",
			path:      "/api/iam/v1/users",
			resolver:  resolver,
			wantScope: "platform",
			wantWsID:  0,
			wantNsID:  0,
		},
		{
			name:      "workspace scope",
			path:      "/api/iam/v1/workspaces/123",
			resolver:  resolver,
			wantScope: "workspace",
			wantWsID:  123,
			wantNsID:  0,
		},
		{
			name:      "namespace scope with workspace in path",
			path:      "/api/iam/v1/workspaces/123/namespaces/456",
			resolver:  resolver,
			wantScope: "namespace",
			wantWsID:  123,
			wantNsID:  456,
		},
		{
			name:      "namespace scope without workspace - resolver fills in",
			path:      "/api/iam/v1/namespaces/456",
			resolver:  resolver,
			wantScope: "namespace",
			wantWsID:  789,
			wantNsID:  456,
		},
		{
			name:      "namespace scope without workspace - resolver no match",
			path:      "/api/iam/v1/namespaces/999",
			resolver:  resolver,
			wantScope: "namespace",
			wantWsID:  0,
			wantNsID:  999,
		},
		{
			name:      "custom verb does not affect scope",
			path:      "/api/iam/v1/users/1:workspaces",
			resolver:  resolver,
			wantScope: "platform",
			wantWsID:  0,
			wantNsID:  0,
		},
		{
			name:      "deep nesting - namespace scope",
			path:      "/api/iam/v1/workspaces/123/namespaces/456/users",
			resolver:  resolver,
			wantScope: "namespace",
			wantWsID:  123,
			wantNsID:  456,
		},
		{
			name:      "non-numeric workspace ID",
			path:      "/api/iam/v1/workspaces/abc",
			resolver:  resolver,
			wantScope: "platform",
			wantWsID:  0,
			wantNsID:  0,
		},
		{
			name:      "nil resolver - namespace without workspace",
			path:      "/api/iam/v1/namespaces/456",
			resolver:  nil,
			wantScope: "namespace",
			wantWsID:  0,
			wantNsID:  456,
		},
		{
			name:      "workspace list under workspace",
			path:      "/api/iam/v1/workspaces/123/namespaces",
			resolver:  resolver,
			wantScope: "workspace",
			wantWsID:  123,
			wantNsID:  0,
		},
		{
			name:      "cross module - infra hosts",
			path:      "/api/infra/v1/hosts",
			resolver:  resolver,
			wantScope: "platform",
			wantWsID:  0,
			wantNsID:  0,
		},
		{
			name:      "cross module - infra namespace networks",
			path:      "/api/infra/v1/workspaces/10/namespaces/20/networks",
			resolver:  resolver,
			wantScope: "namespace",
			wantWsID:  10,
			wantNsID:  20,
		},
		{
			name:      "custom verb with namespace ID",
			path:      "/api/iam/v1/workspaces/5/namespaces/6:users",
			resolver:  resolver,
			wantScope: "namespace",
			wantWsID:  5,
			wantNsID:  6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := resolveRequestInfo(tt.path, tt.resolver)
			if info.Scope != tt.wantScope {
				t.Errorf("scope = %q, want %q", info.Scope, tt.wantScope)
			}
			if info.WorkspaceID != tt.wantWsID {
				t.Errorf("workspaceID = %d, want %d", info.WorkspaceID, tt.wantWsID)
			}
			if info.NamespaceID != tt.wantNsID {
				t.Errorf("namespaceID = %d, want %d", info.NamespaceID, tt.wantNsID)
			}
		})
	}
}

func TestWithRequestInfoMiddleware(t *testing.T) {
	resolver := &mockNamespaceResolver{
		mapping: map[int64]int64{456: 789},
	}

	var captured *RequestInfo
	handler := WithRequestInfo(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = RequestInfoFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/iam/v1/workspaces/123/namespaces/456", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if captured == nil {
		t.Fatal("expected RequestInfo in context")
	}
	if captured.Scope != "namespace" || captured.WorkspaceID != 123 || captured.NamespaceID != 456 {
		t.Errorf("unexpected RequestInfo: %+v", captured)
	}
}

func TestRequestInfoFromContextDefault(t *testing.T) {
	info := RequestInfoFromContext(context.Background())
	if info.Scope != "platform" || info.WorkspaceID != 0 || info.NamespaceID != 0 {
		t.Errorf("unexpected default RequestInfo: %+v", info)
	}
}
