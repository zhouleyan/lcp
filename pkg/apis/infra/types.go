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
	// +openapi:description=IP 配置列表（创建时可选，自动或手动分配子网 IP）
	IPs []IPConfig `json:"ips,omitempty"`
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
	// +openapi:description=所属租户名称（只读）
	WorkspaceName string `json:"workspaceName,omitempty"`
	// +openapi:description=所属项目 ID（namespace scope 时必填）
	NamespaceID string `json:"namespaceId,omitempty"`
	// +openapi:description=所属项目名称（只读）
	NamespaceName string `json:"namespaceName,omitempty"`
	// +openapi:description=绑定的环境 ID（只读）
	EnvironmentID string `json:"environmentId,omitempty"`
	// +openapi:description=绑定的环境名称（只读）
	EnvironmentName string `json:"environmentName,omitempty"`
	// +openapi:description=已分配的 IP 列表（只读）
	AllocatedIPs []AllocatedIP `json:"allocatedIPs,omitempty"`
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

// --- Action request types ---

// BindEnvironmentRequest is the request body for host bind-environment action.
type BindEnvironmentRequest struct {
	runtime.TypeMeta `json:",inline"`
	EnvironmentID    string `json:"environmentId"`
}

func (ber *BindEnvironmentRequest) GetTypeMeta() *runtime.TypeMeta { return &ber.TypeMeta }

// --- Region types ---

// Region
// +openapi:description=区域管理：可用域/地理区域，CMDB 顶层位置资源。
type Region struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             RegionSpec `json:"spec"`
}

func (r *Region) GetTypeMeta() *runtime.TypeMeta { return &r.TypeMeta }

// RegionSpec
// +openapi:description=区域属性：包含显示名称、状态、经纬度等信息。
type RegionSpec struct {
	// +openapi:description=显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=描述
	Description string `json:"description,omitempty"`
	// +openapi:description=状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
	// +openapi:description=纬度
	Latitude *float64 `json:"latitude,omitempty"`
	// +openapi:description=经度
	Longitude *float64 `json:"longitude,omitempty"`
	// +openapi:description=下属数据中心数量（只读）
	SiteCount int64 `json:"siteCount,omitempty"`
}

// RegionList
// +openapi:description=区域列表：分页返回的区域集合。
type RegionList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Region `json:"items"`
	TotalCount       int64    `json:"totalCount"`
}

func (r *RegionList) GetTypeMeta() *runtime.TypeMeta { return &r.TypeMeta }

// --- Site types ---

// Site
// +openapi:description=数据中心管理：数据中心/物理站点，属于某个区域。
type Site struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             SiteSpec `json:"spec"`
}

func (s *Site) GetTypeMeta() *runtime.TypeMeta { return &s.TypeMeta }

// SiteSpec
// +openapi:description=数据中心属性：包含区域关联、地址、联系人、经纬度等信息。
type SiteSpec struct {
	// +openapi:description=显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=描述
	Description string `json:"description,omitempty"`
	// +openapi:required
	// +openapi:description=所属区域 ID
	RegionID string `json:"regionId"`
	// +openapi:description=所属区域名称（只读）
	RegionName string `json:"regionName,omitempty"`
	// +openapi:description=状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
	// +openapi:description=物理地址
	Address string `json:"address,omitempty"`
	// +openapi:description=纬度
	Latitude *float64 `json:"latitude,omitempty"`
	// +openapi:description=经度
	Longitude *float64 `json:"longitude,omitempty"`
	// +openapi:description=负责人姓名
	ContactName string `json:"contactName,omitempty"`
	// +openapi:description=负责人电话
	ContactPhone string `json:"contactPhone,omitempty"`
	// +openapi:description=负责人邮箱
	// +openapi:format=email
	ContactEmail string `json:"contactEmail,omitempty"`
	// +openapi:description=下属机房数量（只读）
	LocationCount int64 `json:"locationCount,omitempty"`
}

// SiteList
// +openapi:description=数据中心列表：分页返回的数据中心集合。
type SiteList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Site `json:"items"`
	TotalCount       int64  `json:"totalCount"`
}

