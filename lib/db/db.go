package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sort field constants for User list queries.
const (
	UserSortUsername    = "username"
	UserSortEmail      = "email"
	UserSortDisplayName = "display_name"
	UserSortCreatedAt  = "created_at"
	UserSortStatus     = "status"
)

// Sort field constants for Namespace list queries.
const (
	NamespaceSortName       = "name"
	NamespaceSortCreatedAt  = "created_at"
	NamespaceSortVisibility = "visibility"
	NamespaceSortStatus     = "status"
)

// Sort order constants.
const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

// EscapeLike escapes LIKE/ILIKE special characters (%, _, \) in a string
// so they are treated as literals in filter queries.
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// Config holds PostgreSQL connection parameters.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int32
}

// DSN returns the PostgreSQL connection string.
func (c Config) DSN() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	port := c.Port
	if port == 0 {
		port = 5432
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, port, c.DBName, sslMode)
}

// NewPool creates a new pgx connection pool and verifies connectivity.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
