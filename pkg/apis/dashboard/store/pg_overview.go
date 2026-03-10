package store

import (
	"context"

	"lcp.io/lcp/pkg/apis/dashboard"
	"lcp.io/lcp/pkg/db/generated"
)

type pgOverviewStore struct {
	queries *generated.Queries
}

// NewPGOverviewStore creates a PostgreSQL-backed OverviewStore.
func NewPGOverviewStore(queries *generated.Queries) dashboard.OverviewStore {
	return &pgOverviewStore{queries: queries}
}

func (s *pgOverviewStore) GetPlatformStats(ctx context.Context) (*dashboard.DBPlatformStats, error) {
	row, err := s.queries.GetPlatformStats(ctx)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *pgOverviewStore) GetWorkspaceStats(ctx context.Context, workspaceID int64) (*dashboard.DBWorkspaceStats, error) {
	row, err := s.queries.GetWorkspaceStats(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *pgOverviewStore) GetNamespaceStats(ctx context.Context, namespaceID int64) (*dashboard.DBNamespaceStats, error) {
	row, err := s.queries.GetNamespaceStats(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}
