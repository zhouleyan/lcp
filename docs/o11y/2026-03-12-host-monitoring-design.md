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
| 监控接入时机 | 创建主机时可选填 endpointId，也可后续通过 action 按钮接入；创建和安装异步分离 |
| endpoint 关联粒度 | endpoint_id 直接挂在 host 表上，不通过 environment 间接关联 |
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
    endpoint_id     BIGINT REFERENCES o11y_endpoints(id),     -- 关联的监控端点
    monitor_status  VARCHAR(20) NOT NULL DEFAULT 'unmonitored',
    monitor_message VARCHAR(500);
    -- unmonitored / installing / installed / failed
```

- `ssh_password` 和 `ssh_private_key` 存储时加密（AES），API 层返回时脱敏
- `endpoint_id` 直接挂在 host 上，同一环境内的主机可接入不同监控端点
- `monitor_status` 追踪 node_exporter 安装状态
- `monitor_message` 记录失败原因，成功时为空
- `monitor_status` 为 `installed` 的主机才会出现在 HTTP SD 的 target 列表中

### 监控状态机

```
                       创建主机（未选监控）
                              │
                              ↓
                        unmonitored ←──────── uninstall-monitor
                              │
              选了监控 / 点接入按钮
                              │
                              ↓
                        installing ──失败──→ failed
                              │                 │
                              成功          点"重试"
                              ↓                 │
                         installed ←────────────┘
```

### 监控接入流程

创建主机和安装监控异步分离：创建立即返回，安装后台执行。

无论新建主机还是已有主机，都走同一个 action：

```
                    ┌── 创建主机时填了 endpointId ──┐
                    │                               ↓
                    │                    自动调用 install-monitor
                    │                               ↓
用户点"接入监控" ────┴──→ POST /hosts/{id}/install-monitor
                              body: { endpointId }
                                     │
                         ┌───────────┼───────────┐
                         ↓           ↓           ↓
                    校验连通性    安装 exporter   注册到 SD
                         │           │           │
                         └───────────┼───────────┘
                                     ↓
                              更新 monitor_status
```

- 创建主机时填了 endpointId → 主机创建成功后自动触发 install-monitor
- 已有主机 → 前端提供"接入监控"按钮，调用同一个 action
- 失败 → monitor_status=failed + monitor_message 记录原因，前端提供"重试"按钮
- 前端轮询主机的 monitor_status 字段获取安装进度

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

### 软件包管理：统一 YUM 源

所有需要部署的软件（node_exporter、未来的 MySQL agent、Redis exporter 等）统一打成 RPM 包，通过离线 YUM 源分发。没有官方 RPM 的二进制软件（如 node_exporter）由 LCP 团队预先打包。

**为什么不支持多格式（binary + RPM + DEB）**：
- 统一 RPM 后安装/卸载/升级都是 `yum install/remove/update`，LCP 不需要管安装细节
- systemd unit、用户创建、文件权限全在 RPM spec 里定义，一致性有保障
- 版本管理、依赖管理交给 yum 处理

**LCP 只需管理 YUM 源地址**，不需要自己存文件：

```sql
-- 复用 artifact_repositories 表，或直接在配置文件中指定
-- 目标主机的 /etc/yum.repos.d/ 需预先配置好内部 YUM 源
-- LCP 负责的是：SSH 上去执行 yum install
```

**RPM 打包**：`packaging/rpm/` 目录下为每个需要打包的软件维护 spec 文件和构建脚本。

```
packaging/rpm/
├── node_exporter/
│   ├── node_exporter.spec      # RPM spec 文件
│   ├── node_exporter.service   # systemd unit
│   ├── node_exporter.default   # 环境变量配置
│   └── build.sh                # 下载二进制 + 构建 RPM
└── (未来其他软件)/
```

构建产物上传到内部 YUM 源（如 Nexus raw-hosted repo），目标主机通过 `yum install node_exporter` 安装。

### 安装流程（详细）

```
POST /hosts/{id}/install-monitor { endpointId: 5 }
  │
  ├── 更新 host: endpoint_id=5, monitor_status=installing
  │
  ├── 异步 goroutine ─────────────────────────────────────
  │     │
  │     ├── Step 1: SSH 连通性校验
  │     │     ssh {user}@{ip} -p {port} "echo ok"
  │     │     └── 失败 → failed + "SSH: connection refused" / "auth failed"
  │     │
  │     ├── Step 2: 安装 node_exporter
  │     │     ssh "yum install -y node_exporter"
  │     │     └── 失败 → failed + "yum install failed: package not found in repo"
  │     │
  │     ├── Step 3: 启动服务
  │     │     ssh "systemctl enable node_exporter && systemctl start node_exporter"
  │     │     └── 失败 → failed + "service start failed"
  │     │
  │     ├── Step 4: 验证 - 进程检查
  │     │     ssh "systemctl is-active node_exporter"
  │     │     └── 期望: "active"
  │     │
  │     ├── Step 5: 验证 - 指标端口检查
  │     │     ssh "curl -sf http://localhost:9100/metrics > /dev/null"
  │     │     └── 期望: exit 0
  │     │
  │     └── 全部通过 → installed, monitor_message=""
  │
  └── 立即返回 202 Accepted（前端轮询 monitor_status）
