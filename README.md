# LCP

LCP 是一个 PaaS 管理平台，后端基于 Go 自研 REST 框架（参考 Kubernetes apiserver 模式），前端使用 React + TypeScript + shadcn/ui。内置 OIDC 认证、三级 RBAC 权限体系（平台/工作空间/项目）。

## 快速部署

使用 Docker Compose 一键启动（含 PostgreSQL + lcp-server），无需本地安装 Node.js/Go：

```bash
cd deployment/docker
docker compose up -d
```

服务启动后访问 http://localhost:8428 ，默认管理员：

| 字段 | 值 |
|------|------|
| 用户名 | `admin` |
| 密码 | `Admin123!` |

停止服务：

```bash
cd deployment/docker
docker compose down          # 保留数据
docker compose down -v       # 清除数据
```

### 单独构建镜像

两种模式，自动检测 podman/docker：

```bash
# 完整构建：前端 + 后端全在容器内构建，无需本地 Node.js
make docker-build

# 预构建：使用本地已构建的 ui/dist（更快，需要本地 Node.js + pnpm）
make docker-build-local

# 自定义选项
make docker-build IMAGE_TAG=v1.0.0
make docker-build CONTAINER_ENGINE=docker
```

## 配置

配置文件位于 `app/lcp-server/config.yaml`，优先级：命令行参数 > 环境变量 > 配置文件 > 默认值。支持 SIGHUP 热重载。

### 数据库

```yaml
database:
  host: "localhost"       # env: DB_HOST
  port: 5432              # env: DB_PORT
  user: "lcp"             # env: DB_USER
  password: "lcp"         # env: DB_PASSWORD
  dbName: "lcp"           # env: DB_NAME
  sslMode: "disable"      # env: DB_SSL_MODE
  maxConns: 10            # env: DB_MAX_CONNS
```

### OIDC 认证

签名密钥在首次启动时自动生成并存储在 PostgreSQL 中，无需手动配置密钥文件。

```yaml
oidc:
  issuer: "http://localhost:8428"
  algorithm: "EdDSA"
  accessTokenTTL: "1h"
  refreshTokenTTL: "168h"
  authCodeTTL: "5m"
  loginUrl: "/login"
  clients:
    - id: "lcp-ui"
      public: true
      redirectUris:
        - "http://localhost:8428/auth/callback"
      scopes: ["openid", "profile", "email", "phone"]
```

不配置 OIDC 时认证禁用，所有 API 开放访问。

### 初始管理员

```yaml
admin:
  username: "admin"
  password: "Admin123!"
  email: "admin@lcp.io"
  phone: "13800000000"
  displayName: "Admin"
```

首次启动自动创建，后续启动跳过。

### 日志

```yaml
logger:
  level: "INFO"           # INFO, WARN, ERROR, FATAL, PANIC
  format: "default"       # default, json
```

### Docker Compose 环境变量

`deployment/docker/.env` 控制容器环境：

```env
DB_USER=lcp
DB_PASSWORD=lcp
DB_NAME=lcp
TIME_ZONE=Asia/Shanghai
```

## 开发

### 环境要求

| 工具 | 版本 |
|------|------|
| Go | 1.26+ |
| Node.js | 20+ |
| pnpm | 10+ |
| PostgreSQL | 15+ |

### 本地数据库

```bash
docker run -d --name lcp-postgres \
  -e POSTGRES_USER=lcp -e POSTGRES_PASSWORD=lcp -e POSTGRES_DB=lcp \
  -p 5432:5432 postgres:18

docker exec -i lcp-postgres psql -U lcp -d lcp < pkg/db/schema/schema.sql
```

### 前后端联调

```bash
make dev
```

同时启动后端（:8428）和前端 Vite 开发服务器（:5173），前端热更新。后端使用 `config.dev.yaml`（OIDC redirectUri 指向 :5173）。

### 后端

#### 常用命令

