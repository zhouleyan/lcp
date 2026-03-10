package dashboard

import (
	"context"
	"fmt"
	"strconv"

	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	apierrors "lcp.io/lcp/lib/api/errors"
)

// OverviewStore abstracts the database queries for dashboard statistics.
type OverviewStore interface {
	GetPlatformStats(ctx context.Context) (*DBPlatformStats, error)
	GetWorkspaceStats(ctx context.Context, workspaceID int64) (*DBWorkspaceStats, error)
	GetNamespaceStats(ctx context.Context, namespaceID int64) (*DBNamespaceStats, error)
}

// --- Platform Overview Storage ---

type platformOverviewStorage struct {
	store OverviewStore
}

// NewPlatformOverviewStorage creates a Lister that returns platform-level statistics.
// +openapi:summary=获取平台概览统计
func NewPlatformOverviewStorage(store OverviewStore) rest.Lister {
	return &platformOverviewStorage{store: store}
}

func (s *platformOverviewStorage) List(ctx context.Context, _ *rest.ListOptions) (runtime.Object, error) {
	stats, err := s.store.GetPlatformStats(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	return &PlatformOverview{
		Spec: PlatformOverviewSpec{
			WorkspaceCount: stats.WorkspaceCount,
			NamespaceCount: stats.NamespaceCount,
			UserCount:      stats.UserCount,
			RoleCount:      stats.RoleCount,
		},
	}, nil
}

// --- Workspace Overview Storage ---

type workspaceOverviewStorage struct {
	store OverviewStore
}

// NewWorkspaceOverviewStorage creates a Lister that returns workspace-level statistics.
// +openapi:summary=获取租户概览统计
func NewWorkspaceOverviewStorage(store OverviewStore) rest.Lister {
	return &workspaceOverviewStorage{store: store}
}

func (s *workspaceOverviewStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	wsID, err := parseID(options.PathParams["workspaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid workspace ID: %s", options.PathParams["workspaceId"]), nil)
	}
	stats, err := s.store.GetWorkspaceStats(ctx, wsID)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	return &WorkspaceOverview{
		Spec: WorkspaceOverviewSpec{
			NamespaceCount: stats.NamespaceCount,
			MemberCount:    stats.MemberCount,
			RoleCount:      stats.RoleCount,
		},
	}, nil
}

// --- Namespace Overview Storage ---

type namespaceOverviewStorage struct {
	store OverviewStore
}

// NewNamespaceOverviewStorage creates a Lister that returns namespace-level statistics.
// +openapi:summary=获取项目概览统计
func NewNamespaceOverviewStorage(store OverviewStore) rest.Lister {
	return &namespaceOverviewStorage{store: store}
}

func (s *namespaceOverviewStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	nsID, err := parseID(options.PathParams["namespaceId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid namespace ID: %s", options.PathParams["namespaceId"]), nil)
	}
	stats, err := s.store.GetNamespaceStats(ctx, nsID)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	return &NamespaceOverview{
		Spec: NamespaceOverviewSpec{
			MemberCount: stats.MemberCount,
			RoleCount:   stats.RoleCount,
		},
	}, nil
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
