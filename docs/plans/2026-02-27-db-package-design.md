# lib/db 包设计文档

## 概述

在 `lib/db` 下创建 PostgreSQL 数据库访问层，使用 sqlc 生成类型安全的查询代码，pgx/v5 直连 PostgreSQL。提供 User、Namespace 对象的完整 CRUD，支持多对多关系、动态筛选、排序和分页。

## 技术选型

- **数据库驱动**: pgx/v5 直连（非 database/sql 接口）
- **查询生成**: sqlc v2，生成代码放 `generated/` 子包
- **迁移工具**: 无，手动管理 schema SQL
- **复杂查询**: 纯 sqlc 动态参数（`sqlc.narg` + `CASE WHEN`）

## 目录结构

```
lib/db/
├── sqlc.yaml              # sqlc 配置
├── db.go                  # 连接池管理（pgxpool）
├── schema/
│   └── schema.sql         # DDL（表 + 索引）
├── query/
│   ├── user.sql           # User CRUD + 列表
│   ├── namespace.sql      # Namespace CRUD + 列表
│   └── user_namespace.sql # 关联操作 + 关联查询
└── generated/             # sqlc 生成（不手动编辑）
    ├── db.go
    ├── models.go
    ├── querier.go
    ├── user.sql.go
    ├── namespace.sql.go
    └── user_namespace.sql.go
```

## 数据模型

### users 表

| 字段 | 类型 | 约束 |
|------|------|------|
| id | BIGSERIAL | PK |
| username | VARCHAR(255) | NOT NULL, UNIQUE |
| email | VARCHAR(255) | NOT NULL, UNIQUE |
| display_name | VARCHAR(255) | NOT NULL DEFAULT '' |
| phone | VARCHAR(50) | NOT NULL DEFAULT '' |
| avatar_url | VARCHAR(512) | NOT NULL DEFAULT '' |
| status | VARCHAR(20) | NOT NULL DEFAULT 'active' |
| last_login_at | TIMESTAMPTZ | nullable |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

索引: `idx_users_status`, `idx_users_created_at`, `idx_users_display_name`

### namespaces 表

| 字段 | 类型 | 约束 |
|------|------|------|
| id | BIGSERIAL | PK |
| name | VARCHAR(255) | NOT NULL, UNIQUE |
| display_name | VARCHAR(255) | NOT NULL DEFAULT '' |
| description | TEXT | NOT NULL DEFAULT '' |
| owner_id | BIGINT | NOT NULL, FK -> users(id) |
| visibility | VARCHAR(20) | NOT NULL DEFAULT 'private' |
| max_members | INT | NOT NULL DEFAULT 0 |
| status | VARCHAR(20) | NOT NULL DEFAULT 'active' |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

索引: `idx_namespaces_owner_id`, `idx_namespaces_status`, `idx_namespaces_visibility`, `idx_namespaces_created_at`

### user_namespaces 关联表

| 字段 | 类型 | 约束 |
|------|------|------|
| user_id | BIGINT | NOT NULL, FK -> users(id) ON DELETE CASCADE |
| namespace_id | BIGINT | NOT NULL, FK -> namespaces(id) ON DELETE CASCADE |
| role | VARCHAR(50) | NOT NULL DEFAULT 'member' |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

PK: (user_id, namespace_id)
索引: `idx_user_namespaces_namespace_id`, `idx_user_namespaces_role`

## 查询设计

### 基础 CRUD

每个实体提供: Create, GetByID, Update, Delete。所有查询显式列出字段，不使用 `SELECT *`。

### 复杂列表查询

- **筛选**: 使用 `sqlc.narg` 实现可选参数，NULL 时跳过该条件
- **模糊搜索**: `ILIKE '%' || param || '%'`
- **排序**: `CASE WHEN` 动态选择排序字段和方向
- **分页**: `LIMIT` + `OFFSET`
- **关联**: 用户列表通过 LEFT JOIN 关联 namespace，用 `array_agg` 聚合 namespace 名称

### 关联查询

- ListNamespacesByUserID: 查询用户所属的所有 namespace（带角色）
- ListUsersByNamespaceID: 查询 namespace 下的所有用户（带角色）
- AddUserToNamespace / RemoveUserFromNamespace: 关联表操作
- CountUsers / CountNamespaces: 分页计数

## 连接管理

使用 `pgxpool.Pool` 管理连接池，`Config` 结构体配置连接参数，`NewPool()` 创建并验证连接。
