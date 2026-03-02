# REST Handler 重构设计文档

## 概述

将 `lib/rest/handler.go` 中的通用 `Handle` 函数重构为参考 k8s-apiserver 实现的独立 CRUD 操作函数，并为 User 资源实现完整的增删改查功能。

## 设计目标

1. 将通用的 `Handle()` 函数拆分为 7 个独立的 handler 函数
2. 引入 RESTStorage 接口模式，实现资源操作的标准化
3. 将验证逻辑提升到 handler 层，通过参数传入
4. 为 User 资源实现完整的 CRUD 操作（Get、List、Create、Update、Patch、Delete、DeleteCollection）
5. 补充对应的 sqlc 查询

## 整体架构

### 1. RESTStorage 接口层 (`lib/rest/storage.go`)

定义标准 CRUD 操作接口，采用可组合的接口设计：

```go
// ValidateObjectFunc 验证函数类型
type ValidateObjectFunc func(ctx context.Context, obj runtime.Object) error

// Getter 处理 GET 单个资源
type Getter interface {
    Get(ctx context.Context, id string) (runtime.Object, error)
}

// Lister 处理 GET 集合（支持过滤/分页/排序）
type Lister interface {
    List(ctx context.Context, options *ListOptions) (runtime.Object, error)
}

// Creater 处理 POST 创建
type Creater interface {
    Create(ctx context.Context, obj runtime.Object, validate ValidateObjectFunc, options *CreateOptions) (runtime.Object, error)
}

// Updater 处理 PUT 完整替换
type Updater interface {
    Update(ctx context.Context, id string, obj runtime.Object, validate ValidateObjectFunc, options *UpdateOptions) (runtime.Object, error)
}

// Patcher 处理 PATCH 部分更新（合并非空字段）
type Patcher interface {
    Patch(ctx context.Context, id string, obj runtime.Object, validate ValidateObjectFunc, options *PatchOptions) (runtime.Object, error)
}

// Deleter 处理 DELETE 单个资源
type Deleter interface {
    Delete(ctx context.Context, id string, validate ValidateObjectFunc, options *DeleteOptions) error
}

// CollectionDeleter 处理批量删除（通过显式 ID 列表）
type CollectionDeleter interface {
    DeleteCollection(ctx context.Context, ids []string, validate ValidateObjectFunc, options *DeleteOptions) (*DeletionResult, error)
}

// StandardStorage 组合所有操作
type StandardStorage interface {
    Getter
    Lister
    Creater
    Updater
    Patcher
    Deleter
    CollectionDeleter
}
```

**设计要点：**
- 可组合接口：资源可以只实现需要的操作
- 验证函数作为参数：验证逻辑在上层定义，Storage 层只负责数据操作
- 统一的 ValidateObjectFunc：Update/Patch 如需旧对象，在验证函数内部查询

### 2. 通用 Handler 函数层 (`lib/rest/handler.go`)

替换现有的 `Handle()` 函数，新增 7 个独立的 handler 函数：

#### CreateResource
- 从请求 body 反序列化对象
- 调用 `storage.Create(ctx, obj, validateFunc, options)`
- 返回 201 Created 和创建的对象

#### GetResource
- 从路径参数提取 ID
- 调用 `storage.Get(ctx, id)`
- 返回 200 OK 和对象

#### ListResource
- 从 query 参数解析 ListOptions（过滤、分页、排序）
- 调用 `storage.List(ctx, options)`
- 返回 200 OK 和列表对象

#### UpdateResource
- 从路径参数提取 ID
- 从请求 body 反序列化完整对象
- 调用 `storage.Update(ctx, id, obj, validateFunc, options)`
- 返回 200 OK 和更新后的对象

#### PatchResource
- 从路径参数提取 ID
- 从请求 body 反序列化部分对象
- 调用 `storage.Patch(ctx, id, obj, validateFunc, options)`
- 返回 200 OK 和更新后的对象

#### DeleteResource
- 从路径参数提取 ID
- 调用 `storage.Delete(ctx, id, validateFunc, options)`
- 返回 204 No Content

#### DeleteCollection
- 从请求 body 解析 ID 列表 `{"ids": ["1", "2", "3"]}`
- 调用 `storage.DeleteCollection(ctx, ids, validateFunc, options)`
- 返回 200 OK 和删除结果摘要

**统一特性：**
- 使用 RequestScope 进行错误处理和内容协商
- 支持 JSON/YAML 序列化
- 统一的错误响应格式

### 3. 资源 Storage 实现层 (`app/lcp-server/handler/user_storage.go`)

