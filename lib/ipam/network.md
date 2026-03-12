# LCP 网络管理模块 — 实现现状

> 最后更新: 2026-03-12

## 概述

基于 `lib/ipam` 构建的平台级网络管理模块，采用 **Network → Subnet → IPAllocation** 三层资源模型。通过 bitmap 实现高效 IP 地址分配与回收，bitmap 持久化在 DB 中，服务无状态，支持横向扩容。

**当前阶段：Phase 1（平台级网络地址簿管理）已完成。**

- Phase 1（已完成）：平台级 Network + Subnet + IPAllocation 纯地址簿管理
- Phase 2（未开始）：Workspace 分配、Host 关联（ip_allocations 加 host_id 外键）

## 资源模型

```
Network (平台级 VPC，逻辑分组容器)
  └── Subnet (子网，CIDR + bitmap)
        └── IPAllocation (IP 分配记录，IP + description + is_gateway)
```

| 业务概念 | lib/ipam 对应 | 说明 |
|---------|-------------|------|
| Network | 无直接对应 | 纯逻辑容器，可选 CIDR（限制子网分配范围） |
| Subnet | `ipam.Range` | 每次请求从 DB bitmap 临时构建，用完丢弃 |
| Subnet 的 bitmap | `Range` 内部 bitmap | 持久化在 subnets 表的 `bitmap BYTEA` 字段 |
| 网关 IP | 特殊 IPAllocation | is_gateway=true，创建子网时自动分配 |
| IP 分配 | `ip_allocations` 记录 | Phase 1 无 host_id，纯地址簿 |

## API 路由（已实现）

```
/api/network/v1/networks                                              # CRUD + batch delete
/api/network/v1/networks/{networkId}/subnets                          # CRUD + batch delete
/api/network/v1/networks/{networkId}/subnets/{subnetId}/allocations   # list + create + delete
```

所有资源为平台级，无 Workspace/Namespace 作用域。

## DB Schema（实际）

```sql
CREATE TABLE networks (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    cidr         VARCHAR(50)  NOT NULL DEFAULT '',      -- 可选，限制子网 CIDR 范围
    max_subnets  INT          NOT NULL DEFAULT 10,      -- 子网上限 (1-50)
    is_public    BOOLEAN      NOT NULL DEFAULT true,    -- 公开/私有
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE subnets (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    network_id   BIGINT       NOT NULL REFERENCES networks(id),
    cidr         VARCHAR(50)  NOT NULL,
    gateway      VARCHAR(50)  NOT NULL DEFAULT '',
    bitmap       BYTEA        NOT NULL DEFAULT '',      -- ipam bitmap 持久化
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(network_id, name),
    UNIQUE(network_id, cidr)
);

CREATE TABLE ip_allocations (
    id           BIGSERIAL    PRIMARY KEY,
    subnet_id    BIGINT       NOT NULL REFERENCES subnets(id),
    ip           VARCHAR(45)  NOT NULL,
    is_gateway   BOOLEAN      NOT NULL DEFAULT false,
    description  TEXT         NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(subnet_id, ip)
);
-- 注意：Phase 1 无 host_id 字段，Phase 2 会加 host_id BIGINT REFERENCES hosts(id)
```

## API 类型（实际）

```go
// NetworkSpec
type NetworkSpec struct {
    DisplayName string `json:"displayName,omitempty"`
    Description string `json:"description,omitempty"`
    CIDR        string `json:"cidr,omitempty"`          // 可选，限制子网 CIDR
    MaxSubnets  int32  `json:"maxSubnets,omitempty"`    // 1-50，默认 10
    IsPublic    *bool  `json:"isPublic,omitempty"`      // 公开/私有，默认 true
    Status      string `json:"status,omitempty"`        // active/inactive
    SubnetCount int64  `json:"subnetCount,omitempty"`   // 只读
}

// SubnetSpec
type SubnetSpec struct {
    DisplayName string `json:"displayName,omitempty"`
    Description string `json:"description,omitempty"`
    CIDR        string `json:"cidr"`                    // 必填
    Gateway     string `json:"gateway,omitempty"`       // 可选
    NetworkID   string `json:"networkId,omitempty"`     // 只读
    FreeIPs     int    `json:"freeIPs,omitempty"`       // 只读，bitmap 计算
    UsedIPs     int    `json:"usedIPs,omitempty"`       // 只读
    TotalIPs    int    `json:"totalIPs,omitempty"`      // 只读
    NextFreeIP  string `json:"nextFreeIP,omitempty"`    // 只读
}

// IPAllocationSpec
type IPAllocationSpec struct {
    IP          string `json:"ip"`                      // 必填
    Description string `json:"description,omitempty"`
    IsGateway   bool   `json:"isGateway,omitempty"`     // 只读
    SubnetID    string `json:"subnetId,omitempty"`      // 只读
}
```

