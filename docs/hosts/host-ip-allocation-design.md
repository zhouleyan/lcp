# 主机 IP 分配 — 设计文档

> 日期: 2026-03-13

## 业务规则

- 多 IP：创建主机时可添加多条 IP 配置
- 分配方式：手动指定 + 自动分配
- 创建后可增删 IP
- 移除 IP = 解绑（IP 保留占用在子网中，bitmap 不变）
- 删除主机 = 自动解绑所有 IP（host_id 置空）
- 暂不考虑 IP 用途（purpose）字段

## DDD 分析

Host 域和 Network 域是两个独立的限界上下文（Bounded Context）。

在 Host 域中，Network/Subnet 是 **引用数据**（上下游 Customer-Supplier 关系）。Host 域通过 **防腐层（ACL）** 访问网络数据：infra 模块定义自己的 store 直查 DB，不导入 network 包，不经过 network REST 层。

```
Host 域（infra 模块）                    Network 域（network 模块）
┌─────────────────────┐                ┌─────────────────────┐
│  HostStore          │                │  NetworkStore       │
│  NetworkReader ─────┼── 直查 DB ──→  │  networks 表        │
│  IPBinder     ─────┼── 直查 DB ──→  │  subnets 表         │
│                     │                │  ip_allocations 表  │
│  不导入 network 包   │                │                     │
└─────────────────────┘                └─────────────────────┘
```

## 权限方案：PermissionTargets

### 问题

infra 模块需要暴露网络查询 API，但不应产生新的权限轴（如 `infra:networks:list`）。权限应正交——能操作主机的人自然能查可用网络。

### 方案

`ResourceInfo` 新增 `PermissionTargets []string` 字段：

```go
type ResourceInfo struct {
    Name              string
    Storage           Storage
    SubResources      []ResourceInfo
    // 权限覆盖：用户拥有其中任一权限即可访问，支持通配符
    // 为空则走原有自动推导
    PermissionTargets []string
}
```

- 填写完整权限码，支持通配符（如 `infra:hosts:*`）
- OR 语义：用户拥有任一匹配的权限即放行
- 授权中间件：有 PermissionTargets 时用它检查，否则走自动推导
- 权限自动注册：有 PermissionTargets 的资源跳过，不生成新权限记录

### 注册示例

```go
{
    Name:              "networks",
    Storage:           readOnlyNetworkStorage,
    PermissionTargets: []string{"infra:hosts:*"},
}
```

`GET /api/infra/v1/networks` → 检查用户是否有任何 `infra:hosts:*` 权限。

## API 设计

### 故事 1：创建主机时选择网络并分配 IP

| # | API | 类型 | 权限 |
|---|-----|------|------|
| 1 | `GET /api/infra/v1/networks` | 新增（Lister） | `PermissionTargets: ["infra:hosts:*"]` |
| 2 | `POST /api/infra/v1/hosts` | 扩展 body | `infra:hosts:create` |

`GET /networks` 一次返回网络 + 内嵌子网，前端两级联动数据全部就绪：

```json
{
  "items": [
    {
      "name": "prod-network",
      "displayName": "生产网络",
      "cidr": "10.0.0.0/16",
      "subnets": [
        {
          "id": "1",
          "name": "subnet-a",
          "cidr": "10.0.1.0/24",
          "gateway": "10.0.1.1",
          "freeIPs": 200,
          "totalIPs": 254
        }
      ]
    }
  ]
}
```

### 故事 2：查看主机 IP 列表

| # | API | 类型 | 权限 |
|---|-----|------|------|
| 1 | `GET /api/infra/v1/hosts/{hostId}/ips` | 新增（Lister） | `infra:hosts:ips:list` |

### 故事 3：追加 IP

| # | API | 类型 | 权限 |
|---|-----|------|------|
| 1 | `GET /api/infra/v1/networks` | 复用故事 1 | `PermissionTargets: ["infra:hosts:*"]` |
| 2 | `POST /api/infra/v1/hosts/{hostId}/ips` | 新增（Creator） | `infra:hosts:ips:create` |

### 故事 4：移除 IP

| # | API | 类型 | 权限 |
|---|-----|------|------|
| 1 | `DELETE /api/infra/v1/hosts/{hostId}/ips/{allocationId}` | 新增（Deleter） | `infra:hosts:ips:delete` |

### 故事 5：删除主机自动解绑

无新增 API，`DELETE /hosts` 和批量删除内部逻辑变更：删除前将关联 IP 的 host_id 置空。

### 故事 6：网络侧查看 IP 的主机关联

扩展现有 `GET /api/network/v1/.../allocations` 响应，新增 `hostId`、`hostName` 字段。

## DB Schema 变更

```sql
ALTER TABLE ip_allocations ADD COLUMN host_id BIGINT REFERENCES hosts(id) ON DELETE SET NULL;
CREATE INDEX idx_ip_allocations_host_id ON ip_allocations(host_id);
```

## 实现优先级

1. **框架层**：ResourceInfo.PermissionTargets + 授权中间件 + 权限注册适配
2. **数据层**：DB migration + sqlc 查询
3. **业务层**：ACL store + host IP storage + 创建扩展
4. **联动层**：删除自动解绑 + 网络侧字段扩展
