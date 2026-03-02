package store

import "context"

// UserStore defines operations on users.
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
