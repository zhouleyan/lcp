package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"lcp.io/lcp/pkg/db/generated"
)

// EscapeLike escapes LIKE/ILIKE special characters (%, _, \) in a string
// so they are treated as literals in filter queries.
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// DB holds the database connection pool and query interface.
type DB struct {
	Pool    *pgxpool.Pool
	Queries *generated.Queries
}

// NewDB creates a DB instance: connection pool + sqlc queries.
func NewDB(ctx context.Context, cfg Config) (*DB, error) {
	pool, err := NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	queries := generated.New(pool)
	return &DB{Pool: pool, Queries: queries}, nil
}

// Close closes the underlying connection pool.
func (d *DB) Close() {
	d.Pool.Close()
}

// Reload creates a new connection pool with the given config, verifies it,
// then atomically replaces the old pool and queries. The old pool is closed
// after the swap.
func (d *DB) Reload(ctx context.Context, cfg Config) error {
	newPool, err := NewPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("reload database: %w", err)
	}
	oldPool := d.Pool
	d.Pool = newPool
	d.Queries = generated.New(newPool)
	oldPool.Close()
	return nil
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
