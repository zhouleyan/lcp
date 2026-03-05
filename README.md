# LCP

LCP 是一个 PaaS 管理平台，基于 Go 构建，使用自研 REST API 框架 + PostgreSQL + sqlc。

## 快速开始

### 前置依赖

- Go 1.26+
- Docker（用于运行 PostgreSQL）
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)（用于生成数据库查询代码）

### 1. 启动数据库

```bash
docker run -d \
  --name lcp-postgres \
  -e POSTGRES_USER=lcp \
  -e POSTGRES_PASSWORD=lcp \
  -e POSTGRES_DB=lcp \
  -p 5432:5432 \
  postgres:18
```

### 2. 初始化数据库 Schema

```bash
docker cp pkg/db/schema/schema.sql lcp-postgres:/tmp/schema.sql
docker exec lcp-postgres psql -U lcp -d lcp -f /tmp/schema.sql
```

> 注意：Schema 变更后需要重新执行。如需重建，先 DROP 所有表再重新导入。

### 3. 启动服务

```bash
# 开发模式（带竞态检测）
make lcp-server
./bin/lcp-server-race -config ./app/lcp-server/config.yaml

# 或直接 go run
go run ./app/lcp-server/ -config ./app/lcp-server/config.yaml
```

服务默认监听 `:8428`，可通过 `-httpListenerAddr` 指定：

```bash
lcp-server -httpListenerAddr=:8080 -config ./app/lcp-server/config.yaml
```

### 4. 验证

```bash
curl http://localhost:8428/api/v1/users
# 返回 {"apiVersion":"v1","kind":"UserList","items":[],"totalCount":0}
```

## 配置

配置文件位于 `app/lcp-server/config.yaml`，优先级从高到低：

**CLI 参数 > 环境变量 > 配置文件 > 默认值**

```yaml
database:
  host: "localhost"       # 环境变量: DB_HOST
  port: 5432              # 环境变量: DB_PORT
  user: "lcp"             # 环境变量: DB_USER
  password: "lcp"         # 环境变量: DB_PASSWORD
  dbName: "lcp"           # 环境变量: DB_NAME
  sslMode: "disable"      # 环境变量: DB_SSL_MODE
  maxConns: 10            # 环境变量: DB_MAX_CONNS

logger:
  level: "INFO"           # INFO, WARN, ERROR, FATAL, PANIC
  format: "default"       # default, json
```

运行时发送 `SIGHUP` 信号可热重载配置（不影响 CLI 参数覆盖项）。

## 常用命令

| 命令 | 说明 |
|------|------|
| `make lcp-server` | 开发构建（带 `-race`） |
| `make lcp-server-prod` | 生产构建（`CGO_ENABLED=0`） |
| `make sqlc-generate` | 从 SQL 文件生成 Go 代码 |
| `make openapi-gen` | 生成 OpenAPI 文档到 `docs/` |
| `make test` | 运行测试 |
| `make vet` | 代码静态检查 |
| `make lint` | golangci-lint 检查 |
| `make fmt` | 格式化代码 |

## 项目结构

```
app/lcp-server/           # 应用入口：配置加载、依赖注入、HTTP 服务启动
pkg/
├── apis/
│   ├── install.go         # 全局依赖注入：Store 实例化 → Provider → APIGroupInfo
│   └── iam/               # IAM 业务模块
│       ├── types.go       #   API 类型定义 + DB 类型别名
│       ├── store.go       #   Store 接口定义
│       ├── storage.go     #   REST Storage 实现（HTTP 层 ↔ Store 层桥接）
│       ├── validation.go  #   请求验证
│       ├── provider.go    #   Store 聚合器
│       ├── store/         #   PostgreSQL Store 实现
│       │   └── pg_*.go
│       └── v1/
│           └── install.go #   路由注册
└── db/
    ├── schema/schema.sql  # DDL（表结构定义）
    ├── query/*.sql        # sqlc 查询文件
    ├── generated/         # sqlc 自动生成代码（勿手动修改）
    ├── db.go              # 数据库连接池
    ├── query.go           # 分页/列表通用类型
    └── sqlc.yaml          # sqlc 配置
lib/                       # 框架层（通常不需要修改）
```

