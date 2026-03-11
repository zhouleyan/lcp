# CLAUDE.md

## Project Overview

LCP is a PaaS management platform built in Go. Module: `lcp.io/lcp`, Go 1.26.0.

The project has a custom-built REST API framework (inspired by Kubernetes apiserver patterns) with PostgreSQL as the data store, using sqlc for type-safe query generation.

## Directory Structure

```
app/lcp-server/       # Main server entry point (config, wiring, HTTP listener)
lib/                  # Internal framework libraries (REST framework, runtime, config, logger, etc.)
lib/oidc/             # OIDC provider: JWT tokens, auth codes, sessions, keys, password hashing
lib/rest/filters/     # HTTP middleware (request logging, authentication, authorization, request info)
pkg/apis/             # Business logic: API types, store interfaces, REST storage, validation
pkg/apis/iam/         # IAM module: users, workspaces, namespaces, RBAC, OIDC handlers
pkg/apis/iam/store/   # PostgreSQL store implementations (pg_*.go)
pkg/apis/iam/v1/      # Route registration (install.go)
pkg/apis/dashboard/   # Dashboard module: overview statistics API
pkg/db/               # Database: connection pool, pagination helpers, sqlc config
pkg/db/schema/        # PostgreSQL DDL (schema.sql)
pkg/db/query/         # sqlc SQL query files (*.sql)
pkg/db/generated/     # sqlc auto-generated Go code (DO NOT EDIT)
cmd/openapi-gen/      # OpenAPI spec generator from +openapi: annotations
cmd/init-admin/       # CLI tool to initialize admin user in database
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
- **`handler` package** must not import `iam` or any module package. It receives OIDC mux as `http.Handler` via `RootHandlerConfig.OIDCMux`.
- **`apis` package** is the assembly/bridge layer. Cross-cutting concerns (shared caches, authorizer wiring, OIDC mux creation) live here as bridge functions, not in `Result`.
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
- **IDs**: `int64` (BIGSERIAL) in DB, `string` in API layer. Use `rest.ParseID()` for string→int64 conversion
- **List queries**: Use `restOptionsToListQuery(options)` to convert `rest.ListOptions` → `db.ListQuery`. Result type: `db.ListResult[T]{Items, TotalCount}`
- **Pagination**: `page` (1-based), `pageSize` (default 20, max 100), `sortBy`, `sortOrder`
- **Batch operations**: Use `BatchRequest{IDs []string}` for batch add; `rest.DeleteCollectionRequest{IDs}` for batch delete
- **Content negotiation**: JSON (default) + YAML (via `Accept: application/yaml`)

### Error Handling

```go
apierrors.NewBadRequest(message, details)   // 400
apierrors.NewForbidden(message)             // 403
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
# Authorization middleware checks RBAC permissions per request (scope + permission code).
# Inactive users receive 401 even with a valid token. Token refresh is also blocked for inactive users.

# IAM Module
/api/iam/v1/users                                                    # CRUD + batch delete
/api/iam/v1/users/{userId}/change-password                           # POST change password
/api/iam/v1/users/{userId}:workspaces                                # GET user's joined workspaces
/api/iam/v1/users/{userId}:namespaces                                # GET user's joined namespaces
/api/iam/v1/users/{userId}:rolebindings                              # GET user's role bindings
/api/iam/v1/users/{userId}:permissions                               # GET user's expanded permissions
/api/iam/v1/workspaces                                               # CRUD + batch delete
/api/iam/v1/workspaces/{workspaceId}/transfer-ownership              # POST transfer ownership
/api/iam/v1/workspaces/{workspaceId}/users                           # list + batch add/remove
/api/iam/v1/workspaces/{workspaceId}/namespaces                      # CRUD + batch delete
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/users  # list + batch add/remove
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/transfer-ownership  # POST
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/rolebindings  # list + create/delete
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/roles  # CRUD (scoped)
/api/iam/v1/workspaces/{workspaceId}/rolebindings                    # list + create/delete
/api/iam/v1/workspaces/{workspaceId}/roles                           # CRUD (scoped)
/api/iam/v1/namespaces                                               # CRUD + batch delete
/api/iam/v1/namespaces/{namespaceId}/users                           # list + batch add/remove
/api/iam/v1/namespaces/{namespaceId}/rolebindings                    # list + create/delete
/api/iam/v1/namespaces/{namespaceId}/roles                           # CRUD (scoped)
/api/iam/v1/permissions                                              # list (read-only, auto-registered)
/api/iam/v1/roles                                                    # list (platform-level, read-only)
/api/iam/v1/rolebindings                                             # list + create/delete (platform-level)

