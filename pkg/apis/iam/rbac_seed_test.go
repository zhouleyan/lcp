package iam

import (
	"context"
	"fmt"
	"testing"

	"lcp.io/lcp/pkg/db"
)

// --- mock RoleStore for SeedRBAC ---

type mockRoleStoreForSeed struct {
	roles  map[string]*DBRole
	rules  map[int64][]string
	nextID int64
}

func newMockRoleStoreForSeed() *mockRoleStoreForSeed {
	return &mockRoleStoreForSeed{
		roles:  make(map[string]*DBRole),
		rules:  make(map[int64][]string),
		nextID: 1,
	}
}

func (m *mockRoleStoreForSeed) Create(_ context.Context, role *DBRole) (*DBRole, error) {
	m.nextID++
	role.ID = m.nextID
	m.roles[role.Name] = role
	return role, nil
}

func (m *mockRoleStoreForSeed) GetByID(_ context.Context, id int64) (*DBRoleWithRules, error) {
	for _, r := range m.roles {
		if r.ID == id {
			return &DBRoleWithRules{Role: *r, Rules: m.rules[id]}, nil
		}
	}
	return nil, fmt.Errorf("role %d not found", id)
}

func (m *mockRoleStoreForSeed) GetByName(_ context.Context, name string) (*DBRole, error) {
	if r, ok := m.roles[name]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("role %q not found", name)
}

func (m *mockRoleStoreForSeed) Update(_ context.Context, role *DBRole) (*DBRole, error) {
	return role, nil
}

func (m *mockRoleStoreForSeed) Upsert(_ context.Context, role *DBRole) (*DBRole, error) {
	if existing, ok := m.roles[role.Name]; ok {
		existing.DisplayName = role.DisplayName
		existing.Description = role.Description
		existing.Scope = role.Scope
		existing.Builtin = role.Builtin
		return existing, nil
	}
	m.nextID++
	role.ID = m.nextID
	m.roles[role.Name] = role
	return role, nil
}

func (m *mockRoleStoreForSeed) Delete(_ context.Context, _ int64) error { return nil }

func (m *mockRoleStoreForSeed) List(_ context.Context, _ db.ListQuery) (*db.ListResult[DBRoleListRow], error) {
	return nil, nil
}

func (m *mockRoleStoreForSeed) SetPermissionRules(_ context.Context, roleID int64, patterns []string) error {
	m.rules[roleID] = patterns
	return nil
}

func (m *mockRoleStoreForSeed) GetByNameAndWorkspace(_ context.Context, name string, _ int64) (*DBRole, error) {
	if r, ok := m.roles[name]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("role %q not found", name)
}

func (m *mockRoleStoreForSeed) GetByNameAndNamespace(_ context.Context, name string, _ int64) (*DBRole, error) {
	if r, ok := m.roles[name]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("role %q not found", name)
}

func (m *mockRoleStoreForSeed) CreateBuiltinRolesForWorkspace(_ context.Context, _ int64) error {
	return nil
}

func (m *mockRoleStoreForSeed) CreateBuiltinRolesForNamespace(_ context.Context, _ int64) error {
	return nil
}

func (m *mockRoleStoreForSeed) SeedRBAC(_ context.Context, roles []BuiltinRoleDef, _ string) error {
	for _, def := range roles {
		role, _ := m.Upsert(context.Background(), &DBRole{
			Name:        def.Name,
			DisplayName: def.DisplayName,
			Description: def.Description,
			Scope:       def.Scope,
			Builtin:     true,
		})
		m.rules[role.ID] = def.Rules
	}
	return nil
}

// --- tests ---

