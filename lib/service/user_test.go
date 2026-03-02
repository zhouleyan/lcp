package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/store"
)

// --- mock stores ---

type mockUserStore struct {
	createFn      func(ctx context.Context, user *store.User) (*store.User, error)
	getByIDFn     func(ctx context.Context, id int64) (*store.User, error)
	getByUserFn   func(ctx context.Context, username string) (*store.User, error)
	getByEmailFn  func(ctx context.Context, email string) (*store.User, error)
	updateFn      func(ctx context.Context, user *store.User) (*store.User, error)
	updateLoginFn func(ctx context.Context, id int64) error
	deleteFn      func(ctx context.Context, id int64) error
	listFn        func(ctx context.Context, query store.ListQuery) (*store.ListResult[store.UserWithNamespaces], error)
}

func (m *mockUserStore) Create(ctx context.Context, user *store.User) (*store.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil, nil
}

func (m *mockUserStore) GetByID(ctx context.Context, id int64) (*store.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockUserStore) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	if m.getByUserFn != nil {
		return m.getByUserFn(ctx, username)
	}
	return nil, nil
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *mockUserStore) Update(ctx context.Context, user *store.User) (*store.User, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil, nil
}

func (m *mockUserStore) UpdateLastLogin(ctx context.Context, id int64) error {
	if m.updateLoginFn != nil {
		return m.updateLoginFn(ctx, id)
	}
	return nil
}

func (m *mockUserStore) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserStore) List(ctx context.Context, query store.ListQuery) (*store.ListResult[store.UserWithNamespaces], error) {
	if m.listFn != nil {
		return m.listFn(ctx, query)
	}
	return nil, nil
}

type mockNamespaceStore struct {
	createFn  func(ctx context.Context, ns *store.Namespace) (*store.Namespace, error)
	getByIDFn func(ctx context.Context, id int64) (*store.Namespace, error)
	getByName func(ctx context.Context, name string) (*store.Namespace, error)
	updateFn  func(ctx context.Context, ns *store.Namespace) (*store.Namespace, error)
	deleteFn  func(ctx context.Context, id int64) error
	listFn    func(ctx context.Context, query store.ListQuery) (*store.ListResult[store.NamespaceWithOwner], error)
}

func (m *mockNamespaceStore) Create(ctx context.Context, ns *store.Namespace) (*store.Namespace, error) {
	if m.createFn != nil {
		return m.createFn(ctx, ns)
	}
	return nil, nil
}

func (m *mockNamespaceStore) GetByID(ctx context.Context, id int64) (*store.Namespace, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockNamespaceStore) GetByName(ctx context.Context, name string) (*store.Namespace, error) {
	if m.getByName != nil {
		return m.getByName(ctx, name)
	}
	return nil, nil
}

func (m *mockNamespaceStore) Update(ctx context.Context, ns *store.Namespace) (*store.Namespace, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, ns)
	}
	return nil, nil
}

func (m *mockNamespaceStore) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockNamespaceStore) List(ctx context.Context, query store.ListQuery) (*store.ListResult[store.NamespaceWithOwner], error) {
	if m.listFn != nil {
		return m.listFn(ctx, query)
	}
	return nil, nil
}

type mockUserNamespaceStore struct {
	addFn      func(ctx context.Context, rel *store.UserNamespaceRole) (*store.UserNamespaceRole, error)
	removeFn   func(ctx context.Context, userID, namespaceID int64) error
	updateFn   func(ctx context.Context, rel *store.UserNamespaceRole) (*store.UserNamespaceRole, error)
	getFn      func(ctx context.Context, userID, namespaceID int64) (*store.UserNamespaceRole, error)
	listByUser func(ctx context.Context, userID int64) ([]store.NamespaceWithRole, error)
	listByNsFn func(ctx context.Context, namespaceID int64) ([]store.UserWithRole, error)
}

func (m *mockUserNamespaceStore) Add(ctx context.Context, rel *store.UserNamespaceRole) (*store.UserNamespaceRole, error) {
	if m.addFn != nil {
		return m.addFn(ctx, rel)
	}
	return nil, nil
}

func (m *mockUserNamespaceStore) Remove(ctx context.Context, userID, namespaceID int64) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, userID, namespaceID)
	}
	return nil
}

func (m *mockUserNamespaceStore) UpdateRole(ctx context.Context, rel *store.UserNamespaceRole) (*store.UserNamespaceRole, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, rel)
	}
	return nil, nil
}

func (m *mockUserNamespaceStore) Get(ctx context.Context, userID, namespaceID int64) (*store.UserNamespaceRole, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userID, namespaceID)
	}
	return nil, nil
}

