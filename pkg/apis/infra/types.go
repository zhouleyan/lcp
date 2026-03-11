package infra

import (
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db/generated"
)

// Scope constants reused from iam, duplicated here for infra module independence.
const (
	ScopePlatform  = "platform"
	ScopeWorkspace = "workspace"
	ScopeNamespace = "namespace"
)

// --- Host types ---

// Host
// +openapi:description=主机管理：物理机或虚拟机资源，支持三层创建（平台/租户/项目）和向下分配。
type Host struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             HostSpec `json:"spec"`
}

func (h *Host) GetTypeMeta() *runtime.TypeMeta { return &h.TypeMeta }

// HostSpec
// +openapi:description=主机属性：包含主机名、IP、操作系统、硬件配置、标签、作用域和环境绑定信息。
type HostSpec struct {
	// +openapi:description=主机名称（hostname）
	Hostname string `json:"hostname,omitempty"`
	// +openapi:description=IP 地址
	IPAddress string `json:"ipAddress,omitempty"`
	// +openapi:description=操作系统
	OS string `json:"os,omitempty"`
	// +openapi:description=CPU 架构
	Arch string `json:"arch,omitempty"`
	// +openapi:description=CPU 核心数
	CPUCores int32 `json:"cpuCores,omitempty"`
	// +openapi:description=内存大小（MB）
	MemoryMB int64 `json:"memoryMb,omitempty"`
	// +openapi:description=磁盘大小（GB）
	DiskGB int64 `json:"diskGb,omitempty"`
	// +openapi:description=标签
	Labels map[string]string `json:"labels,omitempty"`
	// +openapi:required
	// +openapi:description=创建层级
	// +openapi:enum=platform,workspace,namespace
	Scope string `json:"scope"`
	// +openapi:description=所属租户 ID（workspace scope 时必填）
	WorkspaceID string `json:"workspaceId,omitempty"`
	// +openapi:description=所属项目 ID（namespace scope 时必填）
	NamespaceID string `json:"namespaceId,omitempty"`
	// +openapi:description=绑定的环境 ID（只读）
	EnvironmentID string `json:"environmentId,omitempty"`
	// +openapi:description=绑定的环境名称（只读）
	EnvironmentName string `json:"environmentName,omitempty"`
	// +openapi:description=主机来源：owned 表示自有，assigned 表示被分配（只读，仅 workspace/namespace 列表返回）
	// +openapi:enum=owned,assigned
	Origin string `json:"origin,omitempty"`
	// +openapi:description=主机状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
}

// HostList
// +openapi:description=主机列表：分页返回的主机集合。
type HostList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Host `json:"items"`
	TotalCount       int64  `json:"totalCount"`
}

func (h *HostList) GetTypeMeta() *runtime.TypeMeta { return &h.TypeMeta }

// --- Environment types ---

// Environment
// +openapi:description=环境管理：按生命周期阶段（开发、测试、生产等）分组资源的管理维度。
type Environment struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             EnvironmentSpec `json:"spec"`
}

func (e *Environment) GetTypeMeta() *runtime.TypeMeta { return &e.TypeMeta }

// EnvironmentSpec
// +openapi:description=环境属性：包含环境类型、作用域、主机数量和状态。
type EnvironmentSpec struct {
	// +openapi:description=环境显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=环境描述
	Description string `json:"description,omitempty"`
	// +openapi:description=环境类型
	// +openapi:enum=development,testing,staging,production,custom
	EnvType string `json:"envType,omitempty"`
	// +openapi:required
	// +openapi:description=创建层级
	// +openapi:enum=platform,workspace,namespace
	Scope string `json:"scope"`
	// +openapi:description=所属租户 ID（workspace scope 时必填）
	WorkspaceID string `json:"workspaceId,omitempty"`
	// +openapi:description=所属项目 ID（namespace scope 时必填）
	NamespaceID string `json:"namespaceId,omitempty"`
	// +openapi:description=关联主机数量（只读）
	HostCount int64 `json:"hostCount,omitempty"`
	// +openapi:description=环境状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
}

// EnvironmentList
// +openapi:description=环境列表：分页返回的环境集合。
type EnvironmentList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Environment `json:"items"`
	TotalCount       int64         `json:"totalCount"`
}

