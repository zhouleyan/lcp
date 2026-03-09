# CLAUDE.md

## Project Overview

LCP is a PaaS management platform built in Go. Module: `lcp.io/lcp`, Go 1.26.0.

The project has a custom-built REST API framework (inspired by Kubernetes apiserver patterns) with PostgreSQL as the data store, using sqlc for type-safe query generation.

## Directory Structure

```
app/lcp-server/       # Main server entry point (config, wiring, HTTP listener)
lib/                  # Internal framework libraries (REST framework, runtime, config, logger, etc.)
lib/oidc/             # OIDC provider: JWT tokens, auth codes, sessions, keys, password hashing
lib/httpserver/filters/ # HTTP middleware (request logging, authentication)
pkg/apis/             # Business logic: API types, store interfaces, REST storage, validation
pkg/apis/iam/         # IAM module: users, workspaces, namespaces, memberships, OIDC handlers
pkg/apis/iam/store/   # PostgreSQL store implementations (pg_*.go)
pkg/apis/iam/v1/      # Route registration (install.go)
pkg/db/               # Database: connection pool, pagination helpers, sqlc config
pkg/db/schema/        # PostgreSQL DDL (schema.sql)
pkg/db/query/         # sqlc SQL query files (*.sql)
pkg/db/generated/     # sqlc auto-generated Go code (DO NOT EDIT)
cmd/openapi-gen/      # OpenAPI spec generator from +openapi: annotations
docs/                 # Generated OpenAPI specs, design docs
scripts/              # Utility scripts (e.g. seed-test-users.sh)
```

## Key Commands

```bash
make lcp-server          # Dev build with -race
make lcp-server-prod     # Production build (CGO_ENABLED=0)
make sqlc-generate       # Regenerate Go code from SQL queries
make openapi-gen         # Generate OpenAPI JSON + YAML specs
make test                # go test ./...
make vet                 # go vet ./...
make lint                # golangci-lint run ./...
make fmt                 # gofmt -w -s .
```

Run the server:
```bash
go run ./app/lcp-server/ -config ./app/lcp-server/config.yaml
# Listens on :8428 by default
```

Database is PostgreSQL, configured in `app/lcp-server/config.yaml`. Local dev uses Docker container `lcp-postgres` (user: lcp, password: lcp, db: lcp).

## Architecture Patterns

### Layered Architecture

```
HTTP Request
  → lib/rest (routing, content negotiation, handler dispatch)
    → pkg/apis/iam/storage.go (REST storage: validation, type conversion, orchestration)
      → pkg/apis/iam/store.go (Store interface)
        → pkg/apis/iam/store/pg_*.go (PostgreSQL implementation via sqlc)
          → pkg/db/generated/ (sqlc-generated queries)
```

### Module Registration & Assembly Rules

- **`apis.Result`** only contains `Groups []*rest.APIGroupInfo` — no stores, caches, authorizers, or other implementation details. `NewAPIGroupInfos` is purely for module registration.
- **`v1.ModuleResult`** only contains `Group *rest.APIGroupInfo` — same principle at the module level.
- **`main.go`** must not import `iam`, `iamstore`, or any internal package. It only calls `apis.*` and `handler.*` factory functions.
- **`apis` package** is the assembly/bridge layer. Cross-cutting concerns (shared caches, authorizer wiring) live here as package-level state, not in `Result`.
- **`handler` package** owns routing logic (`NewRootHandler`, `buildChain`), not main.go.

### Adding a New Resource (Checklist)

1. **Schema**: Add table to `pkg/db/schema/schema.sql`
2. **Queries**: Create `pkg/db/query/<resource>.sql` with sqlc annotations
3. **Generate**: Run `make sqlc-generate`
4. **Types**: Add API types + DB type aliases in `pkg/apis/iam/types.go`
5. **Store interface**: Define in `pkg/apis/iam/store.go`
6. **Store impl**: Create `pkg/apis/iam/store/pg_<resource>.go`
7. **Validation**: Add to `pkg/apis/iam/validation.go`
8. **REST storage**: Implement in `pkg/apis/iam/storage.go`
9. **Provider**: Add store field + accessor in `pkg/apis/iam/provider.go`
10. **Routes**: Register in `pkg/apis/iam/v1/install.go`
11. **Wiring**: Update `pkg/apis/install.go` with concrete store instantiation

