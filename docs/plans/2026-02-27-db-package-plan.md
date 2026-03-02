# lib/db Package Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a PostgreSQL database access layer under `lib/db` using sqlc + pgx/v5, with User/Namespace CRUD, many-to-many relationships, and complex query support (filtering, sorting, pagination, joins).

**Architecture:** Single `lib/db` package with `schema/` for DDL, `query/` for sqlc SQL files, `generated/` for sqlc output, and `db.go` for connection pool management. All queries use explicit field lists. Complex list queries use `sqlc.narg` for optional filters and `CASE WHEN` for dynamic sorting.

**Tech Stack:** Go 1.26, pgx/v5 (pgxpool), sqlc v2

**Design doc:** `docs/plans/2026-02-27-db-package-design.md`

---

### Task 1: Add pgx/v5 dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add pgx/v5 module**

Run:
```bash
cd /Users/zhouleyan/Projects/lcp && go get github.com/jackc/pgx/v5
```

**Step 2: Verify dependency added**

Run:
```bash
grep pgx go.mod
```
Expected: `github.com/jackc/pgx/v5 v5.x.x`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add pgx/v5 for PostgreSQL support"
```

---

### Task 2: Create schema SQL

**Files:**
- Create: `lib/db/schema/schema.sql`

**Step 1: Write the schema file**

Create `lib/db/schema/schema.sql` with the following content:

```sql
-- users table
CREATE TABLE users (
    id            BIGSERIAL    PRIMARY KEY,
    username      VARCHAR(255) NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    display_name  VARCHAR(255) NOT NULL DEFAULT '',
    phone         VARCHAR(50)  NOT NULL DEFAULT '',
    avatar_url    VARCHAR(512) NOT NULL DEFAULT '',
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_display_name ON users(display_name);

-- namespaces table
CREATE TABLE namespaces (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    owner_id     BIGINT       NOT NULL REFERENCES users(id),
    visibility   VARCHAR(20)  NOT NULL DEFAULT 'private',
    max_members  INT          NOT NULL DEFAULT 0,
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_namespaces_owner_id ON namespaces(owner_id);
CREATE INDEX idx_namespaces_status ON namespaces(status);
CREATE INDEX idx_namespaces_visibility ON namespaces(visibility);
CREATE INDEX idx_namespaces_created_at ON namespaces(created_at);

-- user_namespaces join table (many-to-many)
CREATE TABLE user_namespaces (
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    namespace_id BIGINT      NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    role         VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, namespace_id)
);

CREATE INDEX idx_user_namespaces_namespace_id ON user_namespaces(namespace_id);
CREATE INDEX idx_user_namespaces_role ON user_namespaces(role);
```

**Step 2: Commit**

```bash
git add lib/db/schema/schema.sql
git commit -m "feat(db): add PostgreSQL schema for users, namespaces, user_namespaces"
```

---

### Task 3: Create sqlc configuration

**Files:**
- Create: `lib/db/sqlc.yaml`

**Step 1: Write sqlc config**

Create `lib/db/sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "query/"
    schema: "schema/"
    gen:
      go:
        package: "generated"
        out: "generated"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true
```

**Step 2: Commit**

```bash
git add lib/db/sqlc.yaml
git commit -m "feat(db): add sqlc v2 configuration"
```

---

### Task 4: Write User query SQL

**Files:**
- Create: `lib/db/query/user.sql`

**Step 1: Write user queries**

Create `lib/db/query/user.sql` with all User CRUD + list queries. Every query uses explicit field lists.

```sql
-- name: CreateUser :one
INSERT INTO users (username, email, display_name, phone, avatar_url, status)
VALUES (@username, @email, @display_name, @phone, @avatar_url, @status)
RETURNING id, username, email, display_name, phone, avatar_url, status,
          last_login_at, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, email, display_name, phone, avatar_url, status,
       last_login_at, created_at, updated_at
FROM users
WHERE id = @id;

-- name: GetUserByUsername :one
SELECT id, username, email, display_name, phone, avatar_url, status,
       last_login_at, created_at, updated_at
FROM users
WHERE username = @username;

-- name: GetUserByEmail :one
SELECT id, username, email, display_name, phone, avatar_url, status,
       last_login_at, created_at, updated_at
FROM users
WHERE email = @email;

-- name: UpdateUser :one
UPDATE users
SET username = @username,
    email = @email,
    display_name = @display_name,
    phone = @phone,
    avatar_url = @avatar_url,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING id, username, email, display_name, phone, avatar_url, status,
          last_login_at, created_at, updated_at;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = now(), updated_at = now()
WHERE id = @id;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = @id;

-- name: CountUsers :one
SELECT count(id)
FROM users
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('username')::VARCHAR IS NULL OR username ILIKE '%' || sqlc.narg('username') || '%')
    AND (sqlc.narg('email')::VARCHAR IS NULL OR email ILIKE '%' || sqlc.narg('email') || '%')
    AND (sqlc.narg('display_name')::VARCHAR IS NULL OR display_name ILIKE '%' || sqlc.narg('display_name') || '%');

