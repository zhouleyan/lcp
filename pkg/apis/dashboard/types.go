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

// PlatformOverview represents platform-level statistics.
// +openapi:description=平台级概览统计
type PlatformOverview struct {
	runtime.TypeMeta `json:",inline"`
	Spec             PlatformOverviewSpec `json:"spec"`
}

func (o *PlatformOverview) GetTypeMeta() *runtime.TypeMeta { return &o.TypeMeta }

type PlatformOverviewSpec struct {
	// +openapi:description=租户总数
	WorkspaceCount int64 `json:"workspaceCount"`
	// +openapi:description=项目总数
	NamespaceCount int64 `json:"namespaceCount"`
	// +openapi:description=用户总数
	UserCount int64 `json:"userCount"`
	// +openapi:description=平台角色总数
	RoleCount int64 `json:"roleCount"`
}

// WorkspaceOverview represents workspace-level statistics.
// +openapi:description=租户级概览统计
type WorkspaceOverview struct {
	runtime.TypeMeta `json:",inline"`
	Spec             WorkspaceOverviewSpec `json:"spec"`
}

func (o *WorkspaceOverview) GetTypeMeta() *runtime.TypeMeta { return &o.TypeMeta }

type WorkspaceOverviewSpec struct {
	// +openapi:description=项目数量
	NamespaceCount int64 `json:"namespaceCount"`
	// +openapi:description=成员数量
	MemberCount int64 `json:"memberCount"`
	// +openapi:description=角色数量
	RoleCount int64 `json:"roleCount"`
}

// NamespaceOverview represents namespace-level statistics.
// +openapi:description=项目级概览统计
type NamespaceOverview struct {
	runtime.TypeMeta `json:",inline"`
	Spec             NamespaceOverviewSpec `json:"spec"`
}

func (o *NamespaceOverview) GetTypeMeta() *runtime.TypeMeta { return &o.TypeMeta }

type NamespaceOverviewSpec struct {
	// +openapi:description=成员数量
	MemberCount int64 `json:"memberCount"`
	// +openapi:description=角色数量
	RoleCount int64 `json:"roleCount"`
}
