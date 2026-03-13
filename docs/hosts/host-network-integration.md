# 主机-网络集成设计

> 结论文档，基于需求讨论定稿

## 背景

网络模块 Phase 1（Network → Subnet → IPAllocation 纯地址簿管理）已完成。主机模块已有完整的多 scope CRUD、分配、环境绑定等功能。当前 Host 的 `ipAddress` 字段是独立的 VARCHAR(45) 自由文本，与网络模块完全脱节。

本文档定义主机生命周期中的 IP 绑定、解绑、释放功能的设计决策。

## 关键决策

| 决策项 | 结论 | 理由 |
|--------|------|------|
| 单 IP / 多 IP | **多 IP** | 物理服务器有多网卡（管理/业务/存储）；数据模型天然支持（多条 allocation 指向同一 host）；单 IP 后改多 IP 是 breaking change |
| IP 用途区分 | `ip_allocations` 新增 `purpose` 字段 | VARCHAR(50)，如 management / business / storage / backup，默认空串 |
| 删除主机时 IP 策略 | **删除保护** | 有绑定 IP 时禁止删除，返回 409；与现有模式一致（删 Network 要求无 Subnet，删 Subnet 要求无非 gateway Allocation） |
| Host.ipAddress 字段 | **废弃** | 改为从 ip_allocations 关联查询，不再作为 Host 的独立字段 |
| Workspace 网络作用域 | **暂不推进** | 仅考虑 public 网络，跳过 workspace 级网络分配 |
| 跨模块权限 | **后端代理查询** | infra 模块定义自己的 store 直接查网络表，注册为 infra 路由，不经过 network REST 层 |

## DB Schema 变更

```sql
-- ip_allocations 表新增字段
ALTER TABLE ip_allocations ADD COLUMN host_id BIGINT REFERENCES hosts(id) ON DELETE SET NULL;
ALTER TABLE ip_allocations ADD COLUMN purpose VARCHAR(50) NOT NULL DEFAULT '';

CREATE INDEX idx_ip_allocations_host_id ON ip_allocations(host_id);
```

- `ON DELETE SET NULL`：主机删除后 IP 保留为已分配但无主状态（实际流程中删除保护会阻止，此为兜底）
- `purpose`：IP 用途标识（management / business / storage / backup 等）

## 业务流程（共 9 条）

### 主机侧操作（7 条）

| # | 操作 | 说明 |
|---|------|------|
| ① | 查询可用子网 | 创建主机或绑定 IP 时，查看 public 网络下有空闲 IP 的子网列表 |
| ② | 绑定 IP 到主机 | 选定子网后，手动指定或自动分配 IP 绑定到主机（支持多 IP + purpose） |
| ③ | 解绑 IP | 从主机摘除 IP，IP 仍占用在子网中（预留状态，bitmap 不变） |
| ④ | 释放 IP | 从主机摘除 IP 并回收到子网池（bitmap 清零 + 删除 allocation 记录） |
| ⑤ | 查看主机 IP 列表 | 主机详情页展示所有绑定的 IP（含子网名、网络名、purpose） |
| ⑥ | 删除主机前检查 | 有绑定 IP 时拒绝删除，返回 409 + 绑定 IP 列表 |
| ⑦ | 创建主机 | 不带 IP，纯创建。绑定 IP 是后续独立操作 |

### 网络侧操作（2 条）

| # | 操作 | 说明 |
|---|------|------|
| ⑧ | 创建 allocation 时指定 host | `/networks/{id}/subnets/{id}/allocations` 创建时可选传 hostId + purpose |
| ⑨ | 删除 allocation 时联动 | 已绑定主机的 allocation 被删除时，同步解除关联 |

## API 设计 — Custom Verb 方案

### 核心思路

**Custom verb 只是路由区分，不是权限边界。** 权限由 HTTP method + 资源决定，custom verb 不产生新权限码。

```
POST /hosts/{hostId}:bindip   → infra:hosts:create （同 POST /hosts 的权限）
GET  /hosts:subnets            → infra:hosts:list   （同 GET /hosts 的权限）
```

能管主机的人自然能操作 IP，零额外权限配置。