## 添加新业务功能

以新增 `Project` 资源为例，完整流程如下：

### Step 1：定义数据库表

在 `pkg/db/schema/schema.sql` 中添加表定义：

```sql
CREATE TABLE projects (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    workspace_id BIGINT       NOT NULL REFERENCES workspaces(id),
    owner_id     BIGINT       NOT NULL REFERENCES users(id),
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

### Step 2：编写 sqlc 查询

新建 `pkg/db/query/project.sql`，每个查询以 `-- name:` 注释声明方法名和返回类型：

```sql
-- name: CreateProject :one
INSERT INTO projects (name, display_name, description, workspace_id, owner_id, status)
VALUES (@name, @display_name, @description, @workspace_id, @owner_id, @status)
RETURNING id, name, display_name, description, workspace_id, owner_id, status,
          created_at, updated_at;

-- name: GetProjectByID :one
SELECT id, name, display_name, description, workspace_id, owner_id, status,
       created_at, updated_at
FROM projects WHERE id = @id;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = @id;

-- name: CountProjects :one
SELECT count(id) FROM projects
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('name')::VARCHAR IS NULL OR name ILIKE '%' || sqlc.narg('name') || '%');

-- name: ListProjects :many
SELECT p.*, u.username AS owner_username
FROM projects p
JOIN users u ON p.owner_id = u.id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR p.status = sqlc.narg('status'))
  AND (sqlc.narg('name')::VARCHAR IS NULL OR p.name ILIKE '%' || sqlc.narg('name') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN p.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN p.name END DESC,
    p.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
```

**查询文件规范：**
- `:one` 返回单条记录，`:many` 返回列表，`:exec` 无返回
- 必选参数用 `@param`，可选过滤条件用 `sqlc.narg('param')` 配合 `IS NULL OR` 模式
- 排序用 `CASE WHEN` 动态切换字段和方向，尾部加默认排序 `created_at DESC`
- 分页固定用 `sqlc.arg('page_size')` 和 `sqlc.arg('page_offset')`

### Step 3：生成代码

```bash
make sqlc-generate
```

会在 `pkg/db/generated/` 下自动生成 `models.go`（结构体）和 `project.sql.go`（方法），**不要手动修改这些文件**。

### Step 4：定义 API 类型

在 `pkg/apis/iam/types.go` 中添加：

```go
// --- Project types ---

// Project
// +openapi:description=Project is the API representation of a project resource.
// +openapi:path=/projects
type Project struct {
    runtime.TypeMeta `json:",inline"`
    types.ObjectMeta `json:"metadata"`
    Spec             ProjectSpec `json:"spec"`
}

func (p *Project) GetTypeMeta() *runtime.TypeMeta { return &p.TypeMeta }

type ProjectSpec struct {
    DisplayName string `json:"displayName,omitempty"`
    Description string `json:"description,omitempty"`
    WorkspaceID string `json:"workspaceId"`
    OwnerID     string `json:"ownerId"`
    Status      string `json:"status,omitempty"`
}

type ProjectList struct {
    runtime.TypeMeta `json:",inline"`
    Items            []Project `json:"items"`
    TotalCount       int64     `json:"totalCount"`
}

func (p *ProjectList) GetTypeMeta() *runtime.TypeMeta { return &p.TypeMeta }

// DB 类型别名
type DBProject = generated.Project
```

**类型规范：**
- API 类型统一用 `TypeMeta`（承载 apiVersion/kind） + `ObjectMeta`（承载 id/name/时间戳） + `Spec` 三层结构
- DB 层 ID 为 `int64`，API 层为 `string`，转换在 Storage 层完成
- `+openapi:` 注释用于自动生成 OpenAPI 文档

### Step 5：定义 Store 接口

在 `pkg/apis/iam/store.go` 中添加：

```go
type ProjectStore interface {
    Create(ctx context.Context, p *DBProject) (*DBProject, error)
    GetByID(ctx context.Context, id int64) (*DBProject, error)
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBProject], error)
}
```

### Step 6：实现 PostgreSQL Store

新建 `pkg/apis/iam/store/pg_project.go`：

```go
package store