func (s *SiteList) GetTypeMeta() *runtime.TypeMeta { return &s.TypeMeta }

// --- Location types ---

// Location
// +openapi:description=机房管理：数据中心内的物理机房，属于某个数据中心。
type Location struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             LocationSpec `json:"spec"`
}

func (l *Location) GetTypeMeta() *runtime.TypeMeta { return &l.TypeMeta }

// LocationSpec
// +openapi:description=机房属性：包含数据中心关联、楼层、机柜容量、联系人等信息。
type LocationSpec struct {
	// +openapi:description=显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=描述
	Description string `json:"description,omitempty"`
	// +openapi:required
	// +openapi:description=所属数据中心 ID
	SiteID string `json:"siteId"`
	// +openapi:description=所属数据中心名称（只读）
	SiteName string `json:"siteName,omitempty"`
	// +openapi:description=所属区域 ID（只读，通过数据中心关联）
	RegionID string `json:"regionId,omitempty"`
	// +openapi:description=所属区域名称（只读）
	RegionName string `json:"regionName,omitempty"`
	// +openapi:description=状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
	// +openapi:description=楼层
	Floor string `json:"floor,omitempty"`
	// +openapi:description=机柜总容量
	RackCapacity int32 `json:"rackCapacity,omitempty"`
	// +openapi:description=下属机柜数量（只读）
	RackCount int64 `json:"rackCount,omitempty"`
	// +openapi:description=负责人姓名
	ContactName string `json:"contactName,omitempty"`
	// +openapi:description=负责人电话
	ContactPhone string `json:"contactPhone,omitempty"`
	// +openapi:description=负责人邮箱
	// +openapi:format=email
	ContactEmail string `json:"contactEmail,omitempty"`
}

// LocationList
// +openapi:description=机房列表：分页返回的机房集合。
type LocationList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Location `json:"items"`
	TotalCount       int64      `json:"totalCount"`
}

func (l *LocationList) GetTypeMeta() *runtime.TypeMeta { return &l.TypeMeta }

// --- Rack types ---

// Rack
// +openapi:description=机柜管理：数据中心机房内的物理机柜，属于某个机房。
type Rack struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             RackSpec `json:"spec"`
}

func (r *Rack) GetTypeMeta() *runtime.TypeMeta { return &r.TypeMeta }

// RackSpec
// +openapi:description=机柜属性：包含机房关联、U 高度、位置编号、供电容量等信息。
type RackSpec struct {
	// +openapi:description=显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=描述
	Description string `json:"description,omitempty"`
	// +openapi:required
	// +openapi:description=所属机房 ID
	LocationID string `json:"locationId"`
	// +openapi:description=所属机房名称（只读）
	LocationName string `json:"locationName,omitempty"`
	// +openapi:description=所属数据中心 ID（只读，通过机房关联）
	SiteID string `json:"siteId,omitempty"`
	// +openapi:description=所属数据中心名称（只读）
	SiteName string `json:"siteName,omitempty"`
	// +openapi:description=所属区域 ID（只读）
	RegionID string `json:"regionId,omitempty"`
	// +openapi:description=所属区域名称（只读）
	RegionName string `json:"regionName,omitempty"`
	// +openapi:description=状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
	// +openapi:description=机柜 U 高度（如 42）
	UHeight int32 `json:"uHeight,omitempty"`
	// +openapi:description=物理位置编号（如 A-01）
	Position string `json:"position,omitempty"`
	// +openapi:description=供电容量描述
	PowerCapacity string `json:"powerCapacity,omitempty"`
}

// RackList
// +openapi:description=机柜列表：分页返回的机柜集合。
type RackList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Rack `json:"items"`
	TotalCount       int64  `json:"totalCount"`
}

func (r *RackList) GetTypeMeta() *runtime.TypeMeta { return &r.TypeMeta }

// --- Network ACL types (read-only, for host IP allocation) ---

// AvailableNetwork
// +openapi:description=可用网络：主机 IP 分配时的网络选择视图，包含网络及其子网的摘要信息。
type AvailableNetwork struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             AvailableNetworkSpec `json:"spec"`
}

func (n *AvailableNetwork) GetTypeMeta() *runtime.TypeMeta { return &n.TypeMeta }

