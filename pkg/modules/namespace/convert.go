package namespace

import (
	"strconv"

	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"

	nsstore "lcp.io/lcp/pkg/modules/namespace/store"
)

func namespaceToAPI(n *nsstore.Namespace) *Namespace {
	createdAt := n.CreatedAt
	updatedAt := n.UpdatedAt
	return &Namespace{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(n.ID, 10),
			Name:      n.Name,
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
		},
		Spec: NamespaceSpec{
			DisplayName: n.DisplayName,
			Description: n.Description,
			OwnerID:     strconv.FormatInt(n.OwnerID, 10),
			Visibility:  n.Visibility,
			MaxMembers:  int(n.MaxMembers),
			Status:      n.Status,
		},
	}
}

func memberToAPI(r *nsstore.UserNamespaceRole) *NamespaceMember {
	return &NamespaceMember{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "NamespaceMember"},
		Spec: NamespaceMemberSpec{
			UserID: strconv.FormatInt(r.UserID, 10),
			Role:   r.Role,
		},
	}
}

func parseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
