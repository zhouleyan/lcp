# LCP 网络管理模块设计

## 概述

基于 `lib/ipam` 构建 LCP 平台网络管理功能，采用 **Network → Subnet → Allocation** 三层资源模型，通过 REST API 暴露，支持自动和手动 IP 分配。

## 资源模型

```
Network (私有网络/VPC，Workspace 级别)
  └── Subnet (子网，对应一个 CIDR 段)
        └── Allocation (IP 分配记录，关联到主机/资源)
```

**与 lib/ipam 的映射关系：**

| 业务概念 | lib/ipam 对应 | 说明 |
|---------|-------------|------|
| Network | 无直接对应 | 纯 DB 记录，逻辑分组容器 |
| Subnet | `ipam.Manager` 中的一个 Pool | Pool name = `"subnet:<subnetID>"` |
| Subnet 的 CIDR | Pool 中的单个 CIDR | 一个 Subnet = 一个 CIDR |
| 网关 IP | 特殊 Allocation | Owner = `"system:gateway"` |
| IP 分配 | `ipam.Allocation` | Owner = 主机/资源的 ID |

## 业务流程

### 流程 1：管理员创建网络

```
用户操作: POST /api/network/v1/workspaces/{workspaceId}/networks
请求体:   { "metadata": {"name": "vpc-prod"}, "spec": {"description": "生产网络"} }
```

**步骤：**
1. 验证 workspaceId 存在、name 合法
2. 插入 `networks` 表记录
3. 返回 Network 对象

**lib/ipam 交互：** 无。Network 是逻辑容器，不持有 CIDR。

---

### 流程 2：管理员在网络下创建子网

```
用户操作: POST /api/network/v1/workspaces/{wsId}/networks/{netId}/subnets
请求体:   { "metadata": {"name": "web-tier"}, "spec": {"cidr": "10.0.1.0/24", "gateway": "10.0.1.1"} }
```

**步骤：**
1. 验证 network 存在、CIDR 格式合法、CIDR 不与同网络下其他子网重叠
2. 插入 `subnets` 表记录（得到 subnetID）
3. 调用 `ipamManager.CreatePool("subnet:<subnetID>", []string{"10.0.1.0/24"})`
4. 如果指定了 gateway，调用 `ipamManager.Allocate("subnet:<subnetID>", gatewayIP, "system:gateway")` 预留网关
5. 返回 Subnet 对象（含 usedIPs/freeIPs/totalIPs 统计）

**lib/ipam 交互：**
- `Manager.CreatePool()` — 创建 IP 池
- `Manager.Allocate()` — 预留网关 IP

---

### 流程 3：用户为主机分配 IP（自动）

```
用户操作: POST /api/network/v1/workspaces/{wsId}/networks/{netId}/subnets/{subId}/allocations
请求体:   { "spec": {"owner": "host-web-01", "description": "Web 服务器"} }
```

**步骤：**
1. 验证 subnet 存在
2. 调用 `ipamManager.AllocateNext("subnet:<subnetID>", "host-web-01")`
3. 插入 `ip_allocations` 表记录
4. 返回 Allocation 对象（含 IP、CIDR、Owner）

**lib/ipam 交互：** `Manager.AllocateNext()` — 自动分配下一个可用 IP

---

### 流程 4：用户为主机分配 IP（手动指定）

```
用户操作: POST /api/network/v1/.../subnets/{subId}/allocations
请求体:   { "spec": {"ip": "10.0.1.100", "owner": "host-db-01"} }
```

**步骤：**
1. 验证 subnet 存在、IP 在 CIDR 范围内
2. 调用 `ipamManager.Allocate("subnet:<subnetID>", ip, "host-db-01")`
3. 插入 `ip_allocations` 表记录
4. 返回 Allocation 对象

**lib/ipam 交互：** `Manager.Allocate()` — 手动分配指定 IP

---

### 流程 5：释放 IP

```
用户操作: DELETE /api/network/v1/.../subnets/{subId}/allocations/{allocId}
```

**步骤：**
1. 从 DB 查找 allocation 记录获取 IP
2. 调用 `ipamManager.Release("subnet:<subnetID>", ip)`
3. 删除 `ip_allocations` 表记录

**lib/ipam 交互：** `Manager.Release()` — 释放 IP 回池

---

### 流程 6：删除子网