// AvailableNetworkSpec
// +openapi:description=可用网络属性：网络基本信息及下属子网摘要列表。
type AvailableNetworkSpec struct {
	// +openapi:description=网络显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=网络描述
	Description string `json:"description,omitempty"`
	// +openapi:description=网络 CIDR 地址段
	CIDR string `json:"cidr,omitempty"`
	// +openapi:description=是否公开网络
	IsPublic bool `json:"isPublic"`
	// +openapi:description=子网数量
	SubnetCount int64 `json:"subnetCount"`
	// +openapi:description=子网列表
	Subnets []SubnetSummary `json:"subnets"`
}

// SubnetSummary
// +openapi:description=子网摘要：子网基本信息和 IP 使用统计。
type SubnetSummary struct {
	// +openapi:description=子网 ID
	ID string `json:"id"`
	// +openapi:description=子网名称
	Name string `json:"name"`
	// +openapi:description=子网显示名称
	DisplayName string `json:"displayName,omitempty"`
	// +openapi:description=CIDR 地址段
	CIDR string `json:"cidr"`
	// +openapi:description=网关 IP 地址
	Gateway string `json:"gateway,omitempty"`
	// +openapi:description=可用 IP 数量
	FreeIPs int `json:"freeIPs"`
	// +openapi:description=已用 IP 数量
	UsedIPs int `json:"usedIPs"`
	// +openapi:description=总可用 IP 数量
	TotalIPs int `json:"totalIPs"`
}

// AvailableNetworkList
// +openapi:description=可用网络列表：主机 IP 分配时可选的网络集合。
type AvailableNetworkList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []AvailableNetwork `json:"items"`
	TotalCount       int64              `json:"totalCount"`
}

func (nl *AvailableNetworkList) GetTypeMeta() *runtime.TypeMeta { return &nl.TypeMeta }

// --- DB type aliases ---

// DBHost is an alias for the sqlc-generated Host model.
type DBHost = generated.Host

// DBEnvironment is an alias for the sqlc-generated Environment model.
type DBEnvironment = generated.Environment

// DBHostWithEnv extends Host with environment_name from GetHostByID.
type DBHostWithEnv = generated.GetHostByIDRow

// DBEnvWithCounts extends Environment with host_count from GetEnvironmentByID.
type DBEnvWithCounts = generated.GetEnvironmentByIDRow

// DBHostPlatformRow is an alias for ListHostsPlatform row (no origin field).
type DBHostPlatformRow = generated.ListHostsPlatformRow

// DBHostWorkspaceRow is an alias for ListHostsByWorkspaceID row.
type DBHostWorkspaceRow = generated.ListHostsByWorkspaceIDRow

// DBHostNamespaceRow is an alias for ListHostsByNamespaceID row.
type DBHostNamespaceRow = generated.ListHostsByNamespaceIDRow

// DBEnvPlatformRow is an alias for ListEnvironmentsPlatform row.
type DBEnvPlatformRow = generated.ListEnvironmentsPlatformRow

// DBEnvWorkspaceRow is an alias for ListEnvironmentsByWorkspaceID row.
type DBEnvWorkspaceRow = generated.ListEnvironmentsByWorkspaceIDRow

// DBEnvWorkspaceInheritRow is an alias for ListEnvironmentsByWorkspaceIDInherit row.
type DBEnvWorkspaceInheritRow = generated.ListEnvironmentsByWorkspaceIDInheritRow

// DBEnvNamespaceRow is an alias for ListEnvironmentsByNamespaceID row.
type DBEnvNamespaceRow = generated.ListEnvironmentsByNamespaceIDRow

// DBEnvNamespaceInheritRow is an alias for ListEnvironmentsByNamespaceIDInherit row.
type DBEnvNamespaceInheritRow = generated.ListEnvironmentsByNamespaceIDInheritRow

// DBHostByEnvRow is an alias for ListHostsByEnvironmentID row.
type DBHostByEnvRow = generated.ListHostsByEnvironmentIDRow

// --- Region DB type aliases ---

// DBRegion is an alias for the sqlc-generated Region model.
type DBRegion = generated.Region