为每个资源实现 RESTStorage 接口：

```go
type userStorage struct {
    svc *service.Service
}

func (s *userStorage) Create(ctx context.Context, obj runtime.Object, validate ValidateObjectFunc, options *CreateOptions) (runtime.Object, error) {
    // 1. 类型断言
    // 2. 调用验证函数
    // 3. 调用 service 层
    // 4. 返回结果
}

// 实现其他接口方法...
```

**职责：**
- 实现 RESTStorage 接口
- 类型转换（runtime.Object ↔ types.User）
- 调用 service 层
- 错误处理

### 4. 路由注册层 (`app/lcp-server/handler/handler.go`)

更新路由注册，使用新的 handler 函数：

```go
userStorage := newUserStorage(a.svc)

ws.Route(ws.POST("/users").To(rest.CreateResource(scope, userStorage, validateUserCreate)))
ws.Route(ws.GET("/users").To(rest.ListResource(scope, userStorage)))
ws.Route(ws.GET("/users/{userId}").To(rest.GetResource(scope, userStorage)))
ws.Route(ws.PUT("/users/{userId}").To(rest.UpdateResource(scope, userStorage, validateUserUpdate)))
ws.Route(ws.PATCH("/users/{userId}").To(rest.PatchResource(scope, userStorage, validateUserPatch)))
ws.Route(ws.DELETE("/users/{userId}").To(rest.DeleteResource(scope, userStorage, validateUserDelete)))
ws.Route(ws.DELETE("/users").To(rest.DeleteCollection(scope, userStorage, validateUserDelete)))
```

## 数据层实现

### sqlc 查询补充 (`lib/db/query/user.sql`)

**现有查询：**
- ✅ CreateUser
- ✅ GetUserByID
- ✅ GetUserByUsername
- ✅ GetUserByEmail
- ✅ UpdateUser (用于 PUT)
- ✅ DeleteUser
- ✅ ListUsers

**需要新增的查询：**

#### 1. PatchUser - 部分更新（只更新非空字段）
```sql
-- name: PatchUser :one
UPDATE users
SET username = COALESCE(sqlc.narg('username'), username),
    email = COALESCE(sqlc.narg('email'), email),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    phone = COALESCE(sqlc.narg('phone'), phone),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = now()
WHERE id = @id
RETURNING id, username, email, display_name, phone, avatar_url, status,
          last_login_at, created_at, updated_at;
```

#### 2. DeleteUsersByIDs - 批量删除
```sql
-- name: DeleteUsersByIDs :exec
DELETE FROM users WHERE id = ANY(@ids::BIGINT[]);
```

#### 3. GetUsersByIDs - 验证 ID 存在性
```sql
-- name: GetUsersByIDs :many
SELECT id FROM users WHERE id = ANY(@ids::BIGINT[]);
```

**PUT vs PATCH 的区别：**
- **UpdateUser (PUT)**: 直接 SET 所有字段，支持设置为 NULL，完整替换
- **PatchUser (PATCH)**: 使用 COALESCE，只更新提供的非空字段，保持其他字段不变

### Store 层接口补充 (`lib/store/user.go`)

```go
type UserStore interface {
    Create(ctx context.Context, user *User) (*User, error)
    GetByID(ctx context.Context, id int64) (*User, error)
    GetByUsername(ctx context.Context, username string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, user *User) (*User, error)          // PUT - 完整替换
    Patch(ctx context.Context, id int64, user *User) (*User, error) // PATCH - 部分更新
    UpdateLastLogin(ctx context.Context, id int64) error
    Delete(ctx context.Context, id int64) error
    DeleteByIDs(ctx context.Context, ids []int64) (int64, error)    // 批量删除，返回删除数量
    List(ctx context.Context, query ListQuery) (*ListResult[UserWithNamespaces], error)
}
```

### Store 层实现补充 (`lib/store/pg/user.go`)

#### Patch 实现
```go
func (s *pgUserStore) Patch(ctx context.Context, id int64, user *User) (*User, error) {
    // 将零值字段转为 nil，让 COALESCE 保持原值
    row, err := s.queries.PatchUser(ctx, generated.PatchUserParams{
        ID:          id,
        Username:    toNullString(user.Username),
        Email:       toNullString(user.Email),
        DisplayName: toNullString(user.DisplayName),
        Phone:       toNullString(user.Phone),
        AvatarUrl:   toNullString(user.AvatarUrl),
        Status:      toNullString(user.Status),
    })
    if err != nil {
        return nil, fmt.Errorf("patch user: %w", err)
    }
    return &row, nil
}
```