```
用户操作: DELETE /api/network/v1/.../networks/{netId}/subnets/{subId}
```

**步骤：**
1. 查询子网下是否有非 gateway 的 allocation
2. 有 → 返回 409 Conflict（"subnet has allocated IPs"）
3. 无 → 调用 `ipamManager.Release()` 释放 gateway IP
4. 调用 `ipamManager.DeletePool("subnet:<subnetID>")`
5. 删除 `subnets` 表记录

**lib/ipam 交互：**
- `Manager.Release()` — 释放网关
- `Manager.DeletePool()` — 删除 IP 池

---

### 流程 7：删除网络

```
用户操作: DELETE /api/network/v1/.../networks/{netId}
```

**步骤：**
1. 检查网络下是否有子网
2. 有 → 返回 409 Conflict（"network has subnets"）
3. 无 → 删除 `networks` 表记录

**lib/ipam 交互：** 无

---

### 流程 8：服务启动恢复

```
服务重启时，需要从 DB 恢复 ipam.Manager 的内存状态
```

**步骤：**
1. 查询所有 subnets 记录
2. 对每个 subnet 调用 `ipamManager.RestorePool("subnet:<id>", []string{cidr})`
3. 查询每个 subnet 的 allocations
4. 对每个 allocation 调用 `ipamManager.RestoreAllocation(pool, ip, owner)` 重放
5. 完成后 Manager 内存状态与 DB 一致

**需要新增：** `Manager.RestorePool()` 和 `Manager.RestoreAllocation()` 方法（跳过 Store 回写，仅恢复内存状态）

## API 路由

```
# Workspace 下的网络管理
/api/network/v1/workspaces/{workspaceId}/networks                              # CRUD
/api/network/v1/workspaces/{workspaceId}/networks/{networkId}/subnets          # CRUD
/api/network/v1/workspaces/{workspaceId}/networks/{networkId}/subnets/{subnetId}/allocations  # list + create/delete

# 平台级列表（管理员视角）
/api/network/v1/networks                                                        # list all
/api/network/v1/subnets                                                         # list all
```

## 数据交互图

```
                        ┌─────────────────┐
                        │   REST Client   │
                        └────────┬────────┘
                                 │ HTTP
                        ┌────────▼────────┐
                        │  REST Storage   │  pkg/apis/network/storage.go
                        │  (networkStorage│
                        │  subnetStorage  │
                        │  allocStorage)  │
                        └──┬──────────┬───┘
                           │          │
              ┌────────────▼──┐  ┌────▼──────────┐
              │  DB Store     │  │  ipam.Manager  │  lib/ipam/manager.go
              │  (PostgreSQL) │  │  (内存分配引擎) │
              │               │  │                │
              │  networks     │  │  Pool per      │
              │  subnets      │  │  subnet        │
              │  ip_allocs    │  │  Bitmap alloc  │
              └───────────────┘  └────────────────┘
```

**关键设计决策：Manager 使用 NoopStore**

REST 存储层同时操作 DB 和 ipam.Manager，由 REST 层掌控事务边界：
- 写 DB 成功 + ipam 操作成功 → 完成
- 写 DB 成功 + ipam 操作失败 → 回滚 DB
- 写 DB 失败 → 不操作 ipam

避免 ipam.Store 双写 DB 的问题。

## DB Schema

```sql
-- 网络（私有网络/VPC）
CREATE TABLE networks (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    workspace_id BIGINT       NOT NULL REFERENCES workspaces(id),
    owner_id     BIGINT       NOT NULL REFERENCES users(id),
    description  TEXT         NOT NULL DEFAULT '',
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(workspace_id, name)
);
CREATE INDEX idx_networks_workspace_id ON networks(workspace_id);

-- 子网
CREATE TABLE subnets (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    network_id   BIGINT       NOT NULL REFERENCES networks(id),
    cidr         VARCHAR(50)  NOT NULL,
    gateway      VARCHAR(50)  NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(network_id, name),
    UNIQUE(network_id, cidr)
);
CREATE INDEX idx_subnets_network_id ON subnets(network_id);

-- IP 分配记录
CREATE TABLE ip_allocations (
    id           BIGSERIAL    PRIMARY KEY,
    subnet_id    BIGINT       NOT NULL REFERENCES subnets(id),
    ip           VARCHAR(50)  NOT NULL,
    owner        VARCHAR(255) NOT NULL,
    description  TEXT         NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(subnet_id, ip)
);
CREATE INDEX idx_ip_allocations_subnet_id ON ip_allocations(subnet_id);
CREATE INDEX idx_ip_allocations_owner ON ip_allocations(owner);
```

