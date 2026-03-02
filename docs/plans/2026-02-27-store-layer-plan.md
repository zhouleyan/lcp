# Store Layer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a business abstraction layer (`lib/store`) that wraps sqlc-generated queries with interfaces, business types, and logic.

**Architecture:** Interfaces defined in `lib/store/`, PostgreSQL implementation in `lib/store/pg/`. Business types decouple upper layers from `generated` package. `pgtype.Timestamptz` converted to `time.Time` at the pg layer. `WithTx` provides transactional Store instances.

**Tech Stack:** Go 1.26, pgx/v5, sqlc generated code from `lib/db/generated`

**Design doc:** `docs/plans/2026-02-27-store-layer-design.md`

---

### Task 1: Create business types

**Files:**
- Create: `lib/store/types.go`

**Step 1: Write types.go**

```go
package store

import "time"

// Pagination holds common pagination and sorting parameters.
type Pagination struct {
	Page      int    `json:"page"`       // starts from 1
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"` // "asc" or "desc"
}

// ListResult is a generic paginated result.
type ListResult[T any] struct {
	Items      []T   `json:"items"`
	TotalCount int64 `json:"total_count"`
}

// User represents a user business model.
type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Phone       string     `json:"phone"`
	AvatarUrl   string     `json:"avatar_url"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UserWithNamespaces extends User with associated namespace names.
type UserWithNamespaces struct {
	User
	NamespaceNames []string `json:"namespace_names"`
}

// Namespace represents a namespace business model.
type Namespace struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	OwnerID     int64     `json:"owner_id"`
	Visibility  string    `json:"visibility"`
	MaxMembers  int32     `json:"max_members"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NamespaceWithOwner extends Namespace with owner username.
type NamespaceWithOwner struct {
	Namespace
	OwnerUsername string `json:"owner_username"`
}