```

**卸载流程**：
```
POST /hosts/{id}/uninstall-monitor
  → ssh "yum remove -y node_exporter"
  → 更新 monitor_status=unmonitored, endpoint_id=NULL
```

**验证说明**：
- Step 4-5 在目标主机上执行，确认 exporter 进程正常、端口可访问
- 不等待 VM 侧确认（vmagent 抓取有 30-60s 延迟），①② 通过即标记 installed
- VM 可达性可作为后续健康检查，不阻塞安装流程

**前提条件**：目标主机的 `/etc/yum.repos.d/` 需预先配置好内部 YUM 源。这是基础设施准备工作，不由 LCP 自动完成。

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

## 主机列表页实时指标

### 设计决策

后端做 Prometheus/VictoriaMetrics 查询代理，前端 30s 轮询。不用 WebSocket/SSE。

参考 Dynatrace 等成熟监控产品：dashboard 数据都是前端轮询而非服务端推送。监控采集周期 10-60s，推送在体验上没有实质优势，反而增加连接管理复杂度。

### 前端轮询策略

```
页面加载
  ├── GET /hosts → 渲染列表
  ├── GET /hosts/metrics?ids=1,2,...,20 → 填充指标列（立即请求）
  │
  │   ┌── 30s 定时器 ──┐
  │   │  GET /hosts/metrics?ids=1,2,...,20  → 更新指标列
  │   └── ...          ┘
  │
  ├── 用户翻页 / 切换筛选
  │   ├── 取消旧请求（AbortController）
  │   ├── GET /hosts → 渲染新列表
  │   ├── GET /hosts/metrics?ids=21,...,40 → 立即请求新页指标
  │   └── 重启 30s 定时器（新 id 列表）
  │
  └── 页面离开 → 清除定时器
