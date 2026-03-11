package iam

import (
	"context"
	"testing"

	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

// --- mock storage types ---

// mockStandardStorage needs a field to avoid Go zero-size struct pointer sharing.
type mockStandardStorage struct{ id int }

func (*mockStandardStorage) Get(_ context.Context, _ *rest.GetOptions) (runtime.Object, error) {
	return nil, nil
}
func (*mockStandardStorage) List(_ context.Context, _ *rest.ListOptions) (runtime.Object, error) {
	return nil, nil
}
func (*mockStandardStorage) Create(_ context.Context, _ runtime.Object, _ *rest.CreateOptions) (runtime.Object, error) {
	return nil, nil
}
func (*mockStandardStorage) Update(_ context.Context, _ runtime.Object, _ *rest.UpdateOptions) (runtime.Object, error) {
	return nil, nil
}
func (*mockStandardStorage) Patch(_ context.Context, _ runtime.Object, _ *rest.PatchOptions) (runtime.Object, error) {
	return nil, nil
}
func (*mockStandardStorage) Delete(_ context.Context, _ *rest.DeleteOptions) error { return nil }
func (*mockStandardStorage) DeleteCollection(_ context.Context, _ []string, _ *rest.DeleteOptions) (*rest.DeletionResult, error) {
	return nil, nil
}

type mockListerStorage struct{ id int }

func (*mockListerStorage) List(_ context.Context, _ *rest.ListOptions) (runtime.Object, error) {
	return nil, nil
}

// --- tests ---

func TestCollectStorageEntries(t *testing.T) {
	nsStorage := &mockStandardStorage{id: 1}
	nsUserStorage := &mockListerStorage{id: 1}

	resources := []rest.ResourceInfo{
		{Name: "users", Storage: &mockStandardStorage{id: 2}},
		{
			Name:    "workspaces",
			Storage: &mockStandardStorage{id: 3},
			SubResources: []rest.ResourceInfo{
				{
					Name:    "namespaces",
					Storage: nsStorage,
					SubResources: []rest.ResourceInfo{
						{Name: "users", Storage: nsUserStorage},
					},
				},
				{Name: "users", Storage: &mockListerStorage{id: 2}},
			},
		},
		{
			Name:    "namespaces",
			Storage: nsStorage,
			SubResources: []rest.ResourceInfo{
				{Name: "users", Storage: nsUserStorage},
			},
		},
	}

	entries := collectStorageEntries(resources, nil, nil)

	// users, workspaces, workspaces/namespaces, workspaces/namespaces/users,
	// workspaces/users, namespaces, namespaces/users
	if len(entries) != 7 {
		t.Fatalf("expected 7 entries, got %d", len(entries))
	}
}

func TestCanonicalize(t *testing.T) {
	nsStorage := &mockStandardStorage{id: 1}
	nsUserStorage := &mockListerStorage{id: 1}

	entries := []storageEntry{
		{storage: nsStorage, codeParts: []string{"workspaces", "namespaces"}, pathParts: []string{"workspaces", "namespaces"}},
		{storage: nsStorage, codeParts: []string{"namespaces"}, pathParts: []string{"namespaces"}},
		{storage: nsUserStorage, codeParts: []string{"workspaces", "namespaces", "users"}, pathParts: []string{"workspaces", "namespaces", "users"}},
		{storage: nsUserStorage, codeParts: []string{"namespaces", "users"}, pathParts: []string{"namespaces", "users"}},
	}

	canonical := canonicalize(entries)

	if got := canonical[nsStorage]; len(got.codeParts) != 1 || got.codeParts[0] != "namespaces" {
		t.Errorf("nsStorage canonical: got %v, want [namespaces]", got.codeParts)
	}

	if got := canonical[nsUserStorage]; len(got.codeParts) != 2 {
		t.Errorf("nsUserStorage canonical: got %v, want [namespaces users]", got.codeParts)
	}
}

func TestDetectVerbs(t *testing.T) {
	full := &mockStandardStorage{id: 1}
	verbs := detectVerbs(full)
	expected := map[string]bool{
		"list": true, "get": true, "create": true,
		"update": true, "patch": true, "delete": true, "deleteCollection": true,
	}
	for _, v := range verbs {
		if !expected[v] {
			t.Errorf("unexpected verb %q", v)
		}
		delete(expected, v)
	}
	if len(expected) > 0 {
		t.Errorf("missing verbs: %v", expected)
	}

	lister := &mockListerStorage{id: 1}
	verbs = detectVerbs(lister)
	if len(verbs) != 1 || verbs[0] != "list" {
		t.Errorf("lister verbs: got %v, want [list]", verbs)
	}
}

func TestBuildLookup(t *testing.T) {
	nsStorage := &mockStandardStorage{id: 1}

	entries := []storageEntry{
		{storage: nsStorage, codeParts: []string{"workspaces", "namespaces"}, pathParts: []string{"workspaces", "namespaces"}},
		{storage: nsStorage, codeParts: []string{"namespaces"}, pathParts: []string{"namespaces"}},
	}

	canonical := map[rest.Storage]storageEntry{
		nsStorage: {storage: nsStorage, codeParts: []string{"namespaces"}, pathParts: []string{"namespaces"}},
	}

	lookup := make(PermissionLookup)
	buildLookup(lookup, entries, canonical, "iam")

	// Both paths for the same nsStorage should resolve to the same canonical code
	code1 := lookup.Get("iam", "workspaces:namespaces", "list")
	code2 := lookup.Get("iam", "namespaces", "list")

	if code1 != "iam:namespaces:list" {
		t.Errorf("workspaces:namespaces list = %q, want iam:namespaces:list", code1)
	}
	if code2 != "iam:namespaces:list" {
		t.Errorf("namespaces list = %q, want iam:namespaces:list", code2)
	}
	if code1 != code2 {
		t.Errorf("codes should match: %q != %q", code1, code2)
	}
}

// --- mock PermissionStore for SyncPermissions ---

type mockPermissionStoreForSync struct {
	upserted map[string]*DBPermission
	deleted  map[string][]string
	synced   map[string][]DBPermission // modulePrefix → perms passed to SyncModule
}

func newMockPermissionStoreForSync() *mockPermissionStoreForSync {
	return &mockPermissionStoreForSync{
		upserted: make(map[string]*DBPermission),
		deleted:  make(map[string][]string),
		synced:   make(map[string][]DBPermission),
	}
}

func (m *mockPermissionStoreForSync) Upsert(_ context.Context, perm *DBPermission) (*DBPermission, error) {
	m.upserted[perm.Code] = perm
	return perm, nil
}

func (m *mockPermissionStoreForSync) DeleteByModuleNotInCodeScopes(_ context.Context, modulePrefix string, keepCodeScopes []string) error {
	m.deleted[modulePrefix] = keepCodeScopes
	return nil
}

func (m *mockPermissionStoreForSync) SyncModule(_ context.Context, modulePrefix string, perms []DBPermission) error {
	m.synced[modulePrefix] = perms
	// Also populate upserted map so existing test assertions work (key = code:scope)
	for i := range perms {
		key := perms[i].Code + ":" + perms[i].Scope
		m.upserted[key] = &perms[i]
	}
	return nil
}

func (m *mockPermissionStoreForSync) GetByCode(_ context.Context, _, _ string) (*DBPermission, error) {
	return nil, nil
}

func (m *mockPermissionStoreForSync) List(_ context.Context, _ db.ListQuery) (*db.ListResult[DBPermission], error) {
	return nil, nil
}

func (m *mockPermissionStoreForSync) ListAllCodes(_ context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockPermissionStoreForSync) ListCodeScopes(_ context.Context) ([]PermissionCodeScope, error) {
	return nil, nil
}

func TestSyncPermissions(t *testing.T) {
	nsStorage := &mockStandardStorage{id: 1}
	nsUserStorage := &mockListerStorage{id: 1}
	wsUserStorage := &mockListerStorage{id: 2}

	group := &rest.APIGroupInfo{
		GroupName: "iam",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{
				Name:    "users",
				Storage: &mockStandardStorage{id: 2},
				Actions: []rest.ActionInfo{
					{Name: "change-password", Method: "POST"},
				},
			},
			{
				Name:    "workspaces",
				Storage: &mockStandardStorage{id: 3},
				SubResources: []rest.ResourceInfo{
					{
						Name:    "namespaces",
						Storage: nsStorage,
						SubResources: []rest.ResourceInfo{
							{Name: "users", Storage: nsUserStorage},
						},
					},
					{Name: "users", Storage: wsUserStorage},
				},
			},
			{
				Name:    "namespaces",
				Storage: nsStorage,
				SubResources: []rest.ResourceInfo{
					{Name: "users", Storage: nsUserStorage},
				},
			},
		},
	}

	store := newMockPermissionStoreForSync()
	lookup, err := SyncPermissions(context.Background(), store, []*rest.APIGroupInfo{group})
	if err != nil {
		t.Fatalf("SyncPermissions: %v", err)
	}

	// Check key permissions were upserted (key = code:scope)
	// Platform-level resources generate only platform scope
	// Workspace-level resources generate workspace + platform scopes
	// Namespace-level resources generate namespace + workspace + platform scopes
	expectedCodeScopes := []string{
		// Top-level users (platform)
		"iam:users:list:platform", "iam:users:get:platform", "iam:users:create:platform",
		"iam:users:update:platform", "iam:users:patch:platform", "iam:users:delete:platform",
		"iam:users:deleteCollection:platform",
		// Top-level workspaces (platform)
		"iam:workspaces:list:platform", "iam:workspaces:get:platform",
		// Namespaces: natural scope=workspace → workspace + platform
		"iam:namespaces:list:workspace", "iam:namespaces:list:platform",
		// Workspace users: natural scope=workspace → workspace + platform
		"iam:users:list:workspace", "iam:users:list:platform",
		// Namespace users: natural scope=namespace → namespace + workspace + platform
		"iam:users:list:namespace",
		// Action: change-password (platform)
		"iam:users:change-password:platform",
	}
	for _, codeScope := range expectedCodeScopes {
		if _, ok := store.upserted[codeScope]; !ok {
			t.Errorf("expected permission %q to be upserted", codeScope)
		}
	}

	// Shared nsStorage: both paths resolve to same canonical code (scope stripped)
	if code := lookup.Get("iam", "workspaces:namespaces", "list"); code != "iam:namespaces:list" {
		t.Errorf("lookup workspaces:namespaces list = %q, want iam:namespaces:list", code)
	}
	if code := lookup.Get("iam", "namespaces", "list"); code != "iam:namespaces:list" {
		t.Errorf("lookup namespaces list = %q, want iam:namespaces:list", code)
	}

	// Sub-resource users under namespaces → simplified to iam:users:list
	if code := lookup.Get("iam", "namespaces:users", "list"); code != "iam:users:list" {
		t.Errorf("lookup namespaces:users list = %q, want iam:users:list", code)
	}
	// Aliased path for same nsUserStorage
	if code := lookup.Get("iam", "workspaces:namespaces:users", "list"); code != "iam:users:list" {
		t.Errorf("lookup workspaces:namespaces:users list = %q, want iam:users:list", code)
	}

	// SyncModule should be called for "iam:" prefix
	if _, ok := store.synced["iam:"]; !ok {
		t.Error("expected SyncModule for iam: prefix")
	}
}

