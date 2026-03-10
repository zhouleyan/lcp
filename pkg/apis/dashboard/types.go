package dashboard

import (
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db/generated"
)

// DB type aliases.
type (
	DBPlatformStats  = generated.GetPlatformStatsRow
	DBWorkspaceStats = generated.GetWorkspaceStatsRow
	DBNamespaceStats = generated.GetNamespaceStatsRow
)

// Overview represents scope-level statistics.
// Each API level populates the fields relevant to its scope; unused fields are zero-valued.
// +openapi:description=概览统计
type Overview struct {
	runtime.TypeMeta `json:",inline"`
	Spec             OverviewSpec `json:"spec"`
}

func (o *Overview) GetTypeMeta() *runtime.TypeMeta { return &o.TypeMeta }

type OverviewSpec struct {
	// +openapi:description=租户总数
	WorkspaceCount int64 `json:"workspaceCount"`
	// +openapi:description=项目总数
	NamespaceCount int64 `json:"namespaceCount"`
	// +openapi:description=用户总数
	UserCount int64 `json:"userCount"`
	// +openapi:description=成员数量
	MemberCount int64 `json:"memberCount"`
	// +openapi:description=角色数量
	RoleCount int64 `json:"roleCount"`
}