// UserNamespaceRole represents a user's membership in a namespace.
type UserNamespaceRole struct {
	UserID      int64     `json:"user_id"`
	NamespaceID int64     `json:"namespace_id"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// NamespaceWithRole is a namespace with the user's role in it.
type NamespaceWithRole struct {
	Namespace
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// UserWithRole is a user with their role in a namespace.
type UserWithRole struct {
	User
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// CreateUserParams holds parameters for creating a user.
type CreateUserParams struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
	AvatarUrl   string `json:"avatar_url"`
	Status      string `json:"status"`
}

// UpdateUserParams holds parameters for updating a user.
type UpdateUserParams struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
	AvatarUrl   string `json:"avatar_url"`
	Status      string `json:"status"`
}

// ListUsersParams holds parameters for listing users.
type ListUsersParams struct {
	Status      *string `json:"status,omitempty"`
	Username    *string `json:"username,omitempty"`
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Pagination
}

// CreateNamespaceParams holds parameters for creating a namespace.
type CreateNamespaceParams struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OwnerID     int64  `json:"owner_id"`
	Visibility  string `json:"visibility"`
	MaxMembers  int32  `json:"max_members"`
	Status      string `json:"status"`
}

// UpdateNamespaceParams holds parameters for updating a namespace.
type UpdateNamespaceParams struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OwnerID     int64  `json:"owner_id"`
	Visibility  string `json:"visibility"`
	MaxMembers  int32  `json:"max_members"`
	Status      string `json:"status"`
}

// ListNamespacesParams holds parameters for listing namespaces.
type ListNamespacesParams struct {
	Status     *string `json:"status,omitempty"`
	Name       *string `json:"name,omitempty"`
	Visibility *string `json:"visibility,omitempty"`
	OwnerID    *int64  `json:"owner_id,omitempty"`
	Pagination
}

// AddUserNamespaceParams holds parameters for adding a user to a namespace.
type AddUserNamespaceParams struct {
	UserID      int64  `json:"user_id"`
	NamespaceID int64  `json:"namespace_id"`
	Role        string `json:"role"`
}

// UpdateRoleParams holds parameters for updating a user's role in a namespace.
type UpdateRoleParams struct {
	UserID      int64  `json:"user_id"`
	NamespaceID int64  `json:"namespace_id"`
	Role        string `json:"role"`
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/zhouleyan/Projects/lcp && go build ./lib/store/`

**Step 3: Commit**

```bash
git add lib/store/types.go
git commit -m "feat(store): add business types for store layer"
```

---

### Task 2: Create store interfaces

**Files:**
- Create: `lib/store/user.go`
- Create: `lib/store/namespace.go`
- Create: `lib/store/user_namespace.go`
- Create: `lib/store/store.go`

**Step 1: Write user.go**

```go
package store

import "context"

// UserStore defines operations on users.
type UserStore interface {
	Create(ctx context.Context, params CreateUserParams) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, params UpdateUserParams) (*User, error)
	UpdateLastLogin(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params ListUsersParams) (*ListResult[UserWithNamespaces], error)
}
```

**Step 2: Write namespace.go**

```go
package store

import "context"

// NamespaceStore defines operations on namespaces.
type NamespaceStore interface {
	Create(ctx context.Context, params CreateNamespaceParams) (*Namespace, error)
	GetByID(ctx context.Context, id int64) (*Namespace, error)
	GetByName(ctx context.Context, name string) (*Namespace, error)
	Update(ctx context.Context, params UpdateNamespaceParams) (*Namespace, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params ListNamespacesParams) (*ListResult[NamespaceWithOwner], error)
}
```

**Step 3: Write user_namespace.go**

```go
package store

import "context"

// UserNamespaceStore defines operations on user-namespace relationships.
type UserNamespaceStore interface {
	Add(ctx context.Context, params AddUserNamespaceParams) (*UserNamespaceRole, error)
	Remove(ctx context.Context, userID, namespaceID int64) error
	UpdateRole(ctx context.Context, params UpdateRoleParams) (*UserNamespaceRole, error)
	Get(ctx context.Context, userID, namespaceID int64) (*UserNamespaceRole, error)
	ListByUserID(ctx context.Context, userID int64) ([]NamespaceWithRole, error)
	ListByNamespaceID(ctx context.Context, namespaceID int64) ([]UserWithRole, error)
}
```

**Step 4: Write store.go**

```go
package store

import "context"

// Store is the top-level interface aggregating all sub-stores.
type Store interface {
	Users() UserStore
	Namespaces() NamespaceStore
	UserNamespaces() UserNamespaceStore
	WithTx(ctx context.Context, fn func(Store) error) error
	Close()
}
```

**Step 5: Verify it compiles**

Run: `cd /Users/zhouleyan/Projects/lcp && go build ./lib/store/`

**Step 6: Commit**

```bash
git add lib/store/user.go lib/store/namespace.go lib/store/user_namespace.go lib/store/store.go
git commit -m "feat(store): add Store, UserStore, NamespaceStore, UserNamespaceStore interfaces"
```

---

### Task 3: Create pg store implementation (core + helpers)

**Files:**
- Create: `lib/store/pg/store.go`

**Step 1: Write pg/store.go**

```go
package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

// pgStore implements store.Store backed by PostgreSQL.
type pgStore struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// New creates a new PostgreSQL-backed Store.
func New(pool *pgxpool.Pool) store.Store {
	return &pgStore{
		pool:    pool,
		queries: generated.New(pool),
	}
}

func (s *pgStore) Users() store.UserStore {
	return &pgUserStore{queries: s.queries}
}

func (s *pgStore) Namespaces() store.NamespaceStore {
	return &pgNamespaceStore{queries: s.queries}
}

func (s *pgStore) UserNamespaces() store.UserNamespaceStore {
	return &pgUserNamespaceStore{queries: s.queries}
}

func (s *pgStore) WithTx(ctx context.Context, fn func(store.Store) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txStore := &pgStore{
		pool:    s.pool,
		queries: s.queries.WithTx(tx),
	}

	if err := fn(txStore); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *pgStore) Close() {
	s.pool.Close()
}

// Helper: convert pgtype.Timestamptz to time.Time
func toTime(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}

// Helper: convert pgtype.Timestamptz to *time.Time
func toTimePtr(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

// Helper: convert Pagination to offset and ensure defaults
func paginationToOffsetLimit(p store.Pagination) (offset int32, limit int32) {
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return int32((page - 1) * pageSize), int32(pageSize)
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/zhouleyan/Projects/lcp && go build ./lib/store/pg/`

**Step 3: Commit**

```bash
git add lib/store/pg/store.go
git commit -m "feat(store): add pgStore core with WithTx and helpers"
```

---

### Task 4: Implement pgUserStore

**Files:**
- Create: `lib/store/pg/user.go`

**Step 1: Write pg/user.go**

```go
package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

type pgUserStore struct {
	queries *generated.Queries
}

func (s *pgUserStore) Create(ctx context.Context, params store.CreateUserParams) (*store.User, error) {
	row, err := s.queries.CreateUser(ctx, generated.CreateUserParams{
		Username:    params.Username,
		Email:       params.Email,
		DisplayName: params.DisplayName,
		Phone:       params.Phone,
		AvatarUrl:   params.AvatarUrl,
		Status:      params.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) GetByID(ctx context.Context, id int64) (*store.User, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) GetByUsername(ctx context.Context, username string) (*store.User, error) {
	row, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	row, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) Update(ctx context.Context, params store.UpdateUserParams) (*store.User, error) {
	row, err := s.queries.UpdateUser(ctx, generated.UpdateUserParams{
		ID:          params.ID,
		Username:    params.Username,
		Email:       params.Email,
		DisplayName: params.DisplayName,
		Phone:       params.Phone,
		AvatarUrl:   params.AvatarUrl,
		Status:      params.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return userFromRow(row), nil
}

func (s *pgUserStore) UpdateLastLogin(ctx context.Context, id int64) error {
	if err := s.queries.UpdateUserLastLogin(ctx, id); err != nil {
		return fmt.Errorf("update user last login: %w", err)
	}
	return nil
}

func (s *pgUserStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *pgUserStore) List(ctx context.Context, params store.ListUsersParams) (*store.ListResult[store.UserWithNamespaces], error) {
	offset, limit := paginationToOffsetLimit(params.Pagination)

	// Escape LIKE params
	var username, email, displayName *string
	if params.Username != nil {
		v := db.EscapeLike(*params.Username)
		username = &v
	}
	if params.Email != nil {
		v := db.EscapeLike(*params.Email)
		email = &v
	}
	if params.DisplayName != nil {
		v := db.EscapeLike(*params.DisplayName)
		displayName = &v
	}

	sortField := params.SortBy
	if sortField == "" {
		sortField = db.UserSortCreatedAt
	}
	sortOrder := params.SortOrder
	if sortOrder == "" {
		sortOrder = db.SortDesc
	}

	count, err := s.queries.CountUsers(ctx, generated.CountUsersParams{
		Status:      params.Status,
		Username:    username,
		Email:       email,
		DisplayName: displayName,
	})
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	rows, err := s.queries.ListUsers(ctx, generated.ListUsersParams{
		Status:      params.Status,
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		SortField:   sortField,
		SortOrder:   sortOrder,
		PageSize:    limit,
		PageOffset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]store.UserWithNamespaces, 0, len(rows))
	for _, row := range rows {
		items = append(items, store.UserWithNamespaces{
			User: store.User{
				ID:          row.ID,
				Username:    row.Username,
				Email:       row.Email,
				DisplayName: row.DisplayName,
				Phone:       row.Phone,
				AvatarUrl:   row.AvatarUrl,
				Status:      row.Status,
				LastLoginAt: toTimePtr(row.LastLoginAt),
				CreatedAt:   toTime(row.CreatedAt),
				UpdatedAt:   toTime(row.UpdatedAt),
			},
			NamespaceNames: row.NamespaceNames,
		})
	}

	return &store.ListResult[store.UserWithNamespaces]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func userFromRow(row generated.User) *store.User {
	return &store.User{
		ID:          row.ID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Phone:       row.Phone,
		AvatarUrl:   row.AvatarUrl,
		Status:      row.Status,
		LastLoginAt: toTimePtr(row.LastLoginAt),
		CreatedAt:   toTime(row.CreatedAt),
		UpdatedAt:   toTime(row.UpdatedAt),
	}
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/zhouleyan/Projects/lcp && go build ./lib/store/pg/`

**Step 3: Commit**

```bash
git add lib/store/pg/user.go
git commit -m "feat(store): implement pgUserStore"
```

---

### Task 5: Implement pgNamespaceStore

**Files:**
- Create: `lib/store/pg/namespace.go`

**Step 1: Write pg/namespace.go**

```go
package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

type pgNamespaceStore struct {
	queries *generated.Queries
}

func (s *pgNamespaceStore) Create(ctx context.Context, params store.CreateNamespaceParams) (*store.Namespace, error) {
	row, err := s.queries.CreateNamespace(ctx, generated.CreateNamespaceParams{
		Name:        params.Name,
		DisplayName: params.DisplayName,
		Description: params.Description,
		OwnerID:     params.OwnerID,
		Visibility:  params.Visibility,
		MaxMembers:  params.MaxMembers,
		Status:      params.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	// Business logic: auto-add owner as member with role "owner"
	_, err = s.queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      params.OwnerID,
		NamespaceID: row.ID,
		Role:        "owner",
	})
	if err != nil {
		return nil, fmt.Errorf("add owner to namespace: %w", err)
	}

	return namespaceFromRow(row), nil
}

func (s *pgNamespaceStore) GetByID(ctx context.Context, id int64) (*store.Namespace, error) {
	row, err := s.queries.GetNamespaceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get namespace by id: %w", err)
	}
	return namespaceFromRow(row), nil
}

func (s *pgNamespaceStore) GetByName(ctx context.Context, name string) (*store.Namespace, error) {
	row, err := s.queries.GetNamespaceByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get namespace by name: %w", err)
	}
	return namespaceFromRow(row), nil
}

func (s *pgNamespaceStore) Update(ctx context.Context, params store.UpdateNamespaceParams) (*store.Namespace, error) {
	row, err := s.queries.UpdateNamespace(ctx, generated.UpdateNamespaceParams{
		ID:          params.ID,
		Name:        params.Name,
		DisplayName: params.DisplayName,
		Description: params.Description,
		OwnerID:     params.OwnerID,
		Visibility:  params.Visibility,
		MaxMembers:  params.MaxMembers,
		Status:      params.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
	}
	return namespaceFromRow(row), nil
}

func (s *pgNamespaceStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteNamespace(ctx, id); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

func (s *pgNamespaceStore) List(ctx context.Context, params store.ListNamespacesParams) (*store.ListResult[store.NamespaceWithOwner], error) {
	offset, limit := paginationToOffsetLimit(params.Pagination)

	var name *string
	if params.Name != nil {
		v := db.EscapeLike(*params.Name)
		name = &v
	}

	sortField := params.SortBy
	if sortField == "" {
		sortField = db.NamespaceSortCreatedAt
	}
	sortOrder := params.SortOrder
	if sortOrder == "" {
		sortOrder = db.SortDesc
	}

	count, err := s.queries.CountNamespaces(ctx, generated.CountNamespacesParams{
		Status:     params.Status,
		Name:       name,
		Visibility: params.Visibility,
		OwnerID:    params.OwnerID,
	})
	if err != nil {
		return nil, fmt.Errorf("count namespaces: %w", err)
	}

	rows, err := s.queries.ListNamespaces(ctx, generated.ListNamespacesParams{
		Status:     params.Status,
		Name:       name,
		Visibility: params.Visibility,
		OwnerID:    params.OwnerID,
		SortField:  sortField,
		SortOrder:  sortOrder,
		PageSize:   limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	items := make([]store.NamespaceWithOwner, 0, len(rows))
	for _, row := range rows {
		items = append(items, store.NamespaceWithOwner{
			Namespace: store.Namespace{
				ID:          row.ID,
				Name:        row.Name,
				DisplayName: row.DisplayName,
				Description: row.Description,
				OwnerID:     row.OwnerID,
				Visibility:  row.Visibility,
				MaxMembers:  row.MaxMembers,
				Status:      row.Status,
				CreatedAt:   toTime(row.CreatedAt),
				UpdatedAt:   toTime(row.UpdatedAt),
			},
			OwnerUsername: row.OwnerUsername,
		})
	}

	return &store.ListResult[store.NamespaceWithOwner]{
		Items:      items,
		TotalCount: count,
	}, nil
}

func namespaceFromRow(row generated.Namespace) *store.Namespace {
	return &store.Namespace{
		ID:          row.ID,
		Name:        row.Name,
		DisplayName: row.DisplayName,
		Description: row.Description,
		OwnerID:     row.OwnerID,
		Visibility:  row.Visibility,
		MaxMembers:  row.MaxMembers,
		Status:      row.Status,
		CreatedAt:   toTime(row.CreatedAt),
		UpdatedAt:   toTime(row.UpdatedAt),
	}
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/zhouleyan/Projects/lcp && go build ./lib/store/pg/`

**Step 3: Commit**

```bash
git add lib/store/pg/namespace.go
git commit -m "feat(store): implement pgNamespaceStore with auto-add owner"
```

---

### Task 6: Implement pgUserNamespaceStore

**Files:**
- Create: `lib/store/pg/user_namespace.go`

**Step 1: Write pg/user_namespace.go**

```go
package pg

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/db/generated"
	"lcp.io/lcp/lib/store"
)

type pgUserNamespaceStore struct {
	queries *generated.Queries
}

func (s *pgUserNamespaceStore) Add(ctx context.Context, params store.AddUserNamespaceParams) (*store.UserNamespaceRole, error) {
	row, err := s.queries.AddUserToNamespace(ctx, generated.AddUserToNamespaceParams{
		UserID:      params.UserID,
		NamespaceID: params.NamespaceID,
		Role:        params.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("add user to namespace: %w", err)
	}
	return userNamespaceFromRow(row), nil
}

func (s *pgUserNamespaceStore) Remove(ctx context.Context, userID, namespaceID int64) error {
	if err := s.queries.RemoveUserFromNamespace(ctx, generated.RemoveUserFromNamespaceParams{
		UserID:      userID,
		NamespaceID: namespaceID,
	}); err != nil {
		return fmt.Errorf("remove user from namespace: %w", err)
	}
	return nil
}

func (s *pgUserNamespaceStore) UpdateRole(ctx context.Context, params store.UpdateRoleParams) (*store.UserNamespaceRole, error) {
	row, err := s.queries.UpdateUserNamespaceRole(ctx, generated.UpdateUserNamespaceRoleParams{
		UserID:      params.UserID,
		NamespaceID: params.NamespaceID,
		Role:        params.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("update user namespace role: %w", err)
	}
	return userNamespaceFromRow(row), nil
}

func (s *pgUserNamespaceStore) Get(ctx context.Context, userID, namespaceID int64) (*store.UserNamespaceRole, error) {
	row, err := s.queries.GetUserNamespace(ctx, generated.GetUserNamespaceParams{
		UserID:      userID,
		NamespaceID: namespaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user namespace: %w", err)
	}
	return userNamespaceFromRow(row), nil
}

func (s *pgUserNamespaceStore) ListByUserID(ctx context.Context, userID int64) ([]store.NamespaceWithRole, error) {
	rows, err := s.queries.ListNamespacesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list namespaces by user: %w", err)
	}

	items := make([]store.NamespaceWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, store.NamespaceWithRole{
			Namespace: store.Namespace{
				ID:          row.ID,
				Name:        row.Name,
				DisplayName: row.DisplayName,
				Description: row.Description,
				OwnerID:     row.OwnerID,
				Visibility:  row.Visibility,
				MaxMembers:  row.MaxMembers,
				Status:      row.Status,
				CreatedAt:   toTime(row.CreatedAt),
				UpdatedAt:   toTime(row.UpdatedAt),
			},
			Role:     row.Role,
			JoinedAt: toTime(row.JoinedAt),
		})
	}
	return items, nil
}

func (s *pgUserNamespaceStore) ListByNamespaceID(ctx context.Context, namespaceID int64) ([]store.UserWithRole, error) {
	rows, err := s.queries.ListUsersByNamespaceID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("list users by namespace: %w", err)
	}

	items := make([]store.UserWithRole, 0, len(rows))
	for _, row := range rows {
		items = append(items, store.UserWithRole{
			User: store.User{
				ID:          row.ID,
				Username:    row.Username,
				Email:       row.Email,
				DisplayName: row.DisplayName,
				Phone:       row.Phone,
				AvatarUrl:   row.AvatarUrl,
				Status:      row.Status,
				LastLoginAt: toTimePtr(row.LastLoginAt),
				CreatedAt:   toTime(row.CreatedAt),
				UpdatedAt:   toTime(row.UpdatedAt),
			},
			Role:     row.Role,
			JoinedAt: toTime(row.JoinedAt),
		})
	}
	return items, nil
}

func userNamespaceFromRow(row generated.UserNamespace) *store.UserNamespaceRole {
	return &store.UserNamespaceRole{
		UserID:      row.UserID,
		NamespaceID: row.NamespaceID,
		Role:        row.Role,
		CreatedAt:   toTime(row.CreatedAt),
	}
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/zhouleyan/Projects/lcp && go build ./lib/store/pg/`

**Step 3: Commit**

```bash
git add lib/store/pg/user_namespace.go
git commit -m "feat(store): implement pgUserNamespaceStore"
```

---

### Task 7: Create store example

**Files:**
- Create: `lib/store/example/main.go`

**Step 1: Write example/main.go**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"lcp.io/lcp/lib/db"
	"lcp.io/lcp/lib/store"
	pg "lcp.io/lcp/lib/store/pg"
)

