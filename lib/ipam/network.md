# LCP 网络管理模块设计

## 概述

基于 `lib/ipam` 构建 LCP 平台网络管理功能，采用 **Network → Subnet → Allocation** 三层资源模型，通过 REST API 暴露，支持自动和手动 IP 分配。

## 资源模型

```
Network (私有网络/VPC，Workspace 级别)
  └── Subnet (子网，对应一个 CIDR 段，含 bitmap)
        └── Allocation (IP 分配记录，关联到 Host)
```

**与 lib/ipam 的映射关系：**

| 业务概念 | lib/ipam 对应 | 说明 |
|---------|-------------|------|
| Network | 无直接对应 | 纯 DB 记录，逻辑分组容器 |
| Subnet | `ipam.Pool` | 每次请求从 DB bitmap 临时构建，用完丢弃 |
| Subnet 的 CIDR | Pool 中的单个 CIDR | 一个 Subnet = 一个 CIDR |
| Subnet 的 bitmap | `Pool` 内部 bitmap | 持久化在 subnets 表的 `bitmap BYTEA` 字段 |
| 网关 IP | 特殊 Allocation | host_id = NULL，标记为 gateway |
| IP 分配 | `ip_allocations` 记录 | host_id 外键关联 hosts 表 |

**与现有 hosts/environments 表的关系：**

```
Workspace
├── Network (Workspace 级，逻辑分组容器)
│     └── Subnet (持有 CIDR + bitmap)
│           └── ip_allocations ←── host_id ──→ Host
├── Namespace
│     └── Host ←── environment_id ──→ Environment
```

ip_allocations 本质是 **hosts 与 subnets 之间的多对多关联表**：一台主机可以在多个子网中各持有 IP，一个子网可以分配给多台主机。

注意 scope 差异：Network 是 Workspace 级资源，Host 是多 scope（platform/workspace/namespace）资源。平台级或 Namespace 级 Host 也可从 Workspace 级 Network 中分配 IP。

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
2. 创建 ipam.Pool，根据 CIDR 生成初始 bitmap
3. 如果指定了 gateway，在 Pool 中标记 gateway IP 为已占用
4. 序列化 bitmap 为 `[]byte`
5. BEGIN 事务
6. 插入 `subnets` 表记录（含 bitmap 字段）
7. 如果指定了 gateway，插入 `ip_allocations` 记录（host_id = NULL，标记为 gateway）
8. COMMIT
9. 返回 Subnet 对象（含 usedIPs/freeIPs/totalIPs 统计）

**lib/ipam 交互：**
- `Pool.CreateFromCIDR()` — 创建 IP 池并生成 bitmap
- `Pool.Allocate()` — 预留网关 IP
- `Pool.SaveToBytes()` — 序列化 bitmap

---

### 流程 3：用户为主机分配 IP（自动）

```
用户操作: POST /api/network/v1/workspaces/{wsId}/networks/{netId}/subnets/{subId}/allocations
请求体:   { "spec": {"hostId": "123", "description": "Web 服务器"} }
```

**步骤：**
1. 验证 subnet 存在、hostId 有效
2. BEGIN 事务
3. `SELECT bitmap FROM subnets WHERE id = ? FOR UPDATE` — 行锁
4. 反序列化 bitmap → 构建临时 ipam.Pool
5. 调用 `Pool.AllocateNext()` 获取下一个可用 IP
6. 序列化更新后的 bitmap
7. `UPDATE subnets SET bitmap = ? WHERE id = ?`
8. 插入 `ip_allocations` 表记录（含 host_id 外键）
9. COMMIT
10. 返回 Allocation 对象（含 IP、CIDR、hostId）

**lib/ipam 交互：**
- `Pool.LoadFromBytes()` — 从 DB bitmap 恢复 Pool 状态
- `Pool.AllocateNext()` — 自动分配下一个可用 IP
- `Pool.SaveToBytes()` — 序列化 bitmap 写回 DB

---

### 流程 4：用户为主机分配 IP（手动指定）

```
用户操作: POST /api/network/v1/.../subnets/{subId}/allocations
请求体:   { "spec": {"ip": "10.0.1.100", "hostId": "456"} }
```

**步骤：**
1. 验证 subnet 存在、IP 在 CIDR 范围内、hostId 有效
2. BEGIN 事务
3. `SELECT bitmap FROM subnets WHERE id = ? FOR UPDATE` — 行锁
4. 反序列化 bitmap → 构建临时 ipam.Pool
5. 调用 `Pool.Allocate(ip)` 标记指定 IP
6. 序列化更新后的 bitmap
7. `UPDATE subnets SET bitmap = ? WHERE id = ?`
8. 插入 `ip_allocations` 表记录
9. COMMIT
10. 返回 Allocation 对象

**lib/ipam 交互：**
- `Pool.LoadFromBytes()` → `Pool.Allocate()` → `Pool.SaveToBytes()`

---

### 流程 5：释放 IP

```
用户操作: DELETE /api/network/v1/.../subnets/{subId}/allocations/{allocId}
```

**步骤：**
1. BEGIN 事务
2. 从 DB 查找 allocation 记录获取 IP
3. `SELECT bitmap FROM subnets WHERE id = ? FOR UPDATE` — 行锁
4. 反序列化 bitmap → `Pool.Release(ip)` → 序列化写回
5. `UPDATE subnets SET bitmap = ? WHERE id = ?`
6. 删除 `ip_allocations` 表记录
7. COMMIT

**lib/ipam 交互：**
- `Pool.LoadFromBytes()` → `Pool.Release()` → `Pool.SaveToBytes()`

---

### 流程 6：删除子网

```
用户操作: DELETE /api/network/v1/.../networks/{netId}/subnets/{subId}
```