## 核心业务流程

### 创建子网（含网关预分配）

```
POST /api/network/v1/networks/{networkId}/subnets
Body: { metadata: {name}, spec: {cidr, gateway?, displayName?, description?} }
```

1. 验证 name、CIDR 格式
2. 检查 network 的子网数量是否达到 maxSubnets 上限 → 409
3. 如果 network 有 CIDR，验证子网 CIDR 在 network CIDR 范围内
4. 查询同 network 下已有 CIDR，检查重叠 → 409
5. `ipam.NewCIDRRange(cidr)` 创建 Range
6. 如有 gateway → `range.Allocate(gatewayIP)` 预分配
7. `range.SaveToBytes()` 序列化 bitmap
8. BEGIN 事务：INSERT subnet + INSERT gateway allocation（is_gateway=true）
9. COMMIT，返回 Subnet 对象

### 分配 IP

```
POST /api/network/v1/networks/{networkId}/subnets/{subnetId}/allocations
Body: { spec: {ip, description?} }
```

```
BEGIN TX
  SELECT subnet FOR UPDATE                → 行锁 + 获取 bitmap
  ipam.NewCIDRRange(cidr) + LoadFromBytes → 恢复 bitmap
  Allocate(ip)                            → 标记分配（已分配返回 409）
  SaveToBytes → UPDATE bitmap             → 写回
  INSERT ip_allocation                    → 记录
COMMIT
```

### 释放 IP

```
DELETE /api/network/v1/networks/{networkId}/subnets/{subnetId}/allocations/{allocationId}
```

- Gateway IP 不可直接释放 → 400
- 同样的锁定路径：行锁 → 恢复 → Release → 写回 → 删除记录
- 如释放的是 gateway IP（通过删除子网触发），同时清空 subnet 的 gateway 字段

### 删除保护

| 操作 | 保护条件 | 错误 |
|------|---------|------|
| 删除 Network | 有子网时 | 409 Conflict |
| 删除 Subnet | 有非 gateway 的 allocation 时 | 409 Conflict |
| 删除 Gateway IP | 直接删除 | 400 Bad Request |

## lib/ipam 交互方式

```go
// 序列化：bitmap → DB
func (r *Range) SaveToBytes() []byte

// 反序列化：DB → bitmap
func (r *Range) LoadFromBytes(data []byte) error
```

基于 `AllocationBitmap.Snapshot()` 和 `Restore()` 实现。每次 IP 操作都是：

```
DB 读 bitmap → NewCIDRRange + LoadFromBytes → 操作 → SaveToBytes → DB 写 bitmap
```

服务无状态，并发安全通过 DB 行锁（`SELECT ... FOR UPDATE`）保证。

## 数据流

```
REST Client
    │ HTTP
    ▼
REST Storage (pkg/apis/network/storage.go)
  networkStorage  — 标准 CRUD，删除保护
  subnetStorage   — CRUD + bitmap 交互，usage 排序
  allocationStorage — Create/Delete 事务，行锁 bitmap
    │
    ▼
DB Store (pkg/apis/network/store/pg_*.go)
  networks / subnets / ip_allocations 表
    │ bitmap 读写
    ▼
lib/ipam Range (临时构建，用完丢弃)
  LoadFromBytes → 操作 → SaveToBytes
```

## 验证规则

| 资源 | 字段 | 规则 |
|------|------|------|
| Network | name | 必填，`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$` |
| Network | cidr | 可选，合法 CIDR 格式 |
| Network | maxSubnets | 1-50 |
| Network | description | 最长 1024 字符 |
| Subnet | name | 同 Network |
| Subnet | cidr | 必填，合法格式，在 network CIDR 范围内，不与已有子网重叠 |
| Subnet | gateway | 可选，合法 IP，在 CIDR 范围内 |
| IPAllocation | ip | 必填，合法 IPv4 格式 |
| IPAllocation | description | 最长 512 字符 |