### 主机侧 API 路由

```
# 查询可用子网（collection-level custom verb）
GET  /api/infra/v1/hosts:subnets
GET  /api/infra/v1/workspaces/{wsId}/hosts:subnets
GET  /api/infra/v1/workspaces/{wsId}/namespaces/{nsId}/hosts:subnets

# 查看主机绑定的 IP 列表（item-level custom verb）
GET  /api/infra/v1/hosts/{hostId}:ips
GET  /api/infra/v1/workspaces/{wsId}/hosts/{hostId}:ips
GET  /api/infra/v1/workspaces/{wsId}/namespaces/{nsId}/hosts/{hostId}:ips

# 绑定 IP（item-level custom verb，POST）
POST /api/infra/v1/hosts/{hostId}:bindip
POST /api/infra/v1/workspaces/{wsId}/hosts/{hostId}:bindip
POST /api/infra/v1/workspaces/{wsId}/namespaces/{nsId}/hosts/{hostId}:bindip

# 解绑 IP（item-level custom verb，DELETE）
DELETE /api/infra/v1/hosts/{hostId}:unbindip
DELETE /api/infra/v1/workspaces/{wsId}/hosts/{hostId}:unbindip
DELETE /api/infra/v1/workspaces/{wsId}/namespaces/{nsId}/hosts/{hostId}:unbindip

# 释放 IP（item-level custom verb，DELETE）
DELETE /api/infra/v1/hosts/{hostId}:releaseip
DELETE /api/infra/v1/workspaces/{wsId}/hosts/{hostId}:releaseip
DELETE /api/infra/v1/workspaces/{wsId}/namespaces/{nsId}/hosts/{hostId}:releaseip
```

### 权限映射（零新增）

| API | HTTP Method | 复用的权限码 |
|-----|-------------|-------------|
| `GET /hosts:subnets` | GET (collection) | `infra:hosts:list` |
| `GET /hosts/{id}:ips` | GET (item) | `infra:hosts:get` |
| `POST /hosts/{id}:bindip` | POST (item) | `infra:hosts:create` |
| `DELETE /hosts/{id}:unbindip` | DELETE (item) | `infra:hosts:delete` |
| `DELETE /hosts/{id}:releaseip` | DELETE (item) | `infra:hosts:delete` |

权限由 scope 级别的 RBAC role binding 控制，不需要任何额外角色配置。

## 后端代理实现架构

```
用户（仅有 infra 权限）
  │
  ▼
infra REST Storage (custom verb storage)
  │  权限检查：infra:hosts:list / get / create / delete
  │
  ▼
infra Store (pg_network_reader.go / pg_ip_binding.go)
  │  直接查询 networks / subnets / ip_allocations 表
  ▼
PostgreSQL
```

infra 模块定义自己的 store 接口：

```go
// NetworkReader — 查询 public 网络下的可用子网
type NetworkReader interface {
    ListAvailableSubnets(ctx context.Context, query db.ListQuery) (*db.ListResult[AvailableSubnetRow], error)
    GetAvailableSubnet(ctx context.Context, subnetID int64) (*AvailableSubnetRow, error)
}

// IPBinder — 主机 IP 绑定/解绑/释放操作
type IPBinder interface {
    BindIP(ctx context.Context, hostID, subnetID int64, ip string, purpose string) (*BoundIPRow, error)
    UnbindIP(ctx context.Context, allocationID int64) error
    ReleaseIP(ctx context.Context, allocationID int64) error
    ListByHostID(ctx context.Context, hostID int64) ([]BoundIPRow, error)
    CountByHostID(ctx context.Context, hostID int64) (int64, error)
}
```

store 实现直接查 networks/subnets/ip_allocations 表（同库不同查询，不导入 network 包，无跨模块依赖）。

## 框架改动 — Custom Verb 扩展（设计中）

### 核心原则

Custom verb 是围绕标准 REST 资源的附加操作。三种业务场景天然决定了路径级别：

