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

func TestCreateNamespace_Success(t *testing.T) {
	now := time.Now()
	ms := &mockStore{
		users: &mockUserStore{
			getByIDFn: func(_ context.Context, id int64) (*store.User, error) {
				return &store.User{ID: id, Username: "owner"}, nil
			},
		},
		namespaces: &mockNamespaceStore{
			createFn: func(_ context.Context, ns *store.Namespace) (*store.Namespace, error) {
				return &store.Namespace{
					ID:          10,
					Name:        ns.Name,
					DisplayName: ns.DisplayName,
					Description: ns.Description,
					OwnerID:     ns.OwnerID,
					Visibility:  ns.Visibility,
					MaxMembers:  ns.MaxMembers,
					Status:      "active",
					CreatedAt:   now,
					UpdatedAt:   now,
				}, nil
			},
		},
		userNs: &mockUserNamespaceStore{},
	}

	svc := New(ms)
	result, err := svc.Namespaces().CreateNamespace(context.Background(), &types.Namespace{
		ObjectMeta: types.ObjectMeta{Name: "my-namespace"},
		Spec: types.NamespaceSpec{
			OwnerID:    "1",
			Visibility: "public",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ObjectMeta.ID != "10" {
		t.Errorf("expected ID '10', got %q", result.ObjectMeta.ID)
	}
	if result.Spec.OwnerID != "1" {
		t.Errorf("expected OwnerID '1', got %q", result.Spec.OwnerID)
	}
	if result.Kind != "Namespace" {
		t.Errorf("expected Kind 'Namespace', got %q", result.Kind)
	}
}

func TestCreateNamespace_ValidationError(t *testing.T) {
	ms := &mockStore{
		users:      &mockUserStore{},
		namespaces: &mockNamespaceStore{},
		userNs:     &mockUserNamespaceStore{},
	}

	svc := New(ms)
	_, err := svc.Namespaces().CreateNamespace(context.Background(), &types.Namespace{
		ObjectMeta: types.ObjectMeta{Name: ""}, // missing name
		Spec:       types.NamespaceSpec{OwnerID: ""},
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

func TestCreateNamespace_OwnerNotFound(t *testing.T) {
	ms := &mockStore{
		users: &mockUserStore{
			getByIDFn: func(_ context.Context, _ int64) (*store.User, error) {
				return nil, errors.New("not found")
			},
		},
		namespaces: &mockNamespaceStore{},
		userNs:     &mockUserNamespaceStore{},
	}

	svc := New(ms)
	_, err := svc.Namespaces().CreateNamespace(context.Background(), &types.Namespace{
		ObjectMeta: types.ObjectMeta{Name: "my-namespace"},
		Spec:       types.NamespaceSpec{OwnerID: "999"},
	})
	if err == nil {
		t.Fatal("expected error for missing owner, got nil")
	}
	se, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("expected *StatusError, got %T", err)
	}
	if se.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", se.Status)
	}
}

func TestAddMember_Success(t *testing.T) {
	now := time.Now()
	ms := &mockStore{
		users: &mockUserStore{
			getByIDFn: func(_ context.Context, id int64) (*store.User, error) {
				return &store.User{ID: id, Username: "user1"}, nil
			},
		},
		namespaces: &mockNamespaceStore{
			getByIDFn: func(_ context.Context, id int64) (*store.Namespace, error) {
				return &store.Namespace{ID: id, Name: "ns1"}, nil
			},
		},
		userNs: &mockUserNamespaceStore{
			addFn: func(_ context.Context, rel *store.UserNamespaceRole) (*store.UserNamespaceRole, error) {
				return &store.UserNamespaceRole{
					UserID:      rel.UserID,
					NamespaceID: rel.NamespaceID,
					Role:        rel.Role,
					CreatedAt:   now,
				}, nil
			},
		},
	}

	svc := New(ms)
	result, err := svc.Namespaces().AddMember(context.Background(), "10", &types.NamespaceMember{
		Spec: types.NamespaceMemberSpec{
			UserID: "5",
			Role:   "member",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Spec.UserID != "5" {
		t.Errorf("expected UserID '5', got %q", result.Spec.UserID)
	}
	if result.Spec.Role != "member" {
		t.Errorf("expected Role 'member', got %q", result.Spec.Role)
	}
	if result.Kind != "NamespaceMember" {
		t.Errorf("expected Kind 'NamespaceMember', got %q", result.Kind)
	}
}

func TestAddMember_ValidationError(t *testing.T) {
	ms := &mockStore{
		users:      &mockUserStore{},
		namespaces: &mockNamespaceStore{},
		userNs:     &mockUserNamespaceStore{},
	}

	svc := New(ms)
	_, err := svc.Namespaces().AddMember(context.Background(), "10", &types.NamespaceMember{
		Spec: types.NamespaceMemberSpec{
			UserID: "",
			Role:   "",
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