// DBRegionWithCounts extends Region with site_count from GetRegionByID.
type DBRegionWithCounts = generated.GetRegionByIDRow

// DBRegionListRow is an alias for ListRegions row (with site_count).
type DBRegionListRow = generated.ListRegionsRow

// --- Site DB type aliases ---

// DBSite is an alias for the sqlc-generated Site model.
type DBSite = generated.Site

// DBSiteWithDetails extends Site with region_name and location_count from GetSiteByID.
type DBSiteWithDetails = generated.GetSiteByIDRow

// DBSiteListRow is an alias for ListSites row.
type DBSiteListRow = generated.ListSitesRow

// --- Location DB type aliases ---

// DBLocation is an alias for the sqlc-generated Location model.
type DBLocation = generated.Location

// DBLocationWithDetails extends Location with site_name, region_id, region_name from GetLocationByID.
type DBLocationWithDetails = generated.GetLocationByIDRow

// DBLocationListRow is an alias for ListLocations row.
type DBLocationListRow = generated.ListLocationsRow

// --- Rack DB type aliases ---

// DBRack is an alias for the sqlc-generated Rack model.
type DBRack = generated.Rack

// DBRackWithDetails extends Rack with location/site/region names from GetRackByID.
type DBRackWithDetails = generated.GetRackByIDRow

// DBRackListRow is an alias for ListRacks row.
type DBRackListRow = generated.ListRacksRow

// --- Network reader DB type aliases (ACL) ---

// DBNetworkACLRow is an alias for ListActiveNetworksWithSubnetCount row.
type DBNetworkACLRow = generated.ListActiveNetworksWithSubnetCountRow

// DBSubnet is an alias for the sqlc-generated Subnet model (for bitmap parsing).
type DBSubnet = generated.Subnet

// --- IP allocation types ---

// IPConfig represents an IP configuration in the host creation request.
// +openapi:description=IP 配置：创建主机时指定子网 ID 和可选 IP 地址。
type IPConfig struct {
	runtime.TypeMeta `json:",inline"`
	// +openapi:required
	// +openapi:description=子网 ID
	SubnetID string `json:"subnetId"`
	// +openapi:description=IP 地址（为空时自动分配）
	IP string `json:"ip,omitempty"`
}

func (c *IPConfig) GetTypeMeta() *runtime.TypeMeta { return &c.TypeMeta }

// AllocatedIP represents an IP address allocated to a host (read-only, returned in API responses).
// +openapi:description=已分配的 IP 信息
type AllocatedIP struct {
	// +openapi:description=分配记录 ID
	ID string `json:"id"`
	// +openapi:description=IP 地址
	IP string `json:"ip"`
	// +openapi:description=子网 ID
	SubnetID string `json:"subnetId"`
	// +openapi:description=子网名称（仅详情接口返回）
	SubnetName string `json:"subnetName,omitempty"`
	// +openapi:description=子网 CIDR（仅详情接口返回）
	SubnetCIDR string `json:"subnetCidr,omitempty"`
}

// AllocatedIPList
// +openapi:description=已分配 IP 列表
type AllocatedIPList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []AllocatedIP `json:"items"`
	TotalCount       int64         `json:"totalCount"`
}

func (l *AllocatedIPList) GetTypeMeta() *runtime.TypeMeta { return &l.TypeMeta }

// DBIPConfig is the internal representation of IPConfig with parsed IDs.
type DBIPConfig struct {
	SubnetID int64
	IP       string
}

// DBSubnetRow is an alias for GetSubnetByIDForUpdateACL result (same as Subnet model).
type DBSubnetRow = generated.Subnet

// DBIPAllocationWithHost is an alias for CreateIPAllocationWithHost result.
type DBIPAllocationWithHost = generated.CreateIPAllocationWithHostRow

// DBHostIPAllocationRow is an alias for ListIPAllocationsByHostID result.
type DBHostIPAllocationRow = generated.ListIPAllocationsByHostIDRow

// DBIPAllocationForHostRow is an alias for GetIPAllocationForHost result.
type DBIPAllocationForHostRow = generated.GetIPAllocationForHostRow