## API 类型

```go
// Network 私有网络
type Network struct {
    runtime.TypeMeta `json:",inline"`
    types.ObjectMeta `json:"metadata"`
    Spec             NetworkSpec `json:"spec"`
}
type NetworkSpec struct {
    WorkspaceID string `json:"workspaceId"`
    OwnerID     string `json:"ownerId"`
    Description string `json:"description,omitempty"`
    Status      string `json:"status,omitempty"`
    SubnetCount int    `json:"subnetCount,omitempty"`  // 只读，列表时计算
}

// Subnet 子网
type Subnet struct {
    runtime.TypeMeta `json:",inline"`
    types.ObjectMeta `json:"metadata"`
    Spec             SubnetSpec `json:"spec"`
}
type SubnetSpec struct {
    NetworkID   string `json:"networkId"`
    CIDR        string `json:"cidr"`
    Gateway     string `json:"gateway,omitempty"`
    Description string `json:"description,omitempty"`
    Status      string `json:"status,omitempty"`
    UsedIPs     int    `json:"usedIPs,omitempty"`   // 只读，从 ipam 计算
    FreeIPs     int    `json:"freeIPs,omitempty"`   // 只读
    TotalIPs    int    `json:"totalIPs,omitempty"`  // 只读
}

// IPAllocation IP 分配记录
type IPAllocation struct {
    runtime.TypeMeta `json:",inline"`
    types.ObjectMeta `json:"metadata"`
    Spec             IPAllocationSpec `json:"spec"`
}
type IPAllocationSpec struct {
    SubnetID    string `json:"subnetId"`
    IP          string `json:"ip,omitempty"`      // 手动指定时填写，自动分配时留空
    Owner       string `json:"owner"`
    CIDR        string `json:"cidr,omitempty"`    // 只读
    Description string `json:"description,omitempty"`
}
```

## lib/ipam 需要新增

```go
// RestorePool 从持久化状态恢复池，不触发 Store 回写。用于服务启动恢复。
func (m *Manager) RestorePool(name string, cidrs []string) error

// RestoreAllocation 恢复单个分配记录，不触发 Store 回写。用于服务启动恢复。
func (m *Manager) RestoreAllocation(pool string, ip net.IP, owner string) error
```

## 模块文件结构

```
pkg/apis/network/
  types.go                         — API 类型 (Network, Subnet, IPAllocation)
  store.go                         — Store 接口 (NetworkStore, SubnetStore, IPAllocationStore)
  storage.go                       — REST 存储层 (networkStorage, subnetStorage, ipAllocationStorage)
  validation.go                    — 校验函数
  provider.go                      — Stores 聚合 + RESTStorageProvider
  v1/
    install.go                     — 路由注册 + 模块初始化（含 ipam.Manager 创建和恢复）

pkg/apis/network/store/
  pg_network.go                    — PostgreSQL Network 存储实现
  pg_subnet.go                     — PostgreSQL Subnet 存储实现
  pg_ip_allocation.go              — PostgreSQL IPAllocation 存储实现
  stores.go                        — 工厂函数

pkg/db/query/
  network.sql                      — sqlc 查询
  subnet.sql
  ip_allocation.sql

pkg/db/schema/schema.sql           — 追加建表语句
pkg/apis/install.go                — 注册 network 模块
lib/ipam/manager.go                — 新增 RestorePool / RestoreAllocation
```

## 实现阶段

| Phase | 内容 | 验证 |
|-------|------|------|
| 1 | lib/ipam: 新增 RestorePool/RestoreAllocation + 测试 | `go test ./lib/ipam/...` |
| 2 | DB schema + sqlc 查询 + `make sqlc-generate` | 检查生成代码 |
| 3 | types.go + store.go + provider.go + validation.go | 编译通过 |
| 4 | store/pg_*.go (PostgreSQL 实现) | 编译通过 |
| 5 | storage.go (REST 层，含 ipam 交互) | 编译通过 |
| 6 | v1/install.go + pkg/apis/install.go (装配) | `make vet && make test` |
| 7 | 端到端验证 | curl 测试完整流程 |
