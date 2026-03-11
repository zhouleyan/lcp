package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/infra"
	"lcp.io/lcp/pkg/db/generated"
)

type pgHostAssignmentStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewPGHostAssignmentStore creates a new PostgreSQL-backed HostAssignmentStore.
func NewPGHostAssignmentStore(pool *pgxpool.Pool, queries *generated.Queries) infra.HostAssignmentStore {
	return &pgHostAssignmentStore{pool: pool, queries: queries}
}

func (s *pgHostAssignmentStore) Assign(ctx context.Context, hostID int64, wsID, nsID *int64) (*infra.DBHostAssignment, error) {
	row, err := s.queries.AssignHost(ctx, generated.AssignHostParams{
		HostID:      hostID,
		WorkspaceID: wsID,
		NamespaceID: nsID,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("host_assignment", fmt.Sprintf("host %d", hostID))
		}
		return nil, fmt.Errorf("assign host: %w", err)
	}
	return &row, nil
}

func (s *pgHostAssignmentStore) UnassignWorkspace(ctx context.Context, hostID int64, wsID int64) error {
	return s.queries.UnassignHostWorkspace(ctx, generated.UnassignHostWorkspaceParams{
		HostID:      hostID,
		WorkspaceID: &wsID,
	})
}

func (s *pgHostAssignmentStore) UnassignNamespace(ctx context.Context, hostID int64, nsID int64) error {
	return s.queries.UnassignHostNamespace(ctx, generated.UnassignHostNamespaceParams{
		HostID:      hostID,
		NamespaceID: &nsID,
	})
}

func (s *pgHostAssignmentStore) ListByHostID(ctx context.Context, hostID int64) ([]infra.DBAssignmentRow, error) {
	rows, err := s.queries.ListAssignmentsByHostID(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("list assignments by host id: %w", err)
	}
	return rows, nil
}