#### DeleteByIDs 实现
```go
func (s *pgUserStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
    // 1. 先查询存在的 ID
    existingIDs, err := s.queries.GetUsersByIDs(ctx, ids)
    if err != nil {
        return 0, fmt.Errorf("get users by ids: %w", err)
    }

    // 2. 执行删除
    if len(existingIDs) > 0 {
        if err := s.queries.DeleteUsersByIDs(ctx, ids); err != nil {
            return 0, fmt.Errorf("delete users by ids: %w", err)
        }
    }

    return int64(len(existingIDs)), nil
}
```

### Service 层补充 (`lib/service/user.go`)

Service 层不再负责验证，只负责：
- ID 格式转换（string → int64）
- 调用 store 层
- 类型转换（store.User ↔ types.User）
- 错误包装

```go
// ListUsers - 列表查询
func (u *UserService) ListUsers(ctx context.Context, query *ListQuery) (*types.UserList, error)

// UpdateUser - PUT 完整替换
func (u *UserService) UpdateUser(ctx context.Context, id string, user *types.User) (*types.User, error)

// PatchUser - PATCH 部分更新
func (u *UserService) PatchUser(ctx context.Context, id string, user *types.User) (*types.User, error)

// DeleteUser - 删除单个用户
func (u *UserService) DeleteUser(ctx context.Context, id string) error

// DeleteUsers - 批量删除用户
func (u *UserService) DeleteUsers(ctx context.Context, ids []string) (*DeletionResult, error)
```

## 数据结构

### DeletionResult
```go
type DeletionResult struct {
    SuccessCount int      `json:"successCount"`
    FailedCount  int      `json:"failedCount"`
    FailedIDs    []string `json:"failedIds,omitempty"`
}
```

### ListOptions
```go
type ListOptions struct {
    Filters    map[string]string  // 过滤条件
    Pagination Pagination         // 分页参数
    SortBy     string            // 排序字段
    SortOrder  string            // 排序方向
}
```

### CreateOptions / UpdateOptions / PatchOptions / DeleteOptions
```go
type CreateOptions struct {
    DryRun bool  // 是否只验证不执行
}

type UpdateOptions struct {
    DryRun bool
}

type PatchOptions struct {
    DryRun bool
}

type DeleteOptions struct {
    DryRun bool
}
```

## 实现顺序

1. **定义接口和类型** (`lib/rest/storage.go`, `lib/rest/options.go`)
   - RESTStorage 接口
   - ValidateObjectFunc 类型
   - Options 结构体
   - DeletionResult 结构体

2. **补充 sqlc 查询** (`lib/db/query/user.sql`)
   - PatchUser
   - DeleteUsersByIDs
   - GetUsersByIDs
   - 运行 `sqlc generate`

3. **更新 Store 层** (`lib/store/user.go`, `lib/store/pg/user.go`)
   - 接口新增 Patch 和 DeleteByIDs 方法
   - 实现 Patch 和 DeleteByIDs

4. **更新 Service 层** (`lib/service/user.go`)
   - 新增 ListUsers、UpdateUser、PatchUser、DeleteUser、DeleteUsers 方法
   - 移除验证逻辑

5. **实现通用 Handler 函数** (`lib/rest/handler.go`)
   - CreateResource
   - GetResource
   - ListResource
   - UpdateResource
   - PatchResource
   - DeleteResource
   - DeleteCollection

6. **实现 User Storage** (`app/lcp-server/handler/user_storage.go`)
   - 实现 StandardStorage 接口
   - 定义验证函数

7. **更新路由注册** (`app/lcp-server/handler/handler.go`, `app/lcp-server/handler/user.go`)
   - 使用新的 handler 函数
   - 注册完整的 User CRUD 路由

8. **测试验证**
   - 单元测试
   - 集成测试
   - API 测试

## 关键设计决策

1. **接口可组合性**: 使用小接口组合，资源可以选择性实现操作
2. **验证逻辑上移**: 验证函数作为参数传入，Storage 层保持纯粹
3. **PUT vs PATCH 语义分离**: PUT 完整替换（可设 NULL），PATCH 部分更新（保持原值）
4. **DeleteCollection 安全性**: 使用显式 ID 列表而非过滤器，避免误删
5. **统一错误处理**: 通过 RequestScope 统一处理错误和内容协商
6. **类型安全**: 通过 runtime.Object 接口保持类型灵活性，在实现层做类型断言

## 兼容性

- 现有的 `Handle()` 函数保持不变，逐步迁移
- 新旧 handler 可以共存
- 路由逐个迁移，降低风险
