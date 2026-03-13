# 工作流审批系统设计

> 状态：草稿（待继续讨论）
> 日期：2026-03-13

## 需求概述

LCP 平台需要一套可配置的工作流审批系统，支持对业务操作进行流程审批控制。

### 覆盖场景

- **资源生命周期审批** — 创建/删除工作空间、项目等关键资源
- **运维操作审批** — 主机操作、数据库变更等高风险操作
- **权限变更审批** — 角色绑定、成员添加、所有权转移

### 核心决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 审批流复杂度 | 线性 + 会签/或签 | 后续可扩展条件分支 |
| 流程定义方式 | 配置化（API + DB） | 管理员通过 API 配置，无需改代码 |
| 审批人指定 | 用户 + 角色 | 支持指定具体用户或按角色匹配 |
| 请求处理模式 | 拦截并挂起 | 请求不立即执行，审批通过后系统自动执行 |
| 作用域 | 平台级统一配置 | 只有平台管理员可配置 |
| 工单操作 | 通过、驳回、查看、列表 | 第一版不做撤回/转审 |
| 通知方式 | 预留 hook，第一版轮询 | 后续可接 WebSocket 推送 |
| 前端交互 | react-flow 画布拖拉拽 | 直接上画布，为 DAG 扩展预留 |

## 架构方案：REST Filter 拦截

在现有 REST filter chain 中新增 `WorkflowFilter`，在请求到达 Storage 之前判断是否需要审批。

```
HTTP Request
  → AuthN Filter
    → AuthZ Filter
      → 【WorkflowFilter】 ← 新增
        → 匹配审批规则？
          → 是：序列化请求 → 创建工单 → 返回 202 Accepted
          → 否：放行 → Storage 正常执行
```

### 为什么选 Filter 而非 Storage Decorator

- **零侵入** — 不改任何现有 Storage 代码
- **与现有架构一致** — AuthN、AuthZ 都是 filter，工作流审批本质是"准入控制"
- **规则匹配天然对齐 RBAC** — `(module, resource, verb)` 三元组与权限 code 同构

## 数据模型

### workflow_rules（审批规则）

```sql
CREATE TABLE workflow_rules (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,          -- 规则名称，如"创建主机需要审批"
    description TEXT,
    module      VARCHAR(50)  NOT NULL,          -- 模块，如 "iam", "o11y"
    resource    VARCHAR(50)  NOT NULL,          -- 资源，如 "hosts", "workspaces"
    verb        VARCHAR(20)  NOT NULL,          -- 操作，如 "create", "delete"
    enabled     BOOLEAN      NOT NULL DEFAULT true,
    definition  JSONB        NOT NULL,          -- 流程定义（nodes + edges，对接 react-flow）
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(module, resource, verb)
);
```

### workflow_requests（审批工单）

```sql
CREATE TABLE workflow_requests (
    id              BIGSERIAL PRIMARY KEY,
    rule_id         BIGINT       NOT NULL REFERENCES workflow_rules(id),
    requester_id    BIGINT       NOT NULL REFERENCES users(id),
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',  -- pending/approved/rejected
    module          VARCHAR(50)  NOT NULL,
    resource        VARCHAR(50)  NOT NULL,
    verb            VARCHAR(20)  NOT NULL,
    request_path    TEXT         NOT NULL,       -- 原始请求路径
    request_body    JSONB,                       -- 序列化的请求参数
    request_context JSONB,                       -- path params, query params 等上下文
    current_stage   VARCHAR(50)  NOT NULL,       -- 当前所在节点 ID
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

### workflow_approvals（审批记录）

```sql
CREATE TABLE workflow_approvals (
    id           BIGSERIAL PRIMARY KEY,
    request_id   BIGINT      NOT NULL REFERENCES workflow_requests(id),
    stage_id     VARCHAR(50) NOT NULL,           -- 节点 ID
    approver_id  BIGINT      NOT NULL REFERENCES users(id),
    decision     VARCHAR(10) NOT NULL,           -- "approved" 或 "rejected"
    comment      TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### definition JSONB 结构

直接映射 react-flow 的 nodes + edges，前后端一致：

```json
{
  "nodes": [
    {
      "id": "start",
      "type": "start",
      "position": {"x": 0, "y": 0}
    },
    {
      "id": "stage-1",
      "type": "approval",
      "position": {"x": 200, "y": 0},
      "data": {
        "name": "经理审批",
        "approveType": "any",
        "approvers": [
          {"type": "user", "targetId": "123"},
          {"type": "role", "targetId": "456"}
        ]
      }
    },
    {
      "id": "end",
      "type": "end",
      "position": {"x": 400, "y": 0}
    }
  ],
  "edges": [
    {"source": "start", "target": "stage-1"},
    {"source": "stage-1", "target": "end"}
  ]
}
```

## 实现模块估算

| 模块 | 文件数 | 说明 |
|------|-------|------|
| DB migration | 1 | 3 张表 |
| sqlc queries | 3 | 每张表一个 |
| workflow module (types/store/storage) | ~6 | 套现有 IAM 模式 |
| WorkflowFilter | 1 | 参考 AuthZ filter |
| 审批引擎 (stage 推进 + 请求回放) | 1-2 | 核心新逻辑 |
| install + wiring | 2 | 路由注册 + 组装 |

核心新逻辑集中在 **WorkflowFilter + 审批引擎 + 请求回放**，其余按现有模式实现。

## 待讨论

- [ ] API 路由设计（审批规则 CRUD + 工单操作）
- [ ] WorkflowFilter 详细流程（规则缓存、匹配逻辑）
- [ ] 请求回放机制（如何构造内部请求上下文）
- [ ] 审批引擎状态机细节（会签/或签判定、stage 推进）
- [ ] V1 对 definition 的校验规则（限制为线性链）
- [ ] 前端 react-flow 画布的节点类型定义
- [ ] 通知 hook 接口预留设计