### REST Framework Conventions

- **Storage interfaces**: Implement `rest.Getter`, `rest.Lister`, `rest.Creator`, `rest.Updater`, `rest.Patcher`, `rest.Deleter`, `rest.CollectionDeleter` — or combine as `rest.StandardStorage`
- **All API objects** must implement `runtime.Object` (embed `runtime.TypeMeta`, implement `GetTypeMeta()`)
- **URL path params** are auto-derived from resource names: `"users"` → `{userId}`, `"workspaces"` → `{workspaceId}`
- **Sub-resources** are nested via `ResourceInfo.SubResources` (supports recursive nesting)
- **IDs**: `int64` (BIGSERIAL) in DB, `string` in API layer. Conversion via `strconv.FormatInt` / `strconv.ParseInt`
- **List queries**: Use `db.ListQuery{Filters, Pagination}` → `db.ListResult[T]{Items, TotalCount}`
- **Pagination**: `page` (1-based), `pageSize` (default 20, max 100), `sortBy`, `sortOrder`
- **Batch operations**: Use `BatchRequest{IDs []string}` for batch add; `rest.DeleteCollectionRequest{IDs}` for batch delete
- **Content negotiation**: JSON (default) + YAML (via `Accept: application/yaml`)

### Error Handling

```go
apierrors.NewBadRequest(message, details)   // 400
apierrors.NewNotFound(resource, name)       // 404
apierrors.NewConflict(resource, name)       // 409
apierrors.NewInternalError(err)             // 500
```

Errors serialize as `{apiVersion, kind: "Status", status, reason, message}`.

### Validation Pattern

```go
var nameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

func ValidateXxxCreate(name string, spec *XxxSpec) validation.ErrorList {
    var errs validation.ErrorList
    // field checks, append validation.FieldError{Field, Message}
    return errs
}
```

### Store Implementation Pattern

- Use `pgxpool.Pool` + `generated.Queries` for DB access
- Transactional operations: `pool.Begin()` → `queries.WithTx(tx)` → operations → `tx.Commit()`
- Handle `pgx.ErrNoRows` → `apierrors.NewNotFound()`
- List methods: use `filterStr()`/`filterInt64()` helpers to convert `map[string]any` filters to sqlc nullable params

### sqlc Query Pattern

```sql
-- name: ListXxx :many
SELECT ... FROM xxx
WHERE (sqlc.narg('filter')::TYPE IS NULL OR column = sqlc.narg('filter'))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    ...
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
```

### API Type Pattern

```go
type Xxx struct {
    runtime.TypeMeta `json:",inline"`
    types.ObjectMeta `json:"metadata"`
    Spec             XxxSpec `json:"spec"`
}
func (x *Xxx) GetTypeMeta() *runtime.TypeMeta { return &x.TypeMeta }
```

### OpenAPI Annotation Pattern

Annotations are split across types and storage methods:

**types.go** — resource description + field-level annotations only:
```go
// Xxx
// +openapi:description=资源描述
type Xxx struct { ... }

type XxxSpec struct {
    // +openapi:required
    // +openapi:description=字段描述
    // +openapi:enum=active,inactive
    // +openapi:format=email
    Field string `json:"field"`
}
```

