package user

import (
	"strconv"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"

	userstore "lcp.io/lcp/pkg/modules/user/store"
)

func userToAPI(u *userstore.User) *User {
	createdAt := u.CreatedAt
	updatedAt := u.UpdatedAt
	return &User{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "User"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(u.ID, 10),
			Name:      u.Username,
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
		},
		Spec: UserSpec{
			Username:    u.Username,
			Email:       u.Email,
			DisplayName: u.DisplayName,
			Phone:       u.Phone,
			AvatarURL:   u.AvatarUrl,
			Status:      u.Status,
		},
	}
}

func userWithNamespacesToAPI(u *userstore.UserWithNamespaces) *User {
	return userToAPI(&u.User)
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
