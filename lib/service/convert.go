package service

import (
	"strconv"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/lib/store"
)

func userToAPI(u *store.User) *types.User {
	createdAt := u.CreatedAt
	updatedAt := u.UpdatedAt
	return &types.User{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "User"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(u.ID, 10),
			Name:      u.Username,
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
		},
		Spec: types.UserSpec{
			Username:    u.Username,
			Email:       u.Email,
			DisplayName: u.DisplayName,
			Phone:       u.Phone,
			AvatarURL:   u.AvatarUrl,
			Status:      u.Status,
		},
	}
}

func namespaceToAPI(n *store.Namespace) *types.Namespace {
	createdAt := n.CreatedAt
	updatedAt := n.UpdatedAt
	return &types.Namespace{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(n.ID, 10),
			Name:      n.Name,
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
		},
		Spec: types.NamespaceSpec{
			DisplayName: n.DisplayName,
			Description: n.Description,
			OwnerID:     strconv.FormatInt(n.OwnerID, 10),
			Visibility:  n.Visibility,
			MaxMembers:  int(n.MaxMembers),
			Status:      n.Status,
		},
	}
}

func memberToAPI(r *store.UserNamespaceRole) *types.NamespaceMember {
	return &types.NamespaceMember{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "NamespaceMember"},
		Spec: types.NamespaceMemberSpec{
			UserID: strconv.FormatInt(r.UserID, 10),
			Role:   r.Role,
		},
	}
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
