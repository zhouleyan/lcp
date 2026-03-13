package filters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"lcp.io/lcp/lib/oidc"
)

func TestResolveResourceAndVerb(t *testing.T) {
	tests := []struct {
		method    string
		path      string
		wantMod   string
		wantChain string
		wantVerb  string
	}{
		// Basic CRUD
		{"GET", "/api/iam/v1/users", "iam", "users", "list"},
		{"GET", "/api/iam/v1/users/1", "iam", "users", "get"},
		{"POST", "/api/iam/v1/users", "iam", "users", "create"},
		{"PUT", "/api/iam/v1/users/1", "iam", "users", "update"},
		{"PATCH", "/api/iam/v1/users/1", "iam", "users", "patch"},
		{"DELETE", "/api/iam/v1/users/1", "iam", "users", "delete"},
		{"DELETE", "/api/iam/v1/users", "iam", "users", "deleteCollection"},

		// Nested resources
		{"GET", "/api/iam/v1/workspaces/1/namespaces", "iam", "workspaces:namespaces", "list"},
		{"GET", "/api/iam/v1/workspaces/1/namespaces/2", "iam", "workspaces:namespaces", "get"},
		{"POST", "/api/iam/v1/workspaces/1/namespaces", "iam", "workspaces:namespaces", "create"},
		{"DELETE", "/api/iam/v1/workspaces/1/namespaces/2", "iam", "workspaces:namespaces", "delete"},

		// Deep nesting
		{"GET", "/api/iam/v1/workspaces/1/namespaces/2/users", "iam", "workspaces:namespaces:users", "list"},
		{"POST", "/api/iam/v1/workspaces/1/namespaces/2/users", "iam", "workspaces:namespaces:users", "create"},
		{"DELETE", "/api/iam/v1/workspaces/1/namespaces/2/users", "iam", "workspaces:namespaces:users", "deleteCollection"},

		// Action
		{"POST", "/api/iam/v1/users/1/change-password", "iam", "users", "change-password"},

		// Custom verb (colon syntax maps to parent "get")
		{"GET", "/api/iam/v1/users/1:workspaces", "iam", "users", "get"},
		{"GET", "/api/iam/v1/users/1:namespaces", "iam", "users", "get"},

		// Cross-module
		{"GET", "/api/infra/v1/hosts", "infra", "hosts", "list"},
		{"POST", "/api/infra/v1/workspaces/1/namespaces/2/networks", "infra", "workspaces:namespaces:networks", "create"},

		// Edge cases
		{"GET", "/api/iam/v1", "", "", ""},           // no resource
		{"GET", "/api/iam", "", "", ""},              // no version
		{"GET", "/not-api/iam/v1/users", "", "", ""}, // no /api/ prefix → empty
	}

	for _, tt := range tests {
		mod, chain, verb := ResolveResourceAndVerb(tt.method, tt.path)
		if mod != tt.wantMod || chain != tt.wantChain || verb != tt.wantVerb {
			t.Errorf("ResolveResourceAndVerb(%s, %s) = (%q, %q, %q), want (%q, %q, %q)",
				tt.method, tt.path, mod, chain, verb, tt.wantMod, tt.wantChain, tt.wantVerb)
		}
	}
}

func TestIsSelfUserQuery(t *testing.T) {
	tests := []struct {
		path     string
		permCode string
		userID   int64
		want     bool
	}{
		{"/api/iam/v1/users/1", "iam:users:get", 1, true},
		{"/api/iam/v1/users/1:workspaces", "iam:users:get", 1, true},
		{"/api/iam/v1/users/1:namespaces", "iam:users:get", 1, true},
		{"/api/iam/v1/users/1/change-password", "iam:users:change-password", 1, true},
		{"/api/iam/v1/users/2", "iam:users:get", 1, false},           // different user
		{"/api/iam/v1/users/1", "iam:users:update", 1, false},        // not get/change-password
		{"/api/iam/v1/users/1", "iam:users:delete", 1, false},        // not get/change-password
		{"/api/iam/v1/workspaces/1", "iam:workspaces:get", 1, false}, // not a user path
	}

	for _, tt := range tests {
		got := isSelfUserQuery(tt.path, tt.permCode, tt.userID)
		if got != tt.want {
			t.Errorf("isSelfUserQuery(%q, %q, %d) = %v, want %v",
				tt.path, tt.permCode, tt.userID, got, tt.want)
		}
	}
}

// --- Mock types for middleware integration tests ---

type mockLookup map[string]map[string]map[string][]string

func (m mockLookup) Get(module, chain, verb string) []string {
	if mc, ok := m[module]; ok {
		if rc, ok := mc[chain]; ok {
			return rc[verb]
		}
	}
	return nil
}

type mockChecker struct {
	isAdmin      bool
	permissions  map[string]bool // permCode → allowed
	workspaceIDs []int64
	namespaceIDs []int64
}

func (m *mockChecker) CheckPermission(_ context.Context, _ int64, permCode string, _ string, _, _ int64) (bool, error) {
	return m.permissions[permCode], nil
}