| 命令 | 说明 |
|------|------|
| `make lcp-server` | 开发构建（含 race detector） |
| `make lcp-server-prod` | 生产构建（CGO_ENABLED=0） |
| `make test` | 运行测试 |
| `make vet` | go vet |
| `make lint` | golangci-lint |
| `make fmt` | 格式化 |
| `make sqlc-generate` | 修改 SQL 后重新生成 Go 代码 |
| `make openapi-gen` | 从注解生成 OpenAPI spec |
| `make init-admin` | 手动创建管理员 |

单独启动后端：

```bash
go run ./app/lcp-server/ -config ./app/lcp-server/config.yaml
```

#### 目录结构

```
app/lcp-server/       # 服务入口（配置、装配、HTTP 监听）
lib/                  # 框架库（REST 框架、运行时、配置、日志等）
lib/oidc/             # OIDC 提供者（JWT、授权码、会话、密钥管理）
lib/rest/filters/     # HTTP 中间件（日志、认证、授权、请求解析）
pkg/apis/             # 业务逻辑：API 类型、Store 接口、REST 存储、校验
pkg/apis/iam/         # IAM 模块（用户、工作空间、项目、RBAC）
pkg/apis/dashboard/   # Dashboard 模块（概览统计）
pkg/db/               # 数据库连接池、分页、sqlc 配置
pkg/db/schema/        # DDL（schema.sql）
pkg/db/query/         # sqlc SQL 查询文件
pkg/db/generated/     # sqlc 自动生成的 Go 代码（勿手动修改）
cmd/openapi-gen/      # OpenAPI 规范生成器
cmd/init-admin/       # 管理员初始化 CLI
deployment/docker/    # Dockerfile、docker-compose、初始化脚本
```

#### 添加新资源

1. `pkg/db/schema/schema.sql` — 建表
2. `pkg/db/query/<resource>.sql` — 编写 sqlc 查询
3. `make sqlc-generate` — 生成 Go 代码
4. `pkg/apis/iam/types.go` — 定义 API 类型（TypeMeta + ObjectMeta + Spec）
5. `pkg/apis/iam/store.go` — 定义 Store 接口
6. `pkg/apis/iam/store/pg_<resource>.go` — 实现 PostgreSQL Store
7. `pkg/apis/iam/validation.go` — 添加校验
8. `pkg/apis/iam/storage.go` — 实现 REST Storage（HTTP ↔ Store 桥接）
9. `pkg/apis/iam/provider.go` — 注册 Store 字段
10. `pkg/apis/iam/v1/install.go` — 注册路由
11. `pkg/apis/install.go` — 装配具体 Store 实例

#### REST Storage 接口

按需实现，框架自动注册已实现的路由：

| 接口 | HTTP 方法 | 说明 |
|------|----------|------|
| `rest.Getter` | GET `/{id}` | 获取单个资源 |
| `rest.Lister` | GET `/` | 列表查询 |
| `rest.Creator` | POST `/` | 创建资源 |
| `rest.Updater` | PUT `/{id}` | 全量更新 |
| `rest.Patcher` | PATCH `/{id}` | 部分更新 |
| `rest.Deleter` | DELETE `/{id}` | 删除单个 |
| `rest.CollectionDeleter` | DELETE `/` | 批量删除 |
| `rest.StandardStorage` | 以上全部 | 完整 CRUD |

#### API 请求/响应格式

```bash
# 创建
curl -X POST http://localhost:8428/api/iam/v1/workspaces \
  -H 'Content-Type: application/json' \
  -d '{"metadata":{"name":"my-ws"},"spec":{"displayName":"My Workspace","ownerId":"1"}}'

# 列表（分页 + 过滤 + 排序）
curl 'http://localhost:8428/api/iam/v1/workspaces?page=1&pageSize=10&sortBy=name&sortOrder=asc'

# 批量删除
curl -X DELETE http://localhost:8428/api/iam/v1/workspaces \
  -H 'Content-Type: application/json' -d '{"ids":["1","2"]}'

# YAML 格式
curl -H 'Accept: application/yaml' http://localhost:8428/api/iam/v1/users
```

错误统一格式：

