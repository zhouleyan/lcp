# REST Handler 重构实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标:** 将 lib/rest/handler.go 中的通用 Handle 函数重构为 k8s-apiserver 风格的独立 CRUD 操作函数，并为 User 资源实现完整的增删改查功能。

**架构:** 引入 RESTStorage 接口模式，将验证逻辑提升到 handler 层，通过可组合的接口实现资源操作标准化。每个资源实现 RESTStorage 接口，handler 函数调用接口方法完成操作。

**技术栈:** Go 1.21+, sqlc, PostgreSQL, 现有的 runtime/rest/service 架构

---

## 任务 1: 定义 RESTStorage 接口和选项类型

**文件:**
- 创建: `lib/rest/storage.go`
- 创建: `lib/rest/options.go`

**步骤 1: 创建 storage.go 定义接口**

```go
package rest

import (
	"context"

	"lcp.io/lcp/lib/runtime"
)

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

// DeletionResult 批量删除结果
type DeletionResult struct {
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
	FailedIDs    []string `json:"failedIds,omitempty"`
}
```

**步骤 2: 创建 options.go 定义选项类型**

```go
package rest

// CreateOptions 创建选项
type CreateOptions struct {
	DryRun bool // 是否只验证不执行
}

// UpdateOptions 更新选项
type UpdateOptions struct {
	DryRun bool
}

// PatchOptions 补丁选项
type PatchOptions struct {
	DryRun bool
}

// DeleteOptions 删除选项
type DeleteOptions struct {
	DryRun bool
}

// ListOptions 列表查询选项
type ListOptions struct {
	Filters    map[string]string // 过滤条件
	Pagination Pagination        // 分页参数
	SortBy     string            // 排序字段
	SortOrder  string            // 排序方向 (asc/desc)
}

// Pagination 分页参数
type Pagination struct {
	Page     int // 页码，从 1 开始
	PageSize int // 每页大小
}
```

**步骤 3: 提交**

```bash
git add lib/rest/storage.go lib/rest/options.go
git commit -m "feat(rest): add RESTStorage interfaces and options types

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 任务 2: 补充 sqlc 查询

**文件:**
- 修改: `lib/db/query/user.sql`

**步骤 1: 添加 PatchUser 查询**

在 `lib/db/query/user.sql` 文件末尾添加：

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

**步骤 2: 添加批量删除查询**

继续在文件末尾添加：

```sql
-- name: DeleteUsersByIDs :exec
DELETE FROM users WHERE id = ANY(@ids::BIGINT[]);

-- name: GetUsersByIDs :many
SELECT id FROM users WHERE id = ANY(@ids::BIGINT[]);
```

**步骤 3: 运行 sqlc generate**

```bash
cd /Users/zhouleyan/Projects/lcp
sqlc generate
```

预期输出: 成功生成代码，无错误

**步骤 4: 提交**

```bash
git add lib/db/query/user.sql lib/db/generated/
git commit -m "feat(db): add PatchUser and batch delete queries

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 任务 3: 更新 Store 层接口和实现

**文件:**
- 修改: `lib/store/user.go`
- 修改: `lib/store/pg/user.go`

**步骤 1: 更新 UserStore 接口**

