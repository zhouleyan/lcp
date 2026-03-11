# AI Agent 故障诊断与自动恢复方案

## 整体思路

```
         用户/告警触发
              │
              ▼
┌──────────────────────────┐
│      LCP AI Agent        │
│                          │
│  1. 感知: 发生了什么      │
│  2. 定位: 影响了什么      │
│  3. 分析: 根因是什么      │
│  4. 决策: 怎么恢复       │
│  5. 执行: 自动修复       │
│  6. 验证: 恢复了没有      │
└──────┬───┬───┬───┬───────┘
       │   │   │   │
    CMDB  监控  日志  执行器
```

核心逻辑是一个推理链，每一步需要不同的数据源。

---

## 推理链详解

### 第一步：感知 — 发生了什么

```
告警输入: "订单服务 (order-service) 响应超时，错误率 > 50%"

Agent 动作:
  → 查 LCP: order-service 部署在哪？(app-vm-01, app-vm-02)
  → 查监控: 这两台 VM 的 CPU/内存/网络指标
  → 查日志: order-service 最近 5 分钟的错误日志
```

### 第二步：定位 — 沿关系链展开影响面

CMDB 本体能力的核心价值，一次图遍历拿到整条链路：

```
order-service (应用)
  ├── 部署于 app-vm-01 (VM)
  │    └── 宿主机 web-server-01 (物理机)
  │         ├── 网卡 eth0 ──线缆──→ access-sw-01 (接入交换机)
  │         │                        └── 上联 ──→ core-sw-01 (核心交换机)
  │         ├── 挂载 volume-01 ──→ storage-01 (存储)
  │         └── 机柜 A区-03柜 ──→ 上海机房A
  ├── 部署于 app-vm-02 (VM)
  │    └── 宿主机 web-server-02 ...
  ├── 依赖 mysql-master (数据库)
  │    └── 部署于 db-server-01 ...
  └── 依赖 redis-cluster (缓存)
       └── 部署于 cache-server-01 ...
```

### 第三步：分析 — 交叉比对找根因

```
Agent 拿到所有关联资源后，批量查监控:

  app-vm-01     CPU: 30%  ✓   内存: 40%  ✓   网络: 正常  ✓
  app-vm-02     CPU: 25%  ✓   内存: 35%  ✓   网络: 正常  ✓
  web-server-01 CPU: 20%  ✓   内存: 30%  ✓   磁盘IO: 正常  ✓
  access-sw-01  端口: 正常 ✓   丢包率: 0%  ✓
  mysql-master  CPU: 95%  ✗   连接数: 500/500 ✗   慢查询: 120/min  ✗
  redis-cluster 正常 ✓

LLM 推理:
  "order-service 超时的根因是 mysql-master 负载过高：
   CPU 95%，连接数打满 500，慢查询 120/min。
   数据库成为瓶颈，导致上游服务响应超时。"
```

### 第四步：决策 + 执行 — 自动恢复

```
Agent 生成恢复方案:

  紧急措施 (自动执行):
    1. Kill 慢查询 TOP 5
    2. mysql 连接数临时扩到 800
    3. order-service 开启降级模式，非核心接口熔断

  后续措施 (需人工确认):
    4. 慢查询 SQL 优化建议: SELECT ... 缺少索引
    5. 是否扩容: 增加 mysql 只读副本
```

---

## 技术实现

### 架构

```
┌─────────────────────────────────────────────────────┐
│                    LCP AI Agent                      │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │ MCP Tools│  │ 推理引擎  │  │ Runbook 执行器    │  │
│  │          │  │ (LLM)    │  │                   │  │
│  │ cmdb_*   │  │          │  │ kill_slow_query() │  │
│  │ monitor_*│  │ 关系遍历  │  │ scale_service()   │  │
│  │ log_*    │  │ 指标比对  │  │ toggle_breaker()  │  │
│  │ exec_*   │  │ 根因推理  │  │ restart_pod()     │  │
│  └────┬─────┘  └────┬─────┘  └────────┬──────────┘  │
└───────┼──────────────┼─────────────────┼─────────────┘
        │              │                 │
   ┌────▼────┐   ┌────▼─────┐    ┌─────▼──────┐
   │数据源    │   │ Claude   │    │ Ansible/   │
   │NetBox   │   │ API      │    │ SSH/K8s    │
   │Prometheus│  │          │    │ API        │
   │ELK/Loki │   │          │    │            │
   └─────────┘   └──────────┘    └────────────┘
```

### MCP Tools（Agent 的手和眼）

```
# CMDB 工具 — 查关系
cmdb_get_resource(name) → 资源详情
cmdb_get_dependencies(name, depth=3) → 依赖链（向下）
cmdb_get_dependents(name, depth=3) → 被依赖链（向上）
cmdb_get_connected_devices(device) → 网络连接拓扑

# 监控工具 — 查指标
monitor_get_metrics(resource, metrics[], time_range)
monitor_get_alerts(resource, time_range)

# 日志工具 — 查日志
log_search(service, level="error", time_range, limit)

# 执行工具 — 做恢复
exec_run_playbook(playbook, targets, params)
exec_restart_service(service)
exec_scale_replicas(service, count)
```

### Agent 推理 Prompt 骨架

```
你是 LCP 运维 Agent。收到告警后按以下步骤分析:

1. 用 cmdb_get_dependencies 获取故障服务的完整依赖链
2. 对依赖链上的每个资源，用 monitor_get_metrics 查关键指标
3. 找出指标异常的资源，用 log_search 查错误日志
4. 综合分析根因，输出:
   - 根因资源 + 证据（哪些指标异常）
   - 影响范围（哪些服务受影响）
   - 恢复方案（紧急 + 后续）
5. 紧急措施自动执行，后续措施等人工确认
```

---

## 分阶段落地

```
阶段一  手动查询
        Agent 能查 CMDB 关系 + 监控指标，输出分析报告
        人工执行恢复

阶段二  半自动
        Agent 输出恢复方案，人工确认后自动执行
        积累 Runbook 库

阶段三  全自动
        常见故障模式自动识别 + 自动恢复
        人工只处理未知故障
```