func TestSeedRBAC(t *testing.T) {
	store := newMockRoleStoreForSeed()

	if err := SeedRBAC(context.Background(), store); err != nil {
		t.Fatalf("SeedRBAC: %v", err)
	}

	// SeedRBAC now only seeds platform roles (2 roles)
	if len(store.roles) != 2 {
		t.Fatalf("expected 2 platform roles, got %d", len(store.roles))
	}

	// All roles should be builtin
	for name, role := range store.roles {
		if !role.Builtin {
			t.Errorf("role %q should be builtin", name)
		}
	}

	// Check platform admin role has "*:*" rule
	adminRole := store.roles[RolePlatformAdmin]
	if adminRole == nil {
		t.Fatalf("%s role not found", RolePlatformAdmin)
	}
	adminRules := store.rules[adminRole.ID]
	if len(adminRules) != 1 || adminRules[0] != "*:*" {
		t.Errorf("%s rules = %v, want [*:*]", RolePlatformAdmin, adminRules)
	}

	// Check platform viewer role has "*:list" and "*:get" rules
	viewerRole := store.roles[RolePlatformViewer]
	if viewerRole == nil {
		t.Fatalf("%s role not found", RolePlatformViewer)
	}
	viewerRules := store.rules[viewerRole.ID]
	if len(viewerRules) != 2 {
		t.Errorf("%s rules = %v, want [*:list *:get]", RolePlatformViewer, viewerRules)
	} else {
		ruleSet := map[string]bool{viewerRules[0]: true, viewerRules[1]: true}
		if !ruleSet["*:list"] || !ruleSet["*:get"] {
			t.Errorf("%s rules = %v, want [*:list *:get]", RolePlatformViewer, viewerRules)
		}
	}

	// Check scopes
	expectedScopes := map[string]string{
		RolePlatformAdmin:  "platform",
		RolePlatformViewer: "platform",
	}
	for name, scope := range expectedScopes {
		role := store.roles[name]
		if role == nil {
			t.Errorf("role %q not found", name)
			continue
		}
		if role.Scope != scope {
			t.Errorf("role %q scope = %q, want %q", name, role.Scope, scope)
		}
	}
}

func TestSeedRBACIdempotent(t *testing.T) {
	store := newMockRoleStoreForSeed()

	// Seed twice
	if err := SeedRBAC(context.Background(), store); err != nil {
		t.Fatalf("first SeedRBAC: %v", err)
	}
	if err := SeedRBAC(context.Background(), store); err != nil {
		t.Fatalf("second SeedRBAC: %v", err)
	}

	// Should still have exactly 2 platform roles (no duplicates)
	if len(store.roles) != 2 {
		t.Fatalf("expected 2 roles after double seed, got %d", len(store.roles))
	}

	// Rules should be correct (overwritten, not accumulated)
	adminRole := store.roles[RolePlatformAdmin]
	adminRules := store.rules[adminRole.ID]
	if len(adminRules) != 1 || adminRules[0] != "*:*" {
		t.Errorf("%s rules after double seed = %v, want [*:*]", RolePlatformAdmin, adminRules)
	}
}

func TestBuiltinRoleHelpers(t *testing.T) {
	platform := PlatformBuiltinRoles()
	if len(platform) != 2 {
		t.Errorf("PlatformBuiltinRoles() returned %d roles, want 2", len(platform))
	}
	for _, r := range platform {
		if r.Scope != "platform" {
			t.Errorf("PlatformBuiltinRoles() role %q has scope %q, want platform", r.Name, r.Scope)
		}
	}

	workspace := WorkspaceBuiltinRoles()
	if len(workspace) != 2 {
		t.Errorf("WorkspaceBuiltinRoles() returned %d roles, want 2", len(workspace))
	}
	for _, r := range workspace {
		if r.Scope != "workspace" {
			t.Errorf("WorkspaceBuiltinRoles() role %q has scope %q, want workspace", r.Name, r.Scope)
		}
	}

	namespace := NamespaceBuiltinRoles()
	if len(namespace) != 2 {
		t.Errorf("NamespaceBuiltinRoles() returned %d roles, want 2", len(namespace))
	}
	for _, r := range namespace {
		if r.Scope != "namespace" {
			t.Errorf("NamespaceBuiltinRoles() role %q has scope %q, want namespace", r.Name, r.Scope)
		}
	}
}