func TestSyncPermissionsMultiModule(t *testing.T) {
	iamGroup := &rest.APIGroupInfo{
		GroupName: "iam",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{Name: "users", Storage: &mockStandardStorage{id: 10}},
		},
	}
	infraGroup := &rest.APIGroupInfo{
		GroupName: "infra",
		Version:   "v1",
		Resources: []rest.ResourceInfo{
			{Name: "hosts", Storage: &mockStandardStorage{id: 20}},
		},
	}

	store := newMockPermissionStoreForSync()
	lookup, err := SyncPermissions(context.Background(), store, []*rest.APIGroupInfo{iamGroup, infraGroup})
	if err != nil {
		t.Fatalf("SyncPermissions: %v", err)
	}

	if code := lookup.Get("iam", "users", "list"); code != "iam:users:list" {
		t.Errorf("iam users list = %q", code)
	}
	if code := lookup.Get("infra", "hosts", "list"); code != "infra:hosts:list" {
		t.Errorf("infra hosts list = %q", code)
	}

	if _, ok := store.synced["iam:"]; !ok {
		t.Error("expected SyncModule for iam:")
	}
	if _, ok := store.synced["infra:"]; !ok {
		t.Error("expected SyncModule for infra:")
	}
}

func TestPermissionLookupGet(t *testing.T) {
	lookup := PermissionLookup{
		"iam": {
			"users": {
				"list": "iam:users:list",
				"get":  "iam:users:get",
			},
		},
	}

	if got := lookup.Get("iam", "users", "list"); got != "iam:users:list" {
		t.Errorf("got %q, want iam:users:list", got)
	}
	if got := lookup.Get("iam", "users", "delete"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := lookup.Get("unknown", "users", "list"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