## 前端实现

### 页面

| 页面 | 路由 | 功能 |
|------|------|------|
| 网络列表 | `/network/networks` | 搜索、排序（name/displayName/cidr/subnetCount/createdAt/updatedAt）、分页、创建/编辑/删除、批量删除 |
| 网络详情 | `/network/networks/:networkId` | 基本信息卡片、子网列表（嵌入完整表格）、编辑/删除 |
| 子网详情 | `/network/networks/:networkId/subnets/:subnetId` | 基本信息卡片、IP 分配列表、分配/释放 IP |

### UI 特性

- **子网数量进度条**：网络列表和详情页显示 `used/maxSubnets`，使用 `bg-primary` 三档透明度（20%/50%/100%）区分使用率
- **IP 使用进度条**：子网列表和详情页显示 `usedIPs/totalIPs`，同样的颜色规则
- **CIDR 可用范围**：网络/子网列表和详情页在 CIDR 下方显示可用 IP 范围（如 `10.0.0.1 - 10.0.0.254`）
- **公开/私有 Badge**：网络列表和详情页显示 isPublic 状态
- **网关自动填充**：创建子网时输入 CIDR 后自动推算并填充 gateway
- **CIDR 范围校验**：前端创建子网时校验 CIDR 是否在 network CIDR 范围内
- **保留 IP 提示**：分配 IP 时检测网络地址和广播地址并提示

### 文件清单

```
ui/src/api/network/
  client.ts              — API 客户端（前缀 /api/network/v1）
  networks.ts            — Network CRUD API
  subnets.ts             — Subnet CRUD API
  allocations.ts         — IPAllocation API

ui/src/pages/network/
  routes.tsx             — 路由定义
  networks/
    list.tsx             — 网络列表页（含创建/编辑表单）
    detail.tsx           — 网络详情页（含子网列表、子网表单）
    subnet-detail.tsx    — 子网详情页（含 IP 分配列表）
    utils.ts             — CIDR 可用范围计算

ui/src/i18n/locales/{en-US,zh-CN}/
  network.ts             — 网络模块 i18n（~100 key）
```

### 权限码

```
network:networks:list / get / create / update / patch / delete / deleteCollection
network:subnets:list / get / create / update / patch / delete / deleteCollection
network:allocations:list / get / create / delete
```

## 后端文件清单

```
lib/ipam/
  range.go               — SaveToBytes / LoadFromBytes

pkg/db/schema/schema.sql — networks / subnets / ip_allocations 表
pkg/db/query/
  network.sql            — Network CRUD + 分页列表
  subnet.sql             — Subnet CRUD + FOR UPDATE + bitmap 更新 + CIDR 列表
  ip_allocation.sql      — IPAllocation 创建/删除/列表

pkg/apis/network/
  types.go               — API 类型 + DB 类型别名
  store.go               — Store 接口（NetworkStore / SubnetStore / IPAllocationStore）
  storage.go             — REST Storage（含 ipam bitmap 交互逻辑）
  validation.go          — 校验函数
  provider.go            — Stores 聚合
  v1/install.go          — 路由注册
  store/
    pg_network.go        — PostgreSQL Network Store
    pg_subnet.go         — PostgreSQL Subnet Store（含事务、行锁）
    pg_ip_allocation.go  — PostgreSQL IPAllocation Store
    stores.go            — 工厂函数
    helpers.go           — filterStr/filterInt64 辅助

pkg/apis/install.go      — 注册 network 模块到全局
```

## Phase 2 待实现

| 功能 | 说明 |
|------|------|
| Workspace 级网络 | Network 加 workspace_id FK，API 路由加 workspace 前缀 |
| Host 关联 | ip_allocations 加 host_id FK → hosts 表 |
| 自动分配 | 支持不指定 IP，调用 `AllocateNext()` 自动分配 |
| 前端 Host-IP 绑定 | Host 详情页显示关联的 IP，支持从 Host 页面分配/释放 |
| 网络分配到 Workspace | 平台管理员将公开网络分配给特定 Workspace |
