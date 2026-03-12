package db

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationLockID is a fixed advisory lock ID for migration coordination.
const migrationLockID = 1000000

// Migrate runs all pending migrations from the embedded filesystem.
// It uses a PostgreSQL advisory lock to ensure only one instance runs
// migrations at a time (safe for horizontal scaling).
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys embed.FS) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	// Advisory lock — blocks until acquired, auto-released when connection returns to pool.
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", migrationLockID)

	// Ensure schema_migrations table exists.
	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT      PRIMARY KEY,
			filename   TEXT        NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// Load applied versions.
	applied := make(map[int64]bool)
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("query applied versions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return fmt.Errorf("scan version: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate versions: %w", err)
	}

	// Discover migration files.
	entries, err := fsys.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migration directory: %w", err)
	}

	type migration struct {
		version  int64
		filename string
	}
	var pending []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		version, err := parseVersion(e.Name())
		if err != nil {
			return fmt.Errorf("parse migration filename %q: %w", e.Name(), err)
		}
		if !applied[version] {
			pending = append(pending, migration{version: version, filename: e.Name()})
		}
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].version < pending[j].version })

	if len(pending) == 0 {
		return nil
	}

	// Apply each pending migration in a transaction.
	for _, m := range pending {
		sql, err := fsys.ReadFile(m.filename)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", m.filename, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for %q: %w", m.filename, err)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("execute migration %q: %w", m.filename, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, filename) VALUES ($1, $2)",
			m.version, m.filename,
		); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("record migration %q: %w", m.filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %q: %w", m.filename, err)
		}
	}

	return nil
}

// parseVersion extracts the numeric version prefix from a migration filename.
// e.g. "000001_initial.up.sql" → 1
func parseVersion(filename string) (int64, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid migration filename: %s", filename)
	}
	return strconv.ParseInt(parts[0], 10, 64)
}
