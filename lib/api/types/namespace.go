package types

import "lcp.io/lcp/lib/runtime"

// Namespace is the API representation of a namespace resource.
type Namespace struct {
	runtime.TypeMeta `json:",inline"`
	ObjectMeta       `json:"metadata"`
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

// NamespaceMember is the API representation for adding a member to a namespace.
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
