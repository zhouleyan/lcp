# 主机监控功能设计

## 概述

在 LCP 平台上为主机接入 VictoriaMetrics 监控能力。平台管理员配置 VM 连接端点，通过 SSH 远程安装 node_exporter，前端在主机详情页 Metrics tab 展示指标图表。

本次 metrics 是 o11y（可观测性）模块的第一步，未来将扩展 logs、traces、APM。

## 核心决策

| 项 | 决定 |
|---|---|
| VM 部署 | LCP 不管，管理员在 UI 上配置连接地址 |
| VM 配置存储 | `o11y_endpoints` 表 + 平台管理员 UI 页面，支持多个实例 |
| 端点分配 | 支持公共（全局可用）、分配到工作空间、分配到项目，多对多关系 |
| 安装方式 | LCP 通过 SSH 远程安装 node_exporter |
| SSH 凭证 | 直接加在 Host 表上（端口、用户名、密码/密钥），不新建凭证表 |
| Target 注册 | LCP 提供 HTTP SD 接口，vmagent 通过 `http_sd_configs` 拉取 |
| 指标展示 | LCP 后端代理查询 VM（PromQL），前端 recharts 画图，不依赖 Grafana |
| 展示位置 | 主机详情页新增 Metrics tab |
| 指标范围 | CPU、内存、磁盘、网络、系统（全量） |
| 时间范围 | 预设（1h/6h/24h/7d/30d）+ 自定义起止时间 |

## 数据模型

### Host 表扩展

```sql
ALTER TABLE hosts ADD COLUMN
    ssh_port        INT         NOT NULL DEFAULT 22,
    ssh_user        VARCHAR(64) NOT NULL DEFAULT 'root',
    ssh_auth_type   VARCHAR(20) NOT NULL DEFAULT 'password',  -- password / private_key
    ssh_password    VARCHAR(512),
    ssh_private_key TEXT,
    monitor_status  VARCHAR(20) NOT NULL DEFAULT 'uninstalled';
    -- uninstalled / installing / installed / failed
```

- `ssh_password` 和 `ssh_private_key` 存储时加密（AES），API 层返回时脱敏
- `monitor_status` 追踪 node_exporter 安装状态
- `monitor_status` 为 `installed` 的主机才会出现在 HTTP SD 的 target 列表中

### o11y_endpoints 表