**storage.go** — operation summaries on methods, paths auto-derived from storage type name:
```go
// Storage type name convention: {parent}{resource}Storage
// - userStorage       → resource=User,  path=/users
// - workspaceStorage  → resource=Workspace, path=/workspaces
// - workspaceUserStorage → resource=User, path=/workspaces/{workspaceId}/users

// Extra paths declared on the struct (auto-derived primary path needs no annotation):
// +openapi:path=/workspaces/{workspaceId}/namespaces
type namespaceStorage struct { ... }

// Simple summary (applies to auto-derived path):
// +openapi:summary=获取项目列表
// Qualified summary (applies to the extra path):
// +openapi:summary.workspaces.namespaces=获取工作空间下的项目列表
func (s *namespaceStorage) List(...) { ... }

// Standalone action function:
// +openapi:action=change-password
// +openapi:resource=User
// +openapi:summary=修改用户密码
func NewChangePasswordHandler(...) rest.HandlerFunc { ... }

// Custom verb (read-only list view on resource item):
// +openapi:customverb=workspaces
// +openapi:resource=User
// +openapi:summary=获取用户关联的工作空间列表
func NewUserWorkspacesVerb(...) rest.Lister { ... }
// → generates GET /users/{userId}:workspaces returning WorkspaceList
```

Method name → operation mapping: `List→list`, `Create→create`, `Get→get`, `Update→update`, `Patch→patch`, `Delete→delete`, `DeleteCollection→deleteCollection`.

## Current API Routes

```
# OIDC (public, no authentication)
GET  /.well-known/openid-configuration                            # OIDC discovery
GET  /.well-known/jwks.json                                       # JSON Web Key Set
GET  /oidc/authorize                                              # Authorization endpoint
POST /oidc/login                                                  # Login (username+password)
POST /oidc/token                                                  # Token exchange
GET  /oidc/userinfo                                               # User info

# Business API (authenticated via Bearer token when OIDC is enabled)
# Authentication middleware checks both token validity AND user active status on every request.
# Inactive users receive 401 even with a valid token. Token refresh is also blocked for inactive users.
/api/iam/v1/users                                                    # CRUD + batch delete
/api/iam/v1/users/{userId}/change-password                           # POST change password
/api/iam/v1/users/{userId}:workspaces                                # GET user's joined workspaces (paginated)
/api/iam/v1/users/{userId}:namespaces                                # GET user's joined namespaces (paginated)
/api/iam/v1/workspaces                                               # CRUD + batch delete
/api/iam/v1/workspaces/{workspaceId}/namespaces                      # CRUD + batch delete
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/users  # list + batch add/remove
/api/iam/v1/workspaces/{workspaceId}/users                           # list + batch add/remove
/api/iam/v1/namespaces                                               # CRUD + batch delete
/api/iam/v1/namespaces/{namespaceId}/users                           # list + batch add/remove
```

## Resource Hierarchy

```
Workspace (tenant/organization)
  └── Namespace (project/team scope)
       └── User (member with role)
```

- Creating a Workspace auto-creates a default Namespace and adds owner as member
- Adding a User to a Namespace auto-adds them to the parent Workspace
- Deleting a Workspace requires no child Namespaces
- Deleting a Namespace requires no member Users

## Testing

- Unit tests live in `lib/` (standard `testing` package + `httptest`)
- No test files under `pkg/` or `app/` currently
- E2E testing: start server, use `curl` against `localhost:8428`

## Configuration

Priority: CLI flags > env vars > `config.yaml` > defaults. Supports SIGHUP hot-reload.

Key env vars: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE`, `DB_MAX_CONNS`.

### OIDC Configuration

OIDC is enabled by providing ECDSA P-256 key file paths in `config.yaml`. Without keys, authentication is disabled and all API endpoints are open.

```yaml
oidc:
  issuer: "http://localhost:8428"
  privateKeyFile: "./oidc-private.pem"
  publicKeyFile: "./oidc-public.pem"
  accessTokenTTL: "1h"
  refreshTokenTTL: "168h"
  authCodeTTL: "5m"
  loginUrl: "/login"
  clients:
    - id: "lcp-ui"
      public: true
      redirectUris: ["http://localhost:5173/auth/callback"]
      scopes: ["openid", "profile", "email", "phone"]
```

Generate keys: `openssl ecparam -name prime256v1 -genkey -noout -out oidc-private.pem && openssl ec -in oidc-private.pem -pubout -out oidc-public.pem`