-- name: ListUsers :many
SELECT
    u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
    u.status, u.last_login_at, u.created_at, u.updated_at,
    COALESCE(
        array_agg(DISTINCT n.name) FILTER (WHERE n.name IS NOT NULL),
        '{}'
    )::TEXT[] AS namespace_names
FROM users u
LEFT JOIN user_namespaces un ON u.id = un.user_id
LEFT JOIN namespaces n ON un.namespace_id = n.id
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
    AND (sqlc.narg('username')::VARCHAR IS NULL OR u.username ILIKE '%' || sqlc.narg('username') || '%')
    AND (sqlc.narg('email')::VARCHAR IS NULL OR u.email ILIKE '%' || sqlc.narg('email') || '%')
    AND (sqlc.narg('display_name')::VARCHAR IS NULL OR u.display_name ILIKE '%' || sqlc.narg('display_name') || '%')
GROUP BY u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
         u.status, u.last_login_at, u.created_at, u.updated_at
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.username END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.username END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'email' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.email END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'email' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.email END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.display_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.display_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.status END DESC,
    u.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
```

**Step 2: Commit**

```bash
git add lib/db/query/user.sql
git commit -m "feat(db): add sqlc queries for User CRUD and list"
```

---

### Task 5: Write Namespace query SQL

**Files:**
- Create: `lib/db/query/namespace.sql`

**Step 1: Write namespace queries**

Create `lib/db/query/namespace.sql`:

```sql
-- name: CreateNamespace :one
INSERT INTO namespaces (name, display_name, description, owner_id, visibility, max_members, status)
VALUES (@name, @display_name, @description, @owner_id, @visibility, @max_members, @status)
RETURNING id, name, display_name, description, owner_id, visibility, max_members, status,
          created_at, updated_at;

-- name: GetNamespaceByID :one
SELECT id, name, display_name, description, owner_id, visibility, max_members, status,
       created_at, updated_at
FROM namespaces
WHERE id = @id;

-- name: GetNamespaceByName :one
SELECT id, name, display_name, description, owner_id, visibility, max_members, status,
       created_at, updated_at
FROM namespaces
WHERE name = @name;

-- name: UpdateNamespace :one
UPDATE namespaces
SET name = @name,
    display_name = @display_name,
    description = @description,
    owner_id = @owner_id,
    visibility = @visibility,
    max_members = @max_members,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, owner_id, visibility, max_members, status,
          created_at, updated_at;

-- name: DeleteNamespace :exec
DELETE FROM namespaces WHERE id = @id;

-- name: CountNamespaces :one
SELECT count(id)
FROM namespaces
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('name')::VARCHAR IS NULL OR name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('visibility')::VARCHAR IS NULL OR visibility = sqlc.narg('visibility'))
    AND (sqlc.narg('owner_id')::BIGINT IS NULL OR owner_id = sqlc.narg('owner_id'));

-- name: ListNamespaces :many
SELECT
    ns.id, ns.name, ns.display_name, ns.description, ns.owner_id,
    ns.visibility, ns.max_members, ns.status, ns.created_at, ns.updated_at,
    u.username AS owner_username
FROM namespaces ns
JOIN users u ON ns.owner_id = u.id
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR ns.status = sqlc.narg('status'))
    AND (sqlc.narg('name')::VARCHAR IS NULL OR ns.name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('visibility')::VARCHAR IS NULL OR ns.visibility = sqlc.narg('visibility'))
    AND (sqlc.narg('owner_id')::BIGINT IS NULL OR ns.owner_id = sqlc.narg('owner_id'))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'visibility' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.visibility END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'visibility' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.visibility END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.status END DESC,
    ns.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
```

**Step 2: Commit**

```bash
git add lib/db/query/namespace.sql
git commit -m "feat(db): add sqlc queries for Namespace CRUD and list"
```

---

### Task 6: Write UserNamespace query SQL

**Files:**
- Create: `lib/db/query/user_namespace.sql`

**Step 1: Write user_namespace queries**

Create `lib/db/query/user_namespace.sql`:

```sql
-- name: AddUserToNamespace :one
INSERT INTO user_namespaces (user_id, namespace_id, role)
VALUES (@user_id, @namespace_id, @role)
RETURNING user_id, namespace_id, role, created_at;

