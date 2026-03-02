package service

import (
	"context"
	"fmt"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/api/validation"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/store"
)

type NamespaceService struct {
	s *Service
}

func (n *NamespaceService) CreateNamespace(ctx context.Context, ns *types.Namespace) (*types.Namespace, error) {
	if errs := validation.ValidateNamespaceCreate(ns.ObjectMeta.Name, &ns.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	ownerID, err := parseID(ns.Spec.OwnerID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ownerId: %s", ns.Spec.OwnerID), nil)
	}

	if _, err := n.s.store.Users().GetByID(ctx, ownerID); err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("owner user %d not found", ownerID), nil)
	}

	created, err := n.s.store.Namespaces().Create(ctx, &store.Namespace{
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

	ns, err := n.s.store.Namespaces().GetByID(ctx, nid)
	if err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}

	return namespaceToAPI(ns), nil
}

func (n *NamespaceService) AddMember(ctx context.Context, namespaceID string, member *types.NamespaceMember) (*types.NamespaceMember, error) {
	if errs := validation.ValidateNamespaceMember(&member.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	nsID, err := parseID(namespaceID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", namespaceID), nil)
	}

	userID, err := parseID(member.Spec.UserID)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid userId: %s", member.Spec.UserID), nil)
	}

	if _, err := n.s.store.Users().GetByID(ctx, userID); err != nil {
		return nil, apierrors.NewNotFound("User", member.Spec.UserID)
	}

	if _, err := n.s.store.Namespaces().GetByID(ctx, nsID); err != nil {
		return nil, apierrors.NewNotFound("Namespace", namespaceID)
	}

	role, err := n.s.store.UserNamespaces().Add(ctx, &store.UserNamespaceRole{
		UserID:      userID,
		NamespaceID: nsID,
		Role:        member.Spec.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	return memberToAPI(role), nil
}