在 `lib/store/user.go` 中的 UserStore 接口添加新方法：

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
	DeleteByIDs(ctx context.Context, ids []int64) (int64, error) // 批量删除，返回删除数量
	List(ctx context.Context, query ListQuery) (*ListResult[UserWithNamespaces], error)
}
```

**步骤 2: 在 pg/user.go 添加辅助函数**

在 `lib/store/pg/user.go` 文件顶部添加辅助函数：

```go
// toNullString 将字符串转为 sql.NullString，空字符串转为 NULL
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
```

**步骤 3: 实现 Patch 方法**

在 `lib/store/pg/user.go` 的 pgUserStore 添加 Patch 方法：

```go
func (s *pgUserStore) Patch(ctx context.Context, id int64, user *User) (*User, error) {
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

**步骤 4: 实现 DeleteByIDs 方法**

继续在 pgUserStore 添加 DeleteByIDs 方法：

```go
func (s *pgUserStore) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	// 先查询存在的 ID
	existingIDs, err := s.queries.GetUsersByIDs(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("get users by ids: %w", err)
	}

	// 执行删除
	if len(existingIDs) > 0 {
		if err := s.queries.DeleteUsersByIDs(ctx, ids); err != nil {
			return 0, fmt.Errorf("delete users by ids: %w", err)
		}
	}

	return int64(len(existingIDs)), nil
}
```

**步骤 5: 添加必要的 import**

确保 `lib/store/pg/user.go` 有以下 import：

```go
import (
	"context"
	"database/sql"
	"fmt"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)
```

**步骤 6: 提交**

```bash
git add lib/store/user.go lib/store/pg/user.go
git commit -m "feat(store): add Patch and DeleteByIDs methods to UserStore

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 任务 4: 更新 Service 层

**文件:**
- 修改: `lib/service/user.go`
- 修改: `lib/service/types.go` (如果需要新增类型)

**步骤 1: 添加 ListUsers 方法**

在 `lib/service/user.go` 的 UserService 添加：

```go
func (u *UserService) ListUsers(ctx context.Context, filters map[string]string, pagination store.Pagination, sortBy, sortOrder string) (runtime.Object, error) {
	query := store.ListQuery{
		Filters:    filters,
		Pagination: pagination,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
	}

	result, err := u.s.store.Users().List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]types.User, len(result.Items))
	for i, item := range result.Items {
		items[i] = *userWithNamespacesToAPI(&item)
	}

	return &types.UserList{
		TypeMeta: types.TypeMeta{
			Kind:       "UserList",
			APIVersion: "v1",
		},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}
```

**步骤 2: 添加 UpdateUser 方法**

```go
func (u *UserService) UpdateUser(ctx context.Context, id string, user *types.User) (*types.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	updated, err := u.s.store.Users().Update(ctx, &store.User{
		ID:          uid,
		Username:    user.Spec.Username,
		Email:       user.Spec.Email,
		DisplayName: user.Spec.DisplayName,
		Phone:       user.Spec.Phone,
		AvatarUrl:   user.Spec.AvatarURL,
		Status:      user.Spec.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return userToAPI(updated), nil
}
```

**步骤 3: 添加 PatchUser 方法**

```go
func (u *UserService) PatchUser(ctx context.Context, id string, user *types.User) (*types.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	patched, err := u.s.store.Users().Patch(ctx, uid, &store.User{
		Username:    user.Spec.Username,
		Email:       user.Spec.Email,
		DisplayName: user.Spec.DisplayName,
		Phone:       user.Spec.Phone,
		AvatarUrl:   user.Spec.AvatarURL,
		Status:      user.Spec.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("patch user: %w", err)
	}

	return userToAPI(patched), nil
}
```

**步骤 4: 添加 DeleteUser 方法**

```go
func (u *UserService) DeleteUser(ctx context.Context, id string) error {
	uid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
	}

	if err := u.s.store.Users().Delete(ctx, uid); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}
```

**步骤 5: 添加 DeleteUsers 方法**

```go
func (u *UserService) DeleteUsers(ctx context.Context, ids []string) (*DeletionResult, error) {
	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		uid, err := parseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid user ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, uid)
	}

	count, err := u.s.store.Users().DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, fmt.Errorf("delete users: %w", err)
	}

	return &DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}
```

**步骤 6: 添加 DeletionResult 类型**

在 `lib/service/types.go` 或 `lib/service/user.go` 顶部添加：

```go
// DeletionResult 批量删除结果
type DeletionResult struct {
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
	FailedIDs    []string `json:"failedIds,omitempty"`
}
```

**步骤 7: 添加辅助函数 userWithNamespacesToAPI**

```go
func userWithNamespacesToAPI(u *store.UserWithNamespaces) *types.User {
	user := userToAPI(&u.User)
	// 可以在这里添加 namespace 信息到 user 对象
	return user
}
```

**步骤 8: 提交**

```bash
git add lib/service/user.go lib/service/types.go
git commit -m "feat(service): add List/Update/Patch/Delete methods to UserService

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