func main() {
	ctx := context.Background()

	// 1. 创建连接池
	pool, err := db.NewPool(ctx, db.Config{
		Host:     envOrDefault("DB_HOST", "localhost"),
		Port:     5432,
		User:     envOrDefault("DB_USER", "postgres"),
		Password: envOrDefault("DB_PASSWORD", "postgres"),
		DBName:   envOrDefault("DB_NAME", "lcp"),
		SSLMode:  "disable",
		MaxConns: 10,
	})
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	// 2. 创建 Store
	s := pg.New(pool)
	defer s.Close()

	// 3. 创建用户
	alice, err := s.Users().Create(ctx, store.CreateUserParams{
		Username:    "alice",
		Email:       "alice@example.com",
		DisplayName: "Alice Wang",
		Phone:       "13800000001",
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("create alice: %v", err)
	}
	fmt.Printf("Created user: %s (id=%d)\n", alice.Username, alice.ID)

	bob, err := s.Users().Create(ctx, store.CreateUserParams{
		Username:    "bob",
		Email:       "bob@example.com",
		DisplayName: "Bob Li",
		Status:      "active",
	})
	if err != nil {
		log.Fatalf("create bob: %v", err)
	}
	fmt.Printf("Created user: %s (id=%d)\n", bob.Username, bob.ID)

	// 4. 事务：创建 Namespace（自动加 owner 为成员）
	var ns *store.Namespace
	err = s.WithTx(ctx, func(txStore store.Store) error {
		var txErr error
		ns, txErr = txStore.Namespaces().Create(ctx, store.CreateNamespaceParams{
			Name:        "team-alpha",
			DisplayName: "Team Alpha",
			Description: "The alpha team",
			OwnerID:     alice.ID,
			Visibility:  "private",
			MaxMembers:  50,
			Status:      "active",
		})
		if txErr != nil {
			return txErr
		}

		// 在同一事务中添加 bob 为成员
		_, txErr = txStore.UserNamespaces().Add(ctx, store.AddUserNamespaceParams{
			UserID:      bob.ID,
			NamespaceID: ns.ID,
			Role:        "member",
		})
		return txErr
	})
	if err != nil {
		log.Fatalf("create namespace in tx: %v", err)
	}
	fmt.Printf("Created namespace: %s (id=%d)\n", ns.Name, ns.ID)

	// 5. 查询用户所属的 Namespace
	nsList, err := s.UserNamespaces().ListByUserID(ctx, alice.ID)
	if err != nil {
		log.Fatalf("list namespaces by user: %v", err)
	}
	fmt.Printf("\n%s belongs to %d namespace(s):\n", alice.Username, len(nsList))
	for _, n := range nsList {
		fmt.Printf("  - %s (role: %s)\n", n.Name, n.Role)
	}

	// 6. 查询 Namespace 下的用户
	members, err := s.UserNamespaces().ListByNamespaceID(ctx, ns.ID)
	if err != nil {
		log.Fatalf("list users by namespace: %v", err)
	}
	fmt.Printf("\n%s has %d member(s):\n", ns.Name, len(members))
	for _, m := range members {
		fmt.Printf("  - %s (role: %s)\n", m.Username, m.Role)
	}

	// 7. 复杂查询：用户列表（筛选 + 排序 + 分页 + 关联 namespace）
	activeStatus := "active"
	result, err := s.Users().List(ctx, store.ListUsersParams{
		Status: &activeStatus,
		Pagination: store.Pagination{
			Page:      1,
			PageSize:  10,
			SortBy:    "username",
			SortOrder: "asc",
		},
	})
	if err != nil {
		log.Fatalf("list users: %v", err)
	}
	fmt.Printf("\nListUsers (total=%d):\n", result.TotalCount)
	for _, u := range result.Items {
		fmt.Printf("  - %s <%s> namespaces=%v\n", u.Username, u.Email, u.NamespaceNames)
	}

	// 8. 更新角色
	_, err = s.UserNamespaces().UpdateRole(ctx, store.UpdateRoleParams{
		UserID:      bob.ID,
		NamespaceID: ns.ID,
		Role:        "admin",
	})
	if err != nil {
		log.Fatalf("update role: %v", err)
	}
	fmt.Printf("\nUpdated %s's role to admin\n", bob.Username)

	// 9. 删除成员
	err = s.UserNamespaces().Remove(ctx, bob.ID, ns.ID)
	if err != nil {
		log.Fatalf("remove member: %v", err)
	}
	fmt.Printf("Removed %s from %s\n", bob.Username, ns.Name)

	fmt.Println("\nAll store examples completed successfully!")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/zhouleyan/Projects/lcp && go build ./lib/store/example/`

**Step 3: Commit**

```bash
git add lib/store/example/main.go
git commit -m "feat(store): add example demonstrating store layer usage"
```

---

### Task 8: Final verification

**Step 1: Run go vet on store packages**

Run: `cd /Users/zhouleyan/Projects/lcp && go vet ./lib/store/...`

**Step 2: Run go build on entire project**

Run: `cd /Users/zhouleyan/Projects/lcp && go build ./...`

**Step 3: Verify interface compliance**

The go compiler will verify that `pgStore` implements `store.Store`, `pgUserStore` implements `store.UserStore`, etc. If it compiles, the interfaces are satisfied.
