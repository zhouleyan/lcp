package types

import "lcp.io/lcp/lib/runtime"

// User is the API representation of a user resource.
type User struct {
	runtime.TypeMeta `json:",inline"`
	ObjectMeta       `json:"metadata"`
	Spec             UserSpec `json:"spec"`
}

func (u *User) GetTypeMeta() *runtime.TypeMeta { return &u.TypeMeta }

// UserSpec holds user-specific fields.
type UserSpec struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	Phone       string `json:"phone,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Status      string `json:"status,omitempty"`
}

// UserList 用户列表
type UserList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []User `json:"items"`
	TotalCount       int64  `json:"totalCount"`
}

// GetObjectKind 实现 runtime.Object
func (u *UserList) GetObjectKind() *runtime.TypeMeta {
	return &u.TypeMeta
}