type pgProjectStore struct {
    db      *pgxpool.Pool
    queries *generated.Queries
}

func NewPGProjectStore(pool *pgxpool.Pool, queries *generated.Queries) iam.ProjectStore {
    return &pgProjectStore{db: pool, queries: queries}
}

func (s *pgProjectStore) GetByID(ctx context.Context, id int64) (*iam.DBProject, error) {
    row, err := s.queries.GetProjectByID(ctx, id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, apierrors.NewNotFound("project", fmt.Sprintf("%d", id))
        }
        return nil, fmt.Errorf("get project: %w", err)
    }
    return &row, nil
}

// Create, Delete, List 同理...
```

**Store 实现规范：**
- `pgx.ErrNoRows` 统一转为 `apierrors.NewNotFound()`
- 需要多步操作时使用事务：`pool.Begin()` → `queries.WithTx(tx)` → 操作 → `tx.Commit()`
- `List` 方法使用 `db.PaginationToOffsetLimit()` 转换分页参数
- 可选过滤用 `filterStr()` / `filterInt64()` 辅助函数提取 `map[string]any` 中的值

### Step 7：添加验证

在 `pkg/apis/iam/validation.go` 中添加：

```go
func ValidateProjectCreate(name string, spec *ProjectSpec) validation.ErrorList {
    var errs validation.ErrorList
    if name == "" {
        errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
    }
    if spec.OwnerID == "" {
        errs = append(errs, validation.FieldError{Field: "spec.ownerId", Message: "is required"})
    }
    return errs
}
```

### Step 8：实现 REST Storage

在 `pkg/apis/iam/storage.go` 中添加。Storage 是 HTTP 层与 Store 层之间的桥梁，负责：
- 解析路径参数（`options.PathParams["projectId"]`）
- 调用验证函数
- DB 类型 ↔ API 类型转换
- 处理 `DryRun` 选项

```go
type projectStorage struct {
    projStore ProjectStore
    userStore UserStore
}

func NewProjectStorage(projStore ProjectStore, userStore UserStore) rest.StandardStorage {
    return &projectStorage{projStore: projStore, userStore: userStore}
}

func (s *projectStorage) NewObject() runtime.Object { return &Project{} }

func (s *projectStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
    id := options.PathParams["projectId"]
    pid, err := parseID(id)
    if err != nil {
        return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid project ID: %s", id), nil)
    }
    p, err := s.projStore.GetByID(ctx, pid)
    if err != nil {
        return nil, err
    }
    return projectToAPI(p), nil
}

// List, Create, Update, Patch, Delete, DeleteCollection 同理...
```

**根据需要实现的接口选择能力：**

| 接口 | HTTP 方法 | 说明 |
|------|----------|------|
| `rest.Getter` | GET `/{id}` | 获取单个资源 |
| `rest.Lister` | GET `/` | 列表查询 |
| `rest.Creator` | POST `/` | 创建资源 |
| `rest.Updater` | PUT `/{id}` | 全量更新 |
| `rest.Patcher` | PATCH `/{id}` | 部分更新 |
| `rest.Deleter` | DELETE `/{id}` | 删除单个 |
| `rest.CollectionDeleter` | DELETE `/` | 批量删除（请求体 `{"ids":[...]}`) |
| `rest.StandardStorage` | 以上全部 | 完整 CRUD |

不需要实现全部接口，框架会自动根据类型断言只注册已实现的路由。

### Step 9：注册到 Provider

在 `pkg/apis/iam/provider.go` 中添加 Store 字段和访问器：

```go
type RESTStorageProvider struct {
    // ... 已有字段
    projStore ProjectStore  // 新增
}