func (m *mockUserNamespaceStore) ListByUserID(ctx context.Context, userID int64) ([]store.NamespaceWithRole, error) {
	if m.listByUser != nil {
		return m.listByUser(ctx, userID)
	}
	return nil, nil
}

func (m *mockUserNamespaceStore) ListByNamespaceID(ctx context.Context, namespaceID int64) ([]store.UserWithRole, error) {
	if m.listByNsFn != nil {
		return m.listByNsFn(ctx, namespaceID)
	}
	return nil, nil
}

type mockStore struct {
	users      *mockUserStore
	namespaces *mockNamespaceStore
	userNs     *mockUserNamespaceStore
}

func (m *mockStore) Users() store.UserStore                    { return m.users }
func (m *mockStore) Namespaces() store.NamespaceStore          { return m.namespaces }
func (m *mockStore) UserNamespaces() store.UserNamespaceStore  { return m.userNs }
func (m *mockStore) WithTx(_ context.Context, _ func(store.Store) error) error { return nil }
func (m *mockStore) Close()                                    {}

func TestCreateUser_Success(t *testing.T) {
	now := time.Now()
	ms := &mockStore{
		users: &mockUserStore{
			createFn: func(_ context.Context, u *store.User) (*store.User, error) {
				return &store.User{
					ID:          1,
					Username:    u.Username,
					Email:       u.Email,
					DisplayName: u.DisplayName,
					Phone:       u.Phone,
					AvatarUrl:   u.AvatarUrl,
					Status:      "active",
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil
			},
		},
		namespaces: &mockNamespaceStore{},
		userNs:     &mockUserNamespaceStore{},
	}

	svc := New(ms)
	result, err := svc.Users().CreateUser(context.Background(), &types.User{
		Spec: types.UserSpec{
			Username: "alice",
			Email:    "alice@example.com",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ObjectMeta.ID != "1" {
		t.Errorf("expected ID '1', got %q", result.ObjectMeta.ID)
	}
	if result.Spec.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", result.Spec.Username)
	}
	if result.Kind != "User" {
		t.Errorf("expected Kind 'User', got %q", result.Kind)
	}
}

func TestCreateUser_ValidationError(t *testing.T) {
	ms := &mockStore{
		users:      &mockUserStore{},
		namespaces: &mockNamespaceStore{},
		userNs:     &mockUserNamespaceStore{},
	}

	svc := New(ms)
	_, err := svc.Users().CreateUser(context.Background(), &types.User{
		Spec: types.UserSpec{
			Username: "", // missing required field
			Email:    "",
		},
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	se, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if se.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", se.Status)
	}
}

func TestGetUser_Success(t *testing.T) {
	now := time.Now()
	ms := &mockStore{
		users: &mockUserStore{
			getByIDFn: func(_ context.Context, id int64) (*store.User, error) {
				return &store.User{
					ID:        id,
					Username:  "bob",
					Email:     "bob@example.com",
					Status:    "active",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
		},
		namespaces: &mockNamespaceStore{},
		userNs:     &mockUserNamespaceStore{},
	}

	svc := New(ms)
	obj, err := svc.Users().GetUser(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	user, ok := obj.(*types.User)
	if !ok {
		t.Fatalf("expected *types.User, got %T", obj)
	}
	if user.ObjectMeta.ID != "42" {
		t.Errorf("expected ID '42', got %q", user.ObjectMeta.ID)
	}
	if user.Spec.Username != "bob" {
		t.Errorf("expected username 'bob', got %q", user.Spec.Username)
	}
}

func TestGetUser_InvalidID(t *testing.T) {
	ms := &mockStore{
		users:      &mockUserStore{},
		namespaces: &mockNamespaceStore{},
		userNs:     &mockUserNamespaceStore{},
	}

	svc := New(ms)
	_, err := svc.Users().GetUser(context.Background(), "not-a-number")
	if err == nil {
		t.Fatal("expected error for invalid ID, got nil")
	}
	se, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if se.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", se.Status)
	}
}

func TestCreateUser_StoreError(t *testing.T) {
	ms := &mockStore{
		users: &mockUserStore{
			createFn: func(_ context.Context, _ *store.User) (*store.User, error) {
				return nil, errors.New("db connection failed")
			},
		},
		namespaces: &mockNamespaceStore{},
		userNs:     &mockUserNamespaceStore{},
	}

	svc := New(ms)
	_, err := svc.Users().CreateUser(context.Background(), &types.User{
		Spec: types.UserSpec{
			Username: "alice",
			Email:    "alice@example.com",
		},
	})
	if err == nil {
		t.Fatal("expected store error, got nil")
	}
	if _, ok := err.(*apierrors.StatusError); ok {
		t.Error("expected wrapped store error, not StatusError")
	}
}