func (m *mockChecker) CheckAnyPermission(_ context.Context, _ int64, permCodes []string, _ string, _, _ int64) (bool, error) {
	for _, code := range permCodes {
		if m.permissions[code] {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockChecker) IsPlatformAdmin(_ context.Context, _ int64) (bool, error) {
	return m.isAdmin, nil
}

func (m *mockChecker) GetAccessibleWorkspaceIDs(_ context.Context, _ int64) ([]int64, error) {
	return m.workspaceIDs, nil
}

func (m *mockChecker) GetAccessibleNamespaceIDs(_ context.Context, _ int64) ([]int64, error) {
	return m.namespaceIDs, nil
}

func TestWithAuthorization_Allowed(t *testing.T) {
	lookup := mockLookup{
		"iam": {"users": {"list": {"iam:users:list"}}},
	}
	checker := &mockChecker{permissions: map[string]bool{"iam:users:list": true}}
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/users", nil)
	r = oidc.WithUserID(r, 1)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWithAuthorization_Denied(t *testing.T) {
	lookup := mockLookup{
		"iam": {"users": {"list": {"iam:users:list"}}},
	}
	checker := &mockChecker{permissions: map[string]bool{}}
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/users", nil)
	r = oidc.WithUserID(r, 2)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestWithAuthorization_PlatformAdminBypass(t *testing.T) {
	lookup := mockLookup{
		"iam": {"users": {"list": {"iam:users:list"}}},
	}
	checker := &mockChecker{isAdmin: true, permissions: map[string]bool{}}
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/users", nil)
	r = oidc.WithUserID(r, 1)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for platform admin, got %d", w.Code)
	}
}

func TestWithAuthorization_SelfUserAccess(t *testing.T) {
	lookup := mockLookup{
		"iam": {"users": {"get": {"iam:users:get"}}},
	}
	// User has no permissions but should access own profile
	checker := &mockChecker{permissions: map[string]bool{}}
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/users/42", nil)
	r = oidc.WithUserID(r, 42)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for self user access, got %d", w.Code)
	}
}

func TestWithAuthorization_SelfUserDeniedForOthers(t *testing.T) {
	lookup := mockLookup{
		"iam": {"users": {"get": {"iam:users:get"}}},
	}
	checker := &mockChecker{permissions: map[string]bool{}}
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/users/99", nil)
	r = oidc.WithUserID(r, 42) // user 42 accessing user 99
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for other user access, got %d", w.Code)
	}
}

func TestWithAuthorization_NonAPIPassthrough(t *testing.T) {
	lookup := mockLookup{}
	checker := &mockChecker{}
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/oidc/userinfo", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for non-API path, got %d", w.Code)
	}
}

func TestWithAuthorization_WorkspaceListInjectsAccessFilter(t *testing.T) {
	lookup := mockLookup{
		"iam": {"workspaces": {"list": {"iam:workspaces:list"}}},
	}
	checker := &mockChecker{
		permissions:  map[string]bool{},
		workspaceIDs: []int64{1, 2, 3},
	}

	var capturedFilter *AccessFilter
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedFilter = AccessFilterFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/workspaces", nil)
	r = oidc.WithUserID(r, 42)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for workspace list, got %d", w.Code)
	}
	if capturedFilter == nil {
		t.Fatal("expected access filter in context")
	}
	if len(capturedFilter.WorkspaceIDs) != 3 {
		t.Errorf("expected 3 workspace IDs, got %d", len(capturedFilter.WorkspaceIDs))
	}
}

func TestWithAuthorization_NamespaceListInjectsAccessFilter(t *testing.T) {
	lookup := mockLookup{
		"iam": {"namespaces": {"list": {"iam:namespaces:list"}}},
	}
	checker := &mockChecker{
		permissions:  map[string]bool{},
		namespaceIDs: []int64{10, 20},
	}

	var capturedFilter *AccessFilter
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedFilter = AccessFilterFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/namespaces", nil)
	r = oidc.WithUserID(r, 42)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for namespace list, got %d", w.Code)
	}
	if capturedFilter == nil {
		t.Fatal("expected access filter in context")
	}
	if len(capturedFilter.NamespaceIDs) != 2 {
		t.Errorf("expected 2 namespace IDs, got %d", len(capturedFilter.NamespaceIDs))
	}
}

func TestWithAuthorization_AdminNoAccessFilter(t *testing.T) {
	lookup := mockLookup{
		"iam": {"workspaces": {"list": {"iam:workspaces:list"}}},
	}
	checker := &mockChecker{isAdmin: true}

	var capturedFilter *AccessFilter
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedFilter = AccessFilterFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/workspaces", nil)
	r = oidc.WithUserID(r, 1)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d", w.Code)
	}
	if capturedFilter != nil {
		t.Error("expected no access filter for admin")
	}
}

func TestWithAuthorization_UnregisteredPermissionAllowed(t *testing.T) {
	lookup := mockLookup{} // empty lookup
	checker := &mockChecker{permissions: map[string]bool{}}
	handler := WithAuthorization(lookup, checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/iam/v1/unknown-resource", nil)
	r = oidc.WithUserID(r, 1)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for unregistered permission, got %d", w.Code)
	}
}