func (p *RESTStorageProvider) ProjectStore() ProjectStore { return p.projStore }
```

更新 `NewRESTStorageProvider` 参数列表。

### Step 10：注册路由

在 `pkg/apis/iam/v1/install.go` 中添加资源：

```go
projStorage := iam.NewProjectStorage(p.ProjectStore(), p.UserStore())

// 作为顶层资源
Resources: []rest.ResourceInfo{
    {Name: "projects", Storage: projStorage},
}

// 或作为子资源（如 workspace 下的 project）
SubResources: []rest.ResourceInfo{
    {Name: "projects", Storage: projStorage},
}
```

路由路径自动从资源名推导：`"projects"` → `/{projectId}`。

**子资源嵌套**时，父级路径参数会通过 `options.PathParams` 传递到 Storage：

```go
// /api/v1/workspaces/{workspaceId}/projects → PathParams["workspaceId"] 可用
```

### Step 11：依赖注入

在 `pkg/apis/install.go` 中实例化具体 Store 并传入 Provider：

```go
iamProvider := iam.NewRESTStorageProvider(
    // ... 已有
    iamstore.NewPGProjectStore(database.Pool, database.Queries),  // 新增
)
```

### 完成

运行验证：

```bash
make sqlc-generate   # 确保生成代码是最新的
go build ./...       # 编译通过
go vet ./pkg/...     # 无代码问题
make openapi-gen     # 更新 API 文档
```

## 请求/响应格式

### 创建资源

```bash
curl -X POST http://localhost:8428/api/v1/workspaces \
  -H 'Content-Type: application/json' \
  -d '{
    "metadata": {"name": "my-workspace"},
    "spec": {
      "displayName": "My Workspace",
      "ownerId": "1"
    }
  }'
```

### 列表查询

```bash
# 分页 + 过滤 + 排序
curl 'http://localhost:8428/api/v1/workspaces?page=1&pageSize=10&sortBy=name&sortOrder=asc&status=active'
```

### 批量删除

```bash
curl -X DELETE http://localhost:8428/api/v1/workspaces \
  -H 'Content-Type: application/json' \
  -d '{"ids": ["1", "2", "3"]}'
```

### 批量添加成员

```bash
curl -X POST http://localhost:8428/api/v1/workspaces/1/users \
  -H 'Content-Type: application/json' \
  -d '{"ids": ["2", "3"]}'
```

### 内容协商

默认返回 JSON，通过 `Accept` 头请求 YAML：

```bash
curl -H 'Accept: application/yaml' http://localhost:8428/api/v1/users
```

### 错误响应

所有错误统一格式：

```json
{
  "apiVersion": "v1",
  "kind": "Status",
  "status": 400,
  "reason": "BadRequest",
  "message": "validation failed",
  "details": [
    {"field": "metadata.name", "message": "is required"}
  ]
}
```

## 当前 API 路由

| 方法 | 路径 | 说明 |
|------|------|------|
| CRUD | `/api/v1/users` | 用户管理 |
| CRUD | `/api/v1/workspaces` | 组织管理 |
| CRUD | `/api/v1/workspaces/{workspaceId}/namespaces` | 组织下的命名空间管理 |
| L/C/D | `/api/v1/workspaces/{workspaceId}/users` | 组织成员管理 |
| L/C/D | `/api/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/users` | 命名空间成员管理 |
| CRUD | `/api/v1/namespaces` | 全局命名空间管理 |
| L/C/D | `/api/v1/namespaces/{namespaceId}/users` | 命名空间成员管理 |

> CRUD = POST + GET + GET/{id} + PUT/{id} + PATCH/{id} + DELETE/{id} + DELETE（批量）
> L/C/D = GET + POST + DELETE（列表/批量添加/批量移除）