| 场景 | 有资源 ID？ | 路径级别 | 对应接口 |
|------|------------|---------|---------|
| ① 查询资源的关联信息 | ✅ 有（资源已存在） | itemPath:verb | **Getter** |
| ② 创建资源前查询可选项 | ❌ 没有（资源还未创建） | basePath:verb | **Lister** |
| ③ 变更资源的附加数据 | ✅ 有（操作具体资源） | itemPath:verb | **Creator/Deleter/Updater/Patcher** |

**接口即路径，不需要额外的 `Collection bool` 字段。** 与 `registerRoutes` 完全一致的规则：

- Lister → basePath:verb（collection-level）
- Getter → itemPath:verb（item-level）
- Creator → itemPath:verb（POST on item，区别于标准资源的 basePath POST）
- Deleter → itemPath:verb
- Updater → itemPath:verb
- Patcher → itemPath:verb

### CustomVerbInfo 结构变化

```go
// 改前
type CustomVerbInfo struct {
    Name    string
    Storage Lister  // 仅 GET
}

// 改后
type CustomVerbInfo struct {
    Name    string
    Storage Storage // 按实现的接口自动注册对应 HTTP method + 路径级别
}
```

### GetOptions 扩展（支持场景①的分页）

场景①的 Getter 需要支持分页（如 `/users/1:namespaces`），扩展 `GetOptions`：

```go
type GetOptions struct {
    PathParams map[string]string
    ListQuery  *ListOptions      // custom verb GET 时填充，普通 Get 为 nil
}
```

### 主机网络 custom verb 映射

| 操作 | 场景 | 接口 | 注册路径 |
|------|------|------|---------|
| `:subnets` | ② 创建前查可选子网 | Lister | `GET /hosts:subnets` |
| `:ips` | ① 查主机的 IP | Getter | `GET /hosts/{id}:ips` |
| `:bindip` | ③ 绑定 IP | Creator | `POST /hosts/{id}:bindip` |
| `:unbindip` | ③ 解绑 IP | Deleter | `DELETE /hosts/{id}:unbindip` |
| `:releaseip` | ③ 释放 IP | Deleter | `DELETE /hosts/{id}:releaseip` |

### 现有 custom verb 迁移

所有现有 custom verb 均为场景①（查询资源的关联信息），需从 Lister 迁移为 Getter：

| verb | 改动 |
|------|------|
| `/users/{id}:workspaces` | List → Get，*ListOptions → *GetOptions |
| `/users/{id}:namespaces` | 同上 |
| `/users/{id}:rolebindings` | 同上 |
| `/users/{id}:permissions` | 同上 |
| `/hosts/{id}:assignments` | 同上 |

每个文件改约 3 行（方法名 + 参数类型 + 内部取 `options.ListQuery`）。

### 改动清单

| 文件 | 改动 | 代码量 |
|------|------|--------|
| `lib/rest/apigroup.go` | `CustomVerbInfo.Storage` 类型改为 `Storage` | ~2 行 |
| `lib/rest/storage.go` | `GetOptions` 新增 `ListQuery *ListOptions` | ~1 行 |
| `lib/rest/installer.go` | `installCustomVerb` 按接口注册多方法 + basePath/itemPath | ~30 行 |
| `lib/rest/installer.go` | 新增 `customVerbGetHandler`（解析 ListOptions 填入 GetOptions） | ~15 行 |
| `lib/rest/filters/authorization.go` | `ResolveResourceAndVerb` 处理 collection-level `resource:verb` | ~8 行 |
| 现有 custom verb storages | Lister → Getter 迁移（5 个文件） | ~15 行 |
| 测试 | 新增 collection-level + 多方法 + 迁移验证 | ~50 行 |

**总计约 120 行。**

### 待确认

- GetOptions.ListQuery 方案是否最优，还是有更简洁的方式处理场景①的分页
- installCustomVerb 中 Creator 映射到 itemPath（而非标准 REST 的 basePath）是否需要额外说明

## 网络侧变更

`ip_allocations` 响应新增 `hostId`、`hostName`、`purpose` 字段。创建 allocation 时可选传 `hostId` + `purpose`。删除已绑定主机的 allocation 时，`ON DELETE SET NULL` 的 FK 自动处理关联清除。
