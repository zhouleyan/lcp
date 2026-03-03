package namespace

import (
	"context"
	"fmt"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/store"
)

// namespaceStorage implements rest.Getter, rest.Lister, rest.Creator, rest.Updater, rest.Patcher, rest.Deleter.
type namespaceStorage struct {
	svc *NamespaceService
}

func newNamespaceStorage(svc *NamespaceService) *namespaceStorage {
	return &namespaceStorage{svc: svc}
}

func (s *namespaceStorage) Get(ctx context.Context, id string) (runtime.Object, error) {
	return s.svc.GetNamespace(ctx, id)
}

func (s *namespaceStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	pagination := store.Pagination{
		Page:     options.Pagination.Page,
		PageSize: options.Pagination.PageSize,
	}
	return s.svc.ListNamespaces(ctx, options.Filters, pagination, options.SortBy, string(options.SortOrder))
}

func (s *namespaceStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return nil, fmt.Errorf("expected *Namespace, got %T", obj)
	}

	if errs := ValidateNamespaceCreate(ns.ObjectMeta.Name, &ns.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return ns, nil
	}

	return s.svc.CreateNamespace(ctx, ns)
}

func (s *namespaceStorage) Update(ctx context.Context, id string, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return nil, fmt.Errorf("expected *Namespace, got %T", obj)
	}

	if options.DryRun {
		return ns, nil
	}

	return s.svc.UpdateNamespace(ctx, id, ns)
}

func (s *namespaceStorage) Patch(ctx context.Context, id string, obj runtime.Object, options *rest.PatchOptions) (runtime.Object, error) {
	ns, ok := obj.(*Namespace)
	if !ok {
		return nil, fmt.Errorf("expected *Namespace, got %T", obj)
	}

	if options.DryRun {
		existing, err := s.svc.GetNamespace(ctx, id)
		if err != nil {
			return nil, err
		}
		return existing, nil
	}

	return s.svc.PatchNamespace(ctx, id, ns)
}

func (s *namespaceStorage) Delete(ctx context.Context, id string, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}
	return s.svc.DeleteNamespace(ctx, id)
}

// memberStorage implements rest.Creator and rest.Lister for the members sub-resource.
type memberStorage struct {
	svc *NamespaceService
}

func newMemberStorage(svc *NamespaceService) *memberStorage {
	return &memberStorage{svc: svc}
}

func (s *memberStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	params := rest.PathParamsFromContext(ctx)
	namespaceID := params["namespaceId"]
	if namespaceID == "" {
		return nil, apierrors.NewBadRequest("missing namespaceId in path", nil)
	}

	member, ok := obj.(*NamespaceMember)
	if !ok {
		return nil, fmt.Errorf("expected *NamespaceMember, got %T", obj)
	}

	if errs := ValidateNamespaceMember(&member.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return member, nil
	}

	return s.svc.AddMember(ctx, namespaceID, member)
}

func (s *memberStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	params := rest.PathParamsFromContext(ctx)
	namespaceID := params["namespaceId"]
	if namespaceID == "" {
		return nil, apierrors.NewBadRequest("missing namespaceId in path", nil)
	}

	return s.svc.ListMembers(ctx, namespaceID)
}