```

边界情况：
- 翻页时上一次 metrics 请求未返回 → AbortController 取消，用新页 id 发新请求
- 浏览器 tab 不可见 → `visibilitychange` 暂停轮询，切回时立即请求一次再恢复
- metrics 请求超时/失败 → 指标列显示 `--` 或上次值，不阻塞列表，下次轮询自动重试
- 单台主机监控源不可达 → 后端返回该主机 `metrics: null`，前端显示 `--`

### 后端批量指标代理 API

```
GET /api/o11y/v1/hosts/metrics?ids=1,2,3&metrics=cpu,mem,disk
```

请求参数：
| 参数 | 类型 | 说明 |
|---|---|---|
| `ids` | string（逗号分隔） | 主机 ID 列表，上限跟 pageSize 对齐（最多 100） |
| `metrics` | string（逗号分隔，可选） | 指定要查的指标，默认返回全部常用指标 |

响应结构：
```json
{
  "apiVersion": "o11y/v1",
  "kind": "HostMetricsList",
  "items": [
    {
      "hostId": "1",
      "timestamp": "2026-03-13T10:00:00Z",
      "metrics": {
        "cpu_usage": 45.2,
        "mem_usage": 72.1,
        "disk_usage": 55.0,
        "load1": 2.3
      }
    },
    {
      "hostId": "2",
      "metrics": null,
      "error": "endpoint unreachable"
    }
  ]
}
```

### 后端内部流程

```
收到请求 ids=1,2,3
  │
  ├── 1. 批量查 hosts 表，拿到每台主机的 hostname/ip + endpoint_id
  │      SELECT h.id, h.hostname, h.ip_address, e.endpoint_id
  │      FROM hosts h JOIN environments e ON h.environment_id = e.id
  │      WHERE h.id IN (1,2,3)
  │
  ├── 2. 按 endpoint_id 分组
  │      endpoint_1: [host-a, host-b]
  │      endpoint_2: [host-c]
  │
  ├── 3. 并发查询每个 VM/Prometheus 端点（4 个指标 = 4 次并发 HTTP）
  │      每条 PromQL 用正则批量匹配：instance=~"host-a|host-b"
  │
  ├── 4. 合并结果，按 hostId 组织返回
  │
  └── 5. 某个 endpoint 超时/失败 → 对应主机返回 metrics: null
```

### VM 集群版调用细节（基于 ly-vm 实测）

VM 集群版（vmselect）API 路径需要加租户前缀：`/select/{accountID}/prometheus/api/v1/query`

实际调用的 API 只需要一个：
```
POST {metrics_url}/select/0/prometheus/api/v1/query
```

主机列表所需的 PromQL（基于 ly-vm 实测验证）：

| 指标 | PromQL | 实测数据示例 |
|---|---|---|
| CPU 使用率 % | `100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle",job="node-exporter",instance=~"$instances"}[5m])) * 100)` | 172.24.160.211: 6.1%, kff-lyjkpt-ap1: 16.3% |
| 内存使用率 % | `(1 - node_memory_MemAvailable_bytes{job="node-exporter",instance=~"$instances"} / node_memory_MemTotal_bytes{job="node-exporter",instance=~"$instances"}) * 100` | 172.24.160.211: 39.2%, kff-lyjkpt-ap1: 79.8% |
| 磁盘使用率 % | `(1 - node_filesystem_avail_bytes{mountpoint="/",fstype!="tmpfs",instance=~"$instances"} / node_filesystem_size_bytes{mountpoint="/",fstype!="tmpfs",instance=~"$instances"}) * 100` | 需过滤 `mountpoint="/"` |
| 1min 负载 | `node_load1{job="node-exporter",instance=~"$instances"}` | 172.24.160.211: 0.94, kff-lyjkpt-ap1: 2.46 |

注意事项：
- **VM 集群版路径**：`metrics_url` 存的是基础地址，调用时需拼 `/select/{accountID}/prometheus` 前缀。需在 endpoint 配置中额外存储 `accountID`，或让用户在 `metrics_url` 中填写完整前缀
- **instance 标签匹配**：实测发现 instance 标签有两种格式——IP（`172.24.160.211`）和 hostname（`kff-lyjkpt-ap1`），后端需用主机的 `ipAddress` 和 `hostname` 两个字段去匹配
- **job 过滤**：当前环境固定 `job="node-exporter"`，建议做成可配置项

## 未来扩展

- `logs_url` → 对接 Loki，主机日志查看
- `traces_url` → 对接 Tempo/Jaeger，链路追踪
- `apm_url` → 应用性能监控
- 告警规则配置 → 对接 vmalert