func (e *EnvironmentList) GetTypeMeta() *runtime.TypeMeta { return &e.TypeMeta }

// --- HostAssignment types ---

// HostAssignment
// +openapi:description=主机分配记录：表示上层主机被授权给下层使用。
type HostAssignment struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             HostAssignmentSpec `json:"spec"`
}

func (ha *HostAssignment) GetTypeMeta() *runtime.TypeMeta { return &ha.TypeMeta }

// HostAssignmentSpec
// +openapi:description=主机分配属性：包含主机 ID、目标租户或项目 ID。
type HostAssignmentSpec struct {
	// +openapi:required
	// +openapi:description=被分配的主机 ID
	HostID string `json:"hostId"`
	// +openapi:description=主机名称（只读）
	HostName string `json:"hostName,omitempty"`
	// +openapi:description=目标租户 ID
	WorkspaceID string `json:"workspaceId,omitempty"`
	// +openapi:description=目标租户名称（只读）
	WorkspaceName string `json:"workspaceName,omitempty"`
	// +openapi:description=目标项目 ID
	NamespaceID string `json:"namespaceId,omitempty"`
	// +openapi:description=目标项目名称（只读）
	NamespaceName string `json:"namespaceName,omitempty"`
}

// HostAssignmentList
// +openapi:description=主机分配列表。
type HostAssignmentList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []HostAssignment `json:"items"`
}

func (hal *HostAssignmentList) GetTypeMeta() *runtime.TypeMeta { return &hal.TypeMeta }

// --- Action request types ---

// AssignRequest is the request body for host assign/unassign actions.
type AssignRequest struct {
	runtime.TypeMeta `json:",inline"`
	WorkspaceID      string `json:"workspaceId,omitempty"`
	NamespaceID      string `json:"namespaceId,omitempty"`
}

func (ar *AssignRequest) GetTypeMeta() *runtime.TypeMeta { return &ar.TypeMeta }

// BindEnvironmentRequest is the request body for host bind-environment action.
type BindEnvironmentRequest struct {
	runtime.TypeMeta `json:",inline"`
	EnvironmentID    string `json:"environmentId"`
}

func (ber *BindEnvironmentRequest) GetTypeMeta() *runtime.TypeMeta { return &ber.TypeMeta }

// --- DB type aliases ---

// DBHost is an alias for the sqlc-generated Host model.
type DBHost = generated.Host

// DBEnvironment is an alias for the sqlc-generated Environment model.
type DBEnvironment = generated.Environment

// DBHostAssignment is an alias for the sqlc-generated HostAssignment model.
type DBHostAssignment = generated.HostAssignment

// DBHostWithEnv extends Host with environment_name from GetHostByID.
type DBHostWithEnv = generated.GetHostByIDRow

// DBEnvWithCounts extends Environment with host_count from GetEnvironmentByID.
type DBEnvWithCounts = generated.GetEnvironmentByIDRow

// DBHostPlatformRow is an alias for ListHostsPlatform row (no origin field).
type DBHostPlatformRow = generated.ListHostsPlatformRow

// DBHostWorkspaceRow is an alias for ListHostsByWorkspaceID row (with origin field).
type DBHostWorkspaceRow = generated.ListHostsByWorkspaceIDRow

// DBHostNamespaceRow is an alias for ListHostsByNamespaceID row (with origin field).
type DBHostNamespaceRow = generated.ListHostsByNamespaceIDRow

// DBEnvPlatformRow is an alias for ListEnvironmentsPlatform row.
type DBEnvPlatformRow = generated.ListEnvironmentsPlatformRow

// DBEnvWorkspaceRow is an alias for ListEnvironmentsByWorkspaceID row.
type DBEnvWorkspaceRow = generated.ListEnvironmentsByWorkspaceIDRow

// DBEnvNamespaceRow is an alias for ListEnvironmentsByNamespaceID row.
type DBEnvNamespaceRow = generated.ListEnvironmentsByNamespaceIDRow

// DBHostByEnvRow is an alias for ListHostsByEnvironmentID row.
type DBHostByEnvRow = generated.ListHostsByEnvironmentIDRow

// DBAssignmentRow is an alias for ListAssignmentsByHostID row.
type DBAssignmentRow = generated.ListAssignmentsByHostIDRow
