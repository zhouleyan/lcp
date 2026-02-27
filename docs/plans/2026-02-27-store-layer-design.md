# lib/store 层设计文档

## 概述

在 `lib/store` 下创建业务抽象层，封装 `lib/db/generated` 的 sqlc 查询，提供接口抽象（可 mock 测试）和业务逻辑封装。handler 层只依赖 `lib/store` 接口，不感知底层数据库实现。

## 技术选型

- 接口定义在 `lib/store/` 顶层
- PostgreSQL 实现在 `lib/store/pg/` 子包
- 业务类型独立于 generated 类型（`pgtype.Timestamptz` → `time.Time`）
- 泛型 `ListResult[T]` 统一分页返回

## 目录结构

```
lib/store/
├── store.go              # Store 聚合接口 + 构造函数签名
├── types.go              # 业务层参数/返回类型
├── user.go               # UserStore 接口
├── namespace.go          # NamespaceStore 接口
├── user_namespace.go     # UserNamespaceStore 接口
├── pg/                   # PostgreSQL 实现
│   ├── store.go          # pgStore（含 WithTx 事务）
│   ├── user.go           # pgUserStore
│   ├── namespace.go      # pgNamespaceStore
│   └── user_namespace.go # pgUserNamespaceStore
└── example/
    └── main.go           # 示例代码
```

## 接口设计

### Store 聚合接口

```go
type Store interface {
    Users() UserStore
    Namespaces() NamespaceStore
    UserNamespaces() UserNamespaceStore
    WithTx(ctx context.Context, fn func(Store) error) error
    Close()
}
```

### UserStore

Create, GetByID, GetByUsername, GetByEmail, Update, UpdateLastLogin, Delete, List（带分页+关联namespace名称）

### NamespaceStore

Create（自动加owner为成员）, GetByID, GetByName, Update, Delete, List（带分页+关联owner用户名）

### UserNamespaceStore

Add, Remove, UpdateRole, Get, ListByUserID, ListByNamespaceID

## 业务类型

- `User`, `Namespace`, `UserNamespaceRole` — 基础模型，用 `time.Time` 替代 `pgtype.Timestamptz`
- `UserWithNamespaces`, `NamespaceWithOwner`, `NamespaceWithRole`, `UserWithRole` — 关联查询返回
- `ListResult[T]` — 泛型分页结果（Items + TotalCount）
- `Pagination` — 通用分页参数（Page从1开始，pg层转offset）
- `CreateUserParams`, `UpdateUserParams` 等 — 业务层参数类型

## 业务逻辑

- `NamespaceStore.Create`: 在事务中创建 namespace 并自动将 owner 加入 user_namespaces（role=owner）
- `Pagination.Page` 从 1 开始，pg 实现内部转换为 `offset = (page-1) * pageSize`
- `pgtype.Timestamptz` → `time.Time` / `*time.Time` 转换在 pg 层完成
- LIKE 搜索参数自动调用 `db.EscapeLike` 转义