```json
{
  "apiVersion": "v1",
  "kind": "Status",
  "status": 400,
  "reason": "BadRequest",
  "message": "validation failed",
  "details": [{"field": "metadata.name", "message": "is required"}]
}
```

### 前端

#### 常用命令

```bash
cd ui
pnpm install             # 安装依赖
pnpm dev                 # 开发服务器（:5173）
pnpm build               # 生产构建
pnpm test                # 运行测试
npx tsc --noEmit         # 类型检查
```

#### 目录结构

```
ui/src/
  api/              # API 客户端、类型定义
  api/iam/          # IAM 模块 API（用户、工作空间、项目、RBAC）
  components/       # 共享组件（scope-selector、permission-selector、ui/ shadcn 原语）
  hooks/            # 自定义 Hooks（use-permission、use-list-state）
  i18n/             # 国际化（中英文）
  lib/              # 认证工具（OIDC PKCE 流程、Token 管理）、导航配置
  layouts/          # 布局组件（root-layout、scope 同步）
  pages/            # 页面组件（按模块组织：iam、dashboard、infra）
  stores/           # Zustand 状态管理（auth、permission、scope、workspace）
  routes.tsx        # 路由定义（三级 scope 路由）
  modules.ts        # 模块注册
```

#### 技术栈

- React 18 + TypeScript + Vite
- Tailwind CSS + shadcn/ui（Radix 原语）
- react-hook-form + zod/v4
- ky（HTTP 客户端）
- zustand（状态管理）
- react-i18next（国际化）

#### 添加新页面

1. `pages/<module>/<resource>/` — 创建页面组件
2. `routes.tsx` — 注册三级 scope 路由（platform / workspace / namespace）
3. `lib/nav-config.ts` — 添加 `NAV_ITEMS` 条目（导航、权限、图标、scope）
4. `i18n/locales/` — 添加中英文翻译
5. `api/<module>/` — 添加 API 函数

#### Scope 路由模式

所有支持工作空间/项目 scope 的路由必须在 URL 中嵌入 scope ID：

| Scope | URL 模式 |
|-------|----------|
| 平台 | `/{module}/{resource}` |
| 工作空间 | `/{module}/workspaces/:wsId/{resource}` |
| 项目 | `/{module}/workspaces/:wsId/namespaces/:nsId/{resource}` |

## API 路由

```
# OIDC（公开，无需认证）
GET  /.well-known/openid-configuration
GET  /.well-known/jwks.json
GET  /oidc/authorize
POST /oidc/login
POST /oidc/token
GET  /oidc/userinfo

# IAM 模块
/api/iam/v1/users                                                    # CRUD + 批量删除
/api/iam/v1/users/{userId}/change-password                           # 修改密码
/api/iam/v1/users/{userId}:workspaces                                # 用户关联的工作空间
/api/iam/v1/users/{userId}:namespaces                                # 用户关联的项目
/api/iam/v1/users/{userId}:rolebindings                              # 用户角色绑定
/api/iam/v1/users/{userId}:permissions                               # 用户权限视图
/api/iam/v1/workspaces                                               # CRUD + 批量删除
/api/iam/v1/workspaces/{workspaceId}/transfer-ownership              # 转移所有权
/api/iam/v1/workspaces/{workspaceId}/users                           # 成员管理
/api/iam/v1/workspaces/{workspaceId}/namespaces                      # 项目管理
/api/iam/v1/workspaces/{workspaceId}/roles                           # 工作空间角色
/api/iam/v1/workspaces/{workspaceId}/rolebindings                    # 工作空间角色绑定
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/users  # 项目成员
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/roles  # 项目角色
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/rolebindings
/api/iam/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/transfer-ownership
/api/iam/v1/namespaces                                               # 全局项目管理
/api/iam/v1/permissions                                              # 权限列表（只读）
/api/iam/v1/roles                                                    # 平台角色（只读）
/api/iam/v1/rolebindings                                             # 平台角色绑定

# Dashboard 模块
/api/dashboard/v1/overview                                           # 平台概览
/api/dashboard/v1/workspaces/{workspaceId}/overview                  # 工作空间概览
/api/dashboard/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/overview
```
