package namespace

import (
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
)

// Namespace is the API representation of a namespace resource.
type Namespace struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             NamespaceSpec `json:"spec"`
}

func (n *Namespace) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// NamespaceSpec holds namespace-specific fields.
type NamespaceSpec struct {
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	OwnerID     string `json:"ownerId"`
	Visibility  string `json:"visibility,omitempty"`
	MaxMembers  int    `json:"maxMembers,omitempty"`
	Status      string `json:"status,omitempty"`
}

// NamespaceList is a paginated list of namespaces.
type NamespaceList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Namespace `json:"items"`
	TotalCount       int64       `json:"totalCount"`
}

func (n *NamespaceList) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// NamespaceMember is the API representation for a member in a namespace.
type NamespaceMember struct {
	runtime.TypeMeta `json:",inline"`
	Spec             NamespaceMemberSpec `json:"spec"`
}

func (n *NamespaceMember) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// NamespaceMemberSpec holds member-specific fields.
type NamespaceMemberSpec struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

// NamespaceMemberList is a list of namespace members.
type NamespaceMemberList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []NamespaceMember `json:"items"`
}

func (n *NamespaceMemberList) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }
