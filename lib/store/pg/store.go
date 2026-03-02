package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

// pgStore implements store.Store backed by PostgreSQL.
type pgStore struct {
	pool    *pgxpool.Pool
	db      generated.DBTX
	queries *generated.Queries
}

// New creates a new PostgreSQL-backed Store.
func New(pool *pgxpool.Pool) store.Store {
	return &pgStore{
		pool:    pool,
		db:      pool,
		queries: generated.New(pool),
	}
}

func (s *pgStore) Users() store.UserStore {
	return &pgUserStore{db: s.db, queries: s.queries}
}

func (s *pgStore) Namespaces() store.NamespaceStore {
	return &pgNamespaceStore{db: s.db, queries: s.queries}
}

func (s *pgStore) UserNamespaces() store.UserNamespaceStore {
	return &pgUserNamespaceStore{queries: s.queries}
}

func (s *pgStore) WithTx(ctx context.Context, fn func(store.Store) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txStore := &pgStore{
		pool:    s.pool,
		db:      tx,
		queries: s.queries.WithTx(tx),
	}

	if err := fn(txStore); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *pgStore) Close() {
	s.pool.Close()
}

// paginationToOffsetLimit converts Pagination to offset and limit with defaults.
func paginationToOffsetLimit(p store.Pagination) (offset int32, limit int32) {
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return int32((page - 1) * pageSize), int32(pageSize)
}