-- name: RemoveUserFromNamespace :exec
DELETE FROM user_namespaces
WHERE user_id = @user_id AND namespace_id = @namespace_id;

-- name: UpdateUserNamespaceRole :one
UPDATE user_namespaces
SET role = @role
WHERE user_id = @user_id AND namespace_id = @namespace_id
RETURNING user_id, namespace_id, role, created_at;

-- name: GetUserNamespace :one
SELECT user_id, namespace_id, role, created_at
FROM user_namespaces
WHERE user_id = @user_id AND namespace_id = @namespace_id;

-- name: ListNamespacesByUserID :many
SELECT
    n.id, n.name, n.display_name, n.description, n.owner_id,
    n.visibility, n.max_members, n.status, n.created_at, n.updated_at,
    un.role, un.created_at AS joined_at
FROM namespaces n
JOIN user_namespaces un ON n.id = un.namespace_id
WHERE un.user_id = @user_id
ORDER BY un.created_at DESC;

-- name: ListUsersByNamespaceID :many
SELECT
    u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
    u.status, u.last_login_at, u.created_at, u.updated_at,
    un.role, un.created_at AS joined_at
FROM users u
JOIN user_namespaces un ON u.id = un.user_id
WHERE un.namespace_id = @namespace_id
ORDER BY un.created_at DESC;

-- name: CountUsersByNamespaceID :one
SELECT count(user_id)
FROM user_namespaces
WHERE namespace_id = @namespace_id;

-- name: CountNamespacesByUserID :one
SELECT count(namespace_id)
FROM user_namespaces
WHERE user_id = @user_id;
```

**Step 2: Commit**

```bash
git add lib/db/query/user_namespace.sql
git commit -m "feat(db): add sqlc queries for user-namespace relationships"
```

---

### Task 7: Generate sqlc code

**Files:**
- Generated: `lib/db/generated/*.go`

**Step 1: Install sqlc (if not present)**

Run:
```bash
which sqlc || go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**Step 2: Run sqlc generate**

Run:
```bash
cd /Users/zhouleyan/Projects/lcp/lib/db && sqlc generate
```
Expected: No errors. Files created in `lib/db/generated/`.

**Step 3: Verify generated files**

Run:
```bash
ls -la /Users/zhouleyan/Projects/lcp/lib/db/generated/
```
Expected: `db.go`, `models.go`, `querier.go`, `user.sql.go`, `namespace.sql.go`, `user_namespace.sql.go`

**Step 4: Verify generated code compiles**

Run:
```bash
cd /Users/zhouleyan/Projects/lcp && go build ./lib/db/generated/
```
Expected: No errors.

**Step 5: Commit**

```bash
git add lib/db/generated/
git commit -m "feat(db): generate sqlc code for all queries"
```

---

### Task 8: Write connection pool manager

**Files:**
- Create: `lib/db/db.go`

**Step 1: Write db.go**

Create `lib/db/db.go`:

```go
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

	pool, err := pgxpool.New(ctx, poolCfg.ConnString())
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
```

**Step 2: Verify it compiles**

Run:
```bash
cd /Users/zhouleyan/Projects/lcp && go build ./lib/db/
```
Expected: No errors.

**Step 3: Commit**

```bash
git add lib/db/db.go
git commit -m "feat(db): add connection pool manager with pgxpool"
```

---

### Task 9: Add sqlc generate to Makefile

**Files:**
- Modify: `Makefile`

**Step 1: Add sqlc-generate target**

Add to `Makefile` after the existing `lcp-server` target:

```makefile
sqlc-generate:
	cd lib/db && sqlc generate
```

**Step 2: Commit**

```bash
git add Makefile
git commit -m "build: add sqlc-generate target to Makefile"
```

---

### Task 10: Final verification

**Step 1: Run go vet on the entire db package**

Run:
```bash
cd /Users/zhouleyan/Projects/lcp && go vet ./lib/db/...
```
Expected: No errors.

**Step 2: Run go build on the entire project**

Run:
```bash
cd /Users/zhouleyan/Projects/lcp && go build ./...
```
Expected: No errors.

**Step 3: Review generated models match schema**

Run:
```bash
cat /Users/zhouleyan/Projects/lcp/lib/db/generated/models.go
```
Expected: `User`, `Namespace`, `UserNamespace` structs with correct field types and JSON tags.
