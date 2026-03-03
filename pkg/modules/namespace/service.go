package namespace

import (
	"context"
	"fmt"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/store"

	nsstore "lcp.io/lcp/pkg/modules/namespace/store"
)

// UserLookup is an interface for checking user existence (cross-module dependency).
type UserLookup interface {
	UserExists(ctx context.Context, userID int64) error
}

// NamespaceService handles namespace business logic.
type NamespaceService struct {
	nsStore    nsstore.NamespaceStore
	unStore    nsstore.UserNamespaceStore
	userLookup UserLookup
}

// NewNamespaceService creates a new NamespaceService.
func NewNamespaceService(nsStore nsstore.NamespaceStore, unStore nsstore.UserNamespaceStore, userLookup UserLookup) *NamespaceService {
	return &NamespaceService{
		nsStore:    nsStore,
		unStore:    unStore,
		userLookup: userLookup,
	}
}

func (n *NamespaceService) CreateNamespace(ctx context.Context, ns *Namespace) (*Namespace, error) {
	ownerID, err := parseID(ns.Spec.OwnerID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
	}

	if err := n.userLookup.UserExists(ctx, ownerID); err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("owner user %d not found", ownerID), nil)
	}

	created, err := n.nsStore.Create(ctx, &nsstore.Namespace{
		Name:        ns.ObjectMeta.Name,
		DisplayName: ns.Spec.DisplayName,
		Description: ns.Spec.Description,
		OwnerID:     ownerID,
		Visibility:  ns.Spec.Visibility,
		MaxMembers:  int32(ns.Spec.MaxMembers),
		Status:      ns.Spec.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	return namespaceToAPI(created), nil
}

func (n *NamespaceService) GetNamespace(ctx context.Context, id string) (runtime.Object, error) {
	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
	}

	ns, err := n.nsStore.GetByID(ctx, nid)
	if err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}

	return namespaceToAPI(ns), nil
}

func (n *NamespaceService) ListNamespaces(ctx context.Context, filters map[string]string, pagination store.Pagination, sortBy, sortOrder string) (runtime.Object, error) {
	query := store.ListQuery{
		Filters:    make(map[string]any),
		Pagination: pagination,
	}
	for k, v := range filters {
		query.Filters[k] = v
	}
	if sortBy != "" {
		query.SortBy = sortBy
	}
	if sortOrder != "" {
		query.SortOrder = sortOrder
	}

	result, err := n.nsStore.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	items := make([]Namespace, len(result.Items))
	for i, item := range result.Items {
		items[i] = *namespaceToAPI(&item.Namespace)
	}

	return &NamespaceList{
		TypeMeta:   runtime.TypeMeta{Kind: "NamespaceList", APIVersion: "v1"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

func (n *NamespaceService) UpdateNamespace(ctx context.Context, id string, ns *Namespace) (*Namespace, error) {
	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
	}

	ownerID, err := parseID(ns.Spec.OwnerID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
	}

	updated, err := n.nsStore.Update(ctx, &nsstore.Namespace{
		ID:          nid,
		Name:        ns.ObjectMeta.Name,
		DisplayName: ns.Spec.DisplayName,
		Description: ns.Spec.Description,
		OwnerID:     ownerID,
		Visibility:  ns.Spec.Visibility,
		MaxMembers:  int32(ns.Spec.MaxMembers),
		Status:      ns.Spec.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
	}

	return namespaceToAPI(updated), nil
}

func (n *NamespaceService) PatchNamespace(ctx context.Context, id string, ns *Namespace) (*Namespace, error) {
	nid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
	}

	// For patch, fetch existing and merge
	existing, err := n.nsStore.GetByID(ctx, nid)
	if err != nil {
		return nil, fmt.Errorf("get namespace for patch: %w", err)
	}

	// Merge non-zero fields
	if ns.ObjectMeta.Name != "" {
		existing.Name = ns.ObjectMeta.Name
	}
	if ns.Spec.DisplayName != "" {
		existing.DisplayName = ns.Spec.DisplayName
	}
	if ns.Spec.Description != "" {
		existing.Description = ns.Spec.Description
	}
	if ns.Spec.OwnerID != "" {
		ownerID, err := parseID(ns.Spec.OwnerID)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
		}
		existing.OwnerID = ownerID
	}
	if ns.Spec.Visibility != "" {
		existing.Visibility = ns.Spec.Visibility
	}
	if ns.Spec.MaxMembers != 0 {
		existing.MaxMembers = int32(ns.Spec.MaxMembers)
	}
	if ns.Spec.Status != "" {
		existing.Status = ns.Spec.Status
	}

	updated, err := n.nsStore.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("patch namespace: %w", err)
	}

	return namespaceToAPI(updated), nil
}

func (n *NamespaceService) DeleteNamespace(ctx context.Context, id string) error {
	nid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", id), nil)
	}

	if err := n.nsStore.Delete(ctx, nid); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

func (n *NamespaceService) AddMember(ctx context.Context, namespaceID string, member *NamespaceMember) (*NamespaceMember, error) {
	nsID, err := parseID(namespaceID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", namespaceID), nil)
	}

	userID, err := parseID(member.Spec.UserID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid userId: %s", member.Spec.UserID), nil)
	}

	if err := n.userLookup.UserExists(ctx, userID); err != nil {
		return nil, apierrors.NewNotFound("User", member.Spec.UserID)
	}

	if _, err := n.nsStore.GetByID(ctx, nsID); err != nil {
		return nil, apierrors.NewNotFound("Namespace", namespaceID)
	}

	role, err := n.unStore.Add(ctx, &nsstore.UserNamespaceRole{
		UserID:      userID,
		NamespaceID: nsID,
		Role:        member.Spec.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	return memberToAPI(role), nil
}

func (n *NamespaceService) ListMembers(ctx context.Context, namespaceID string) (runtime.Object, error) {
	nsID, err := parseID(namespaceID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", namespaceID), nil)
	}

	members, err := n.unStore.ListByNamespaceID(ctx, nsID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	items := make([]NamespaceMember, len(members))
	for i, m := range members {
		items[i] = NamespaceMember{
			TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "NamespaceMember"},
			Spec: NamespaceMemberSpec{
				UserID: fmt.Sprintf("%d", m.User.ID),
				Role:   m.Role,
			},
		}
	}

	return &NamespaceMemberList{
		TypeMeta: runtime.TypeMeta{Kind: "NamespaceMemberList", APIVersion: "v1"},
		Items:    items,
	}, nil
}