```sql
CREATE TABLE o11y_endpoints (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(500),
    public      BOOLEAN     NOT NULL DEFAULT false,  -- true=全局可用
    metrics_url VARCHAR(500),    -- VictoriaMetrics
    logs_url    VARCHAR(500),    -- Loki（预留）
    traces_url  VARCHAR(500),    -- Tempo/Jaeger（预留）
    apm_url     VARCHAR(500),    -- APM（预留）
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

一行 = 一套完整的可观测性栈配置，按需填写 URL，未启用的留空。

### o11y_endpoint_assignments 表

```sql
CREATE TABLE o11y_endpoint_assignments (
    id            BIGSERIAL PRIMARY KEY,
    endpoint_id   BIGINT NOT NULL REFERENCES o11y_endpoints(id) ON DELETE CASCADE,
    workspace_id  BIGINT REFERENCES workspaces(id) ON DELETE CASCADE,
    namespace_id  BIGINT REFERENCES namespaces(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(endpoint_id, COALESCE(workspace_id, 0), COALESCE(namespace_id, 0))
);
```

- 只填 `workspace_id` → 分配给该工作空间（及其下属项目）
- 填 `workspace_id` + `namespace_id` → 分配给该项目

### 端点可见性查询

```sql
-- 查某 namespace 可用的端点
SELECT e.* FROM o11y_endpoints e
WHERE e.public = true
   OR e.id IN (SELECT endpoint_id FROM o11y_endpoint_assignments
               WHERE workspace_id = :wsId)
   OR e.id IN (SELECT endpoint_id FROM o11y_endpoint_assignments
               WHERE namespace_id = :nsId)
```

## API 设计

### o11y 模块

```
# 端点管理（平台管理员）
/api/o11y/v1/endpoints                                                      # CRUD + batch delete

# 工作空间级分配（操作 o11y_endpoint_assignments 表）
/api/o11y/v1/workspaces/{workspaceId}/endpoints                             # list + batch add/remove

# 项目级分配（操作 o11y_endpoint_assignments 表）
/api/o11y/v1/workspaces/{workspaceId}/namespaces/{namespaceId}/endpoints    # list + batch add/remove

# Metrics 代理查询（后端转发 PromQL 到 VM）
/api/o11y/v1/endpoints/{endpointId}/metrics/query                           # GET 即时查询
/api/o11y/v1/endpoints/{endpointId}/metrics/query_range                     # GET 范围查询

# HTTP SD 接口（给 vmagent 拉取 target 列表，无需认证）
/api/o11y/v1/discovery/targets                                              # GET Prometheus HTTP SD 格式
```

### infra 模块扩展

```
/api/infra/v1/hosts/{hostId}/install-monitor      # POST 安装 node_exporter
/api/infra/v1/hosts/{hostId}/uninstall-monitor    # POST 卸载 node_exporter
```

### Storage 组织

遵循 CLAUDE.md 中的 storage 组织规则：

| Storage | 路由 | 操作的表 | 模式 |
|---|---|---|---|
| `endpointStorage` | `/endpoints` | `o11y_endpoints` | 端点 CRUD |
| `workspaceEndpointStorage` | `/workspaces/{wsId}/endpoints` | `o11y_endpoint_assignments` | 关联关系，拆分 |
| `namespaceEndpointStorage` | `/workspaces/{wsId}/namespaces/{nsId}/endpoints` | `o11y_endpoint_assignments` | 关联关系，拆分 |

端点管理和分配管理操作的底层表不同，属于关联关系，需要拆分为独立 storage。

## 后端架构

### o11y 模块结构

```
pkg/apis/o11y/
├── types.go                       # Endpoint, EndpointAssignment 类型
├── store.go                       # EndpointStore, EndpointAssignmentStore 接口
├── store/
│   ├── pg_endpoint.go             # 端点 CRUD
│   └── pg_endpoint_assignment.go  # 分配关系
├── storage.go                     # endpointStorage, workspaceEndpointStorage, namespaceEndpointStorage
├── provider.go                    # RESTStorageProvider
├── validation.go
├── discovery.go                   # HTTP SD handler
├── metrics_proxy.go               # PromQL 代理查询
└── v1/install.go                  # 路由注册
```

### infra 模块扩展

```
pkg/apis/infra/
├── ssh.go                         # SSH 连接 + 远程执行封装
├── monitor_installer.go           # node_exporter 安装/卸载逻辑
└── storage.go                     # 新增 install-monitor / uninstall-monitor action
```

### 安装流程

```
POST /hosts/{hostId}/install-monitor
  → 更新 monitor_status = installing
  → 开 goroutine：SSH 连接 → 下载 node_exporter → 配 systemd → 启动
  → 成功：monitor_status = installed
  → 失败：monitor_status = failed
```

前端轮询 Host 的 `monitor_status` 字段获取安装进度。

### metrics_proxy 代理查询

前端传 endpointId + PromQL，后端查出该端点的 `metrics_url`，转发请求到 VM：

```
GET /endpoints/{endpointId}/metrics/query_range?query=...&start=...&end=...&step=...
  → 查 o11y_endpoints 获取 metrics_url
  → 转发：GET {metrics_url}/api/v1/query_range?query=...&start=...&end=...&step=...
  → 返回结果给前端
```

### HTTP SD 接口

查 hosts 表中 `monitor_status = installed` 的主机，返回 Prometheus HTTP SD 格式：

```json
[
  {
    "targets": ["192.168.1.10:9100"],
    "labels": {
      "hostname": "web-01",
      "workspace_id": "1",
      "namespace_id": "2"
    }
  }
]
```

vmagent 配置：

```yaml
scrape_configs:
  - job_name: "lcp-hosts"
    http_sd_configs:
      - url: "http://lcp-server:8428/api/o11y/v1/discovery/targets"
        refresh_interval: 30s
```

### 跨模块依赖

```
o11y 模块
  ├── discovery.go → 依赖 infra.HostStore（读 hosts 表）
  └── metrics_proxy.go → 依赖自己的 EndpointStore

infra 模块
  └── monitor_installer.go → 依赖自己的 HostStore
```

跨模块依赖在 `pkg/apis/install.go` 组装层连接，不在模块间直接引用。

## 前端设计

### 主机详情页 Metrics Tab

- 位置：`ui/src/pages/infra/hosts/detail.tsx` 新增 Metrics tab
- 时间范围选择：预设按钮（1h / 6h / 24h / 7d / 30d）+ 自定义日期时间选择器
- 图表库：recharts（项目已有依赖）

### 指标面板

| 分类 | 指标 | PromQL 示例 |
|---|---|---|
| CPU | 使用率 | `100 - avg(rate(node_cpu_seconds_total{mode="idle",instance="$ip:9100"}[5m])) * 100` |
| CPU | 负载 | `node_load1{instance="$ip:9100"}` / `node_load5` / `node_load15` |
| 内存 | 使用率 | `(1 - node_memory_MemAvailable_bytes/node_memory_MemTotal_bytes) * 100` |
| 内存 | 可用量 | `node_memory_MemAvailable_bytes{instance="$ip:9100"}` |
| 磁盘 | 使用率 | `(1 - node_filesystem_avail_bytes/node_filesystem_size_bytes) * 100` |
| 磁盘 | 读写 IOPS | `rate(node_disk_reads_completed_total[5m])` |
| 磁盘 | 读写吞吐 | `rate(node_disk_read_bytes_total[5m])` |
| 网络 | 收发流量 | `rate(node_network_receive_bytes_total[5m])` |
| 网络 | 收发包量 | `rate(node_network_receive_packets_total[5m])` |
| 系统 | Uptime | `node_time_seconds - node_boot_time_seconds` |
| 系统 | 进程数 | `node_procs_running` / `node_procs_blocked` |

### o11y 端点管理页面

- 平台管理员页面：端点 CRUD，查看分配状态
- 工作空间/项目页面：查看可用端点列表

## 未来扩展

- `logs_url` → 对接 Loki，主机日志查看
- `traces_url` → 对接 Tempo/Jaeger，链路追踪
- `apm_url` → 应用性能监控
- 告警规则配置 → 对接 vmalert
