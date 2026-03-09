package iam

import (
	"context"
	"fmt"
	"testing"

	"lcp.io/lcp/pkg/db"
)

// --- mock RoleStore for SeedBuiltinRoles ---

type mockRoleStoreForSeed struct {
	roles map[string]*DBRole
	rules map[int64][]string
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
	role.Builtin = true
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

func (m *mockRoleStoreForSeed) List(_ context.Context, _ db.ListQuery) (*db.ListResult[DBRole], error) {
	return nil, nil
}

func (m *mockRoleStoreForSeed) SetPermissionRules(_ context.Context, roleID int64, patterns []string) error {
	m.rules[roleID] = patterns
	return nil
}

func (m *mockRoleStoreForSeed) SeedBuiltinRoles(_ context.Context, roles []BuiltinRoleDef) error {
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

func TestSeedBuiltinRoles(t *testing.T) {
	store := newMockRoleStoreForSeed()

	if err := SeedBuiltinRoles(context.Background(), store); err != nil {
		t.Fatalf("SeedBuiltinRoles: %v", err)
	}

	// Should have created 6 roles
	if len(store.roles) != 6 {
		t.Fatalf("expected 6 roles, got %d", len(store.roles))
	}

	// All roles should be builtin
	for name, role := range store.roles {
		if !role.Builtin {
			t.Errorf("role %q should be builtin", name)
		}
	}

	// Check platform-admin has "*:*" rule
	adminRole := store.roles["platform-admin"]
	if adminRole == nil {
		t.Fatal("platform-admin role not found")
	}
	adminRules := store.rules[adminRole.ID]
	if len(adminRules) != 1 || adminRules[0] != "*:*" {
		t.Errorf("platform-admin rules = %v, want [*:*]", adminRules)
	}

	// Check all admin roles have "*:*" rule
	for _, name := range []string{"workspace-admin", "namespace-admin"} {
		role := store.roles[name]
		if role == nil {
			t.Fatalf("%s role not found", name)
		}
		rules := store.rules[role.ID]
		if len(rules) != 1 || rules[0] != "*:*" {
			t.Errorf("%s rules = %v, want [*:*]", name, rules)
		}
	}

	// Check all viewer roles have "*:list" and "*:get" rules
	for _, name := range []string{"platform-viewer", "workspace-viewer", "namespace-viewer"} {
		role := store.roles[name]
		if role == nil {
			t.Fatalf("%s role not found", name)
		}
		rules := store.rules[role.ID]
		if len(rules) != 2 {
			t.Errorf("%s rules = %v, want [*:list *:get]", name, rules)
			continue
		}
		ruleSet := map[string]bool{rules[0]: true, rules[1]: true}
		if !ruleSet["*:list"] || !ruleSet["*:get"] {
			t.Errorf("%s rules = %v, want [*:list *:get]", name, rules)
		}
	}

	// Check scopes
	expectedScopes := map[string]string{
		"platform-admin":   "platform",
		"platform-viewer":  "platform",
		"workspace-admin":  "workspace",
		"workspace-viewer": "workspace",
		"namespace-admin":  "namespace",
		"namespace-viewer": "namespace",
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

func TestSeedBuiltinRolesIdempotent(t *testing.T) {
	store := newMockRoleStoreForSeed()

	// Seed twice
	if err := SeedBuiltinRoles(context.Background(), store); err != nil {
		t.Fatalf("first SeedBuiltinRoles: %v", err)
	}
	if err := SeedBuiltinRoles(context.Background(), store); err != nil {
		t.Fatalf("second SeedBuiltinRoles: %v", err)
	}

	// Should still have exactly 6 roles (no duplicates)
	if len(store.roles) != 6 {
		t.Fatalf("expected 6 roles after double seed, got %d", len(store.roles))
	}

	// Rules should be correct (overwritten, not accumulated)
	adminRole := store.roles["platform-admin"]
	adminRules := store.rules[adminRole.ID]
	if len(adminRules) != 1 || adminRules[0] != "*:*" {
		t.Errorf("platform-admin rules after double seed = %v, want [*:*]", adminRules)
	}
}
