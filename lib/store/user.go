package store

import "context"

// UserStore defines operations on users.
type UserStore interface {
	Create(ctx context.Context, user *User) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	UpdateLastLogin(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query ListQuery) (*ListResult[UserWithNamespaces], error)
}