# Dashboard Module
/api/dashboard/v1/overview                                           # GET platform stats
/api/dashboard/v1/workspaces/{workspaceId}/overview                  # GET workspace stats
/api/dashboard/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/overview  # GET namespace stats
```

## Resource Hierarchy

```
Workspace (tenant/organization)
  └── Namespace (project/team scope)
       └── User (member with role binding)

Role (scoped: platform / workspace / namespace)
  └── Permission Rules (wildcard patterns, e.g. "iam:users:*")

RoleBinding (user + role + scope)
Permission (auto-registered from resource tree, read-only)
```

- Creating a Workspace auto-creates a default Namespace, built-in roles, and adds owner as member
- Creating a Namespace auto-creates built-in roles and adds owner as member
- Adding a User to a Namespace auto-adds them to the parent Workspace
- Deleting a Workspace requires no child Namespaces
- Deleting a Namespace requires no member Users
- Ownership can be transferred via dedicated endpoints (workspace/namespace)

## RBAC Architecture

### Permission Auto-Registration

Permissions are auto-generated from the resource tree at startup via `SyncPermissions`. Each resource + verb combination produces a permission code (e.g., `iam:users:list`, `iam:workspaces:namespaces:create`). No manual permission maintenance needed.

### Three-Level Scope Chain

Permission checking follows `platform → workspace → namespace` inheritance: a platform-level permission grants access at all scopes.

### Permission Cache

`RBACChecker` caches user permission entries with a TTL. Uses `singleflight.Group` to deduplicate concurrent DB loads for the same user. Cache invalidation happens on role binding changes, workspace/namespace deletion (including batch), and ownership transfers — always through the `PermissionChecker` interface, never via direct `sharedPermCache` access.

### Built-in Role Seeding

`SeedRBAC` runs at startup: upserts platform roles with rules, creates scoped roles for existing workspaces/namespaces, and migrates old global roles to scoped roles. Split into sub-functions (`seedBuiltinRoles`, `seedScopedRolesForWorkspaces`, `seedScopedRolesForNamespaces`, `migrateGlobalRolesToScoped`) for clarity.

## Testing

- Unit tests in `lib/` and `pkg/apis/iam/` (standard `testing` package + `httptest`)
- RBAC tests: `rbac_checker_test.go`, `rbac_cache_test.go`, `rbac_match_test.go`, `rbac_seed_test.go`, `rbac_sync_test.go`
- Authorization middleware tests: `lib/rest/filters/authorization_test.go`, `requestinfo_test.go`
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

## Git Worktree 开发注意事项

以下文件被 `.gitignore` 忽略，创建 worktree 后不会自动出现，需要手动处理：

| 文件 | 用途 | 处理方式 |
|------|------|---------|
| `oidc-private.pem` / `oidc-public.pem` | OIDC 签名密钥 | 从主仓库复制：`cp ../../oidc-*.pem .` |
| `app/lcp-server/config.dev.yaml` | `make dev` 开发配置（redirectUri 指向 5173） | 从主仓库复制：`cp ../../app/lcp-server/config.dev.yaml app/lcp-server/` |
| `ui/dist/` | 前端构建产物（Go embed） | 创建占位或构建：`mkdir -p ui/dist && touch ui/dist/.gitkeep`，或 `cd ui && pnpm install && pnpm build` |
| `ui/node_modules/` | 前端依赖 | `cd ui && pnpm install` |

### Worktree 初始化完整流程

```bash
# 1. 创建 worktree
git worktree add .worktrees/<branch-name> -b <branch-name>
cd .worktrees/<branch-name>

# 2. 复制 gitignored 的配置文件
cp ../../oidc-*.pem .
cp ../../app/lcp-server/config.dev.yaml app/lcp-server/

# 3. Go 依赖
go mod download

# 4. 前端依赖 + 构建（服务启动需要 ui/dist）
cd ui && pnpm install && pnpm build && cd ..

# 5. 验证服务可启动
go run ./app/lcp-server/ -config ./app/lcp-server/config.yaml
```

### 数据库 Schema 变更

如果分支修改了 `pkg/db/schema/schema.sql`，本地数据库不会自动迁移。需要手动执行 DDL 变更：

```bash
# 查看表结构
docker exec lcp-postgres psql -U lcp -d lcp -c "\d <table_name>"

# 执行迁移（示例：修改约束）
docker exec lcp-postgres psql -U lcp -d lcp -c "ALTER TABLE ..."
```

注意：切换回 main 分支后数据库 schema 可能不兼容，需要相应回滚。