**步骤：**
1. 查询子网下是否有非 gateway 的 allocation
2. 有 → 返回 409 Conflict（"subnet has allocated IPs"）
3. 无 → BEGIN 事务
4. 删除 gateway 的 `ip_allocations` 记录
5. 删除 `subnets` 表记录（bitmap 随之删除）
6. COMMIT

**lib/ipam 交互：** 无（直接删除 DB 记录即可，无内存状态需要清理）

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
                        └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │  DB Store       │  PostgreSQL
                        │  (PostgreSQL)   │
                        │                 │
                        │  networks       │
                        │  subnets        │  ← bitmap BYTEA 字段
                        │  ip_allocations │  ← host_id 外键 → hosts
                        └────────┬────────┘
                                 │ bitmap 读写
                        ┌────────▼────────┐
                        │  lib/ipam Pool  │  临时构建，用完丢弃
                        │  (bitmap 运算)  │
                        │                 │
                        │  LoadFromBytes  │  DB → Pool
                        │  SaveToBytes    │  Pool → DB
                        └─────────────────┘
```

**关键设计决策：bitmap 存 DB，服务无状态**

为支持横向扩容，bitmap 持久化在 subnets 表中，不再常驻内存：
- 每次 IP 操作：`SELECT bitmap FOR UPDATE` → 反序列化 → ipam 运算 → 序列化写回
- 并发控制通过 DB 行锁（`FOR UPDATE`）实现，多实例安全
- lib/ipam 的 Pool/bitmap 逻辑完全复用，只是不再作为进程级单例
- 无需服务启动恢复流程（无内存状态）

**事务边界：**

```
BEGIN
  SELECT bitmap FROM subnets WHERE id = ? FOR UPDATE   -- 行锁
  → lib/ipam Pool 反序列化 + 操作 + 序列化
  UPDATE subnets SET bitmap = ? WHERE id = ?
  INSERT/DELETE ip_allocations (...)
COMMIT
```

bitmap 更新和 ip_allocations 记录在同一事务中，保证一致性。

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
    bitmap       BYTEA        NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(network_id, name),
    UNIQUE(network_id, cidr)
);
CREATE INDEX idx_subnets_network_id ON subnets(network_id);

-- IP 分配记录（hosts 与 subnets 的多对多关联表）
CREATE TABLE ip_allocations (
    id           BIGSERIAL    PRIMARY KEY,
    subnet_id    BIGINT       NOT NULL REFERENCES subnets(id),
    ip           VARCHAR(50)  NOT NULL,
    host_id      BIGINT       REFERENCES hosts(id),
    is_gateway   BOOLEAN      NOT NULL DEFAULT false,
    description  TEXT         NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(subnet_id, ip)
);
CREATE INDEX idx_ip_allocations_subnet_id ON ip_allocations(subnet_id);
CREATE INDEX idx_ip_allocations_host_id ON ip_allocations(host_id) WHERE host_id IS NOT NULL;
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
    IP          string `json:"ip,omitempty"`          // 手动指定时填写，自动分配时留空
    HostID      string `json:"hostId,omitempty"`      // 关联主机，gateway 时为空
    IsGateway   bool   `json:"isGateway,omitempty"`   // 只读
    CIDR        string `json:"cidr,omitempty"`        // 只读
    Description string `json:"description,omitempty"`
}
```

## lib/ipam 需要新增

```go
// LoadFromBytes 从 DB 中的 []byte 恢复 Pool 的 bitmap 状态
func (p *Pool) LoadFromBytes(bitmap []byte) error

// SaveToBytes 将 Pool 的 bitmap 序列化为 []byte，用于写回 DB
func (p *Pool) SaveToBytes() []byte
```

不再需要 `RestorePool` / `RestoreAllocation`，因为 bitmap 存 DB 后无启动恢复流程。

## 模块文件结构

```
pkg/apis/network/
  types.go                         — API 类型 (Network, Subnet, IPAllocation)
  store.go                         — Store 接口 (NetworkStore, SubnetStore, IPAllocationStore)
  storage.go                       — REST 存储层 (networkStorage, subnetStorage, ipAllocationStorage)
  validation.go                    — 校验函数
  provider.go                      — Stores 聚合 + RESTStorageProvider
  v1/
    install.go                     — 路由注册 + 模块初始化

pkg/apis/network/store/
  pg_network.go                    — PostgreSQL Network 存储实现
  pg_subnet.go                     — PostgreSQL Subnet 存储实现（含 bitmap 读写）
  pg_ip_allocation.go              — PostgreSQL IPAllocation 存储实现
  stores.go                        — 工厂函数

pkg/db/query/
  network.sql                      — sqlc 查询
  subnet.sql                       — 含 SELECT bitmap FOR UPDATE 查询
  ip_allocation.sql

pkg/db/schema/schema.sql           — 追加建表语句
pkg/apis/install.go                — 注册 network 模块
lib/ipam/pool.go                   — 新增 LoadFromBytes / SaveToBytes
```

## 实现阶段

| Phase | 内容 | 验证 |
|-------|------|------|
| 1 | lib/ipam: 新增 Pool.LoadFromBytes/SaveToBytes + 测试 | `go test ./lib/ipam/...` |
| 2 | DB schema + sqlc 查询 + `make sqlc-generate` | 检查生成代码 |
| 3 | types.go + store.go + provider.go + validation.go | 编译通过 |
| 4 | store/pg_*.go（PostgreSQL 实现，含 bitmap 事务操作） | 编译通过 |
| 5 | storage.go（REST 层，含 ipam bitmap 交互） | 编译通过 |
| 6 | v1/install.go + pkg/apis/install.go（装配） | `make vet && make test` |
| 7 | 端到端验证 | curl 测试完整流程 |
