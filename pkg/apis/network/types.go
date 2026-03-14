package network

import (
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db/generated"
)

// --- Network types ---

// Network
// +openapi:description=网络管理：平台级 VPC 逻辑分组容器，用于组织和隔离子网资源。
type Network struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             NetworkSpec `json:"spec"`
}

func (n *Network) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// NetworkSpec
// +openapi:description=网络属性：包含网络描述、状态和子网统计信息。
type NetworkSpec struct {
	// +openapi:description=网络显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=网络描述
	Description string `json:"description,omitempty"`
	// +openapi:description=网络 CIDR 地址段（可选），限制子网 CIDR 分配范围
	CIDR string `json:"cidr,omitempty"`
	// +openapi:description=子网数量上限（1-50，默认 10）
	MaxSubnets int32 `json:"maxSubnets,omitempty"`
	// +openapi:description=是否公开网络（true=平台公开，false=租户私有）
	IsPublic *bool `json:"isPublic,omitempty"`
	// +openapi:description=网络状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
	// +openapi:description=子网数量（只读）
	SubnetCount int64 `json:"subnetCount,omitempty"`
}

// NetworkList
// +openapi:description=网络列表：分页返回的网络集合。
type NetworkList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Network `json:"items"`
	TotalCount       int64     `json:"totalCount"`
}

func (nl *NetworkList) GetTypeMeta() *runtime.TypeMeta { return &nl.TypeMeta }

// --- Subnet types ---

// Subnet
// +openapi:description=子网管理：网络下的 CIDR 子网，支持 IP 地址分配和位图跟踪。
type Subnet struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             SubnetSpec `json:"spec"`
}

func (s *Subnet) GetTypeMeta() *runtime.TypeMeta { return &s.TypeMeta }

// SubnetSpec
// +openapi:description=子网属性：包含 CIDR、网关、IP 使用统计和状态。
type SubnetSpec struct {
	// +openapi:description=子网显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=子网描述
	Description string `json:"description,omitempty"`
	// +openapi:required
	// +openapi:description=CIDR 地址段，如 10.0.0.0/24
	CIDR string `json:"cidr"`
	// +openapi:description=网关 IP 地址
	Gateway string `json:"gateway,omitempty"`
	// +openapi:description=所属网络 ID（只读）
	NetworkID string `json:"networkId,omitempty"`
	// +openapi:description=可用 IP 数量（只读）
	FreeIPs int `json:"freeIPs,omitempty"`
	// +openapi:description=已用 IP 数量（只读）
	UsedIPs int `json:"usedIPs,omitempty"`
	// +openapi:description=总可用 IP 数量（只读）
	TotalIPs int `json:"totalIPs,omitempty"`
	// +openapi:description=最小可分配 IP 地址（只读，无可用时为空）
	NextFreeIP string `json:"nextFreeIP,omitempty"`
}

// SubnetList
// +openapi:description=子网列表：分页返回的子网集合。
type SubnetList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Subnet `json:"items"`
	TotalCount       int64    `json:"totalCount"`
}

func (sl *SubnetList) GetTypeMeta() *runtime.TypeMeta { return &sl.TypeMeta }

// --- IPAllocation types ---

// IPAllocation
// +openapi:description=IP 分配记录：子网中已分配的 IP 地址信息。
type IPAllocation struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             IPAllocationSpec `json:"spec"`
}

func (a *IPAllocation) GetTypeMeta() *runtime.TypeMeta { return &a.TypeMeta }

// IPAllocationSpec
// +openapi:description=IP 分配属性：包含 IP 地址、描述和网关标识。
type IPAllocationSpec struct {
	// +openapi:required
	// +openapi:description=IP 地址
	IP string `json:"ip"`
	// +openapi:description=分配描述
	Description string `json:"description,omitempty"`
	// +openapi:description=是否为网关地址（只读）
	IsGateway bool `json:"isGateway,omitempty"`
	// +openapi:description=所属子网 ID（只读）
	SubnetID string `json:"subnetId,omitempty"`
	// +openapi:description=关联主机 ID（只读）
	HostID string `json:"hostId,omitempty"`
	// +openapi:description=关联主机名称（只读）
	HostName string `json:"hostName,omitempty"`
}

// IPAllocationList
// +openapi:description=IP 分配列表：分页返回的 IP 分配记录集合。
type IPAllocationList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []IPAllocation `json:"items"`
	TotalCount       int64          `json:"totalCount"`
}

func (al *IPAllocationList) GetTypeMeta() *runtime.TypeMeta { return &al.TypeMeta }

// --- DB type aliases ---

// DBNetwork is an alias for the sqlc-generated Network model.
type DBNetwork = generated.Network

// DBNetworkWithCount extends Network with subnet_count from GetNetworkByID.
type DBNetworkWithCount = generated.GetNetworkByIDRow

// DBNetworkListRow is an alias for ListNetworks row.
type DBNetworkListRow = generated.ListNetworksRow

// DBSubnet is an alias for the sqlc-generated Subnet model.
type DBSubnet = generated.Subnet

// DBSubnetCIDR is an alias for ListSubnetCIDRsByNetworkID row.
type DBSubnetCIDR = generated.ListSubnetCIDRsByNetworkIDRow

// DBIPAllocation is an alias for the sqlc-generated IpAllocation model.
type DBIPAllocation = generated.IpAllocation

// DBIPAllocationListRow is an alias for ListIPAllocations row (includes host info).
type DBIPAllocationListRow = generated.ListIPAllocationsRow
