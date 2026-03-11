# 基于 NetBox 的本体论落地实践

## CMDB 与本体论的关系

CMDB 本质上是 IT 领域的应用本体 (Applied Ontology)。

本体论回答：**"世界上存在哪些东西？它们有什么属性？它们之间是什么关系？"**
CMDB 回答：**"IT 环境里有哪些东西？它们的配置是什么？它们如何关联？"**

### 核心概念对照

| 本体论 (Ontology) | CMDB | 例子 |
|-------------------|------|------|
| **类 (Class)** | CI 类型 (CI Type) | "服务器"、"交换机" |
| **个体 (Individual)** | 配置项 (CI) | 编号 SN-001 的那台 Dell 服务器 |
| **属性 (Property)** | CI 属性 | cpu、memory、ip |
| **关系 (Relation)** | CI 关系 | runs_on、located_in、connects_to |
| **层级 (Taxonomy)** | 类型继承 | 网络设备 → 交换机 → 核心交换机 |
| **约束 (Constraint)** | 验证规则 | "虚拟机必须有宿主机" |
| **实例化 (Instantiation)** | 录入资产 | 创建一台具体的服务器记录 |

### 两种 CMDB 设计路线

**固定模型** = 领域本体 (Domain Ontology)
- 预定义所有类和关系（Server, VM, Rack, ...）
- 查询高效，schema 清晰
- 新类型需要改代码

**灵活模型** = 上层本体 (Upper Ontology) 引擎
- 元模型：CIType(name, attributes[]), RelationType(name, source, target)
- 用户自定义类和关系
- 灵活性高，但查询复杂（EAV 模式）

---

## NetBox 的本体能力映射

| 本体要素 | NetBox 实现 |
|---------|------------|
| **类 (Class)** | Device Type, Device Role, Platform 等内置类型 |
| **个体 (Individual)** | 具体的 Device, VM, IP Address 实例 |
| **数据属性 (Data Property)** | 内置字段 + Custom Fields（自定义字段） |
| **对象属性 (Object Property)** | Cables（物理连接）+ Custom Relationships（自定义关系） |
| **分类体系 (Taxonomy)** | Region → Site → Location → Rack（内置层级） |
| **约束 (Constraint)** | Custom Validators（自定义校验脚本） |
| **命名空间 (Namespace)** | Tenant / Tenant Group（多租户隔离） |

---

## 实践路径

### 第一步：建立 TBox（术语层 / 概念模型）

TBox 定义"有哪些类、关系、约束"：

```
1. 规划 Region / Site / Location 层级
   Region: 华东、华北、华南
    └── Site: 上海机房A、北京机房B
         └── Location: A区、B区（楼层/房间）

2. 定义 Device Role（设备角色 = 类）
   - compute-server（计算服务器）
   - core-switch（核心交换机）
   - access-switch（接入交换机）
   - firewall（防火墙）
   - storage-array（存储阵列）

3. 定义 Device Type（设备型号 = 子类）
   Manufacturer: Dell → DeviceType: PowerEdge R750
   Manufacturer: Cisco → DeviceType: Catalyst 9300

4. 用 Custom Fields 扩展属性
   - 给 Device 加 "业务负责人"、"维保到期日"
   - 给 VM 加 "应用名称"、"环境(prod/staging)"

5. 用 Custom Relationships 定义关系
   - "部署于": Application → Device (多对多)
   - "备份到": Device → StorageDevice (多对多)
```

### 第二步：填充 ABox（断言层 / 实例数据）

ABox 定义"具体有哪些个体及其关系"：

```
Device: web-server-01
  ├── Type: Dell PowerEdge R750
  ├── Role: compute-server
  ├── Site: 上海机房A
  ├── Rack: A区-03柜-12U
  ├── IP: 10.0.1.10/24
  ├── Interface: eth0 ──Cable──→ core-sw-01 port Gi0/1
  └── Custom: 负责人=张三, 维保到期=2027-06

VM: app-vm-01
  ├── Cluster: vmware-cluster-01
  ├── Host: web-server-01
  └── Custom: 应用=订单服务, 环境=prod
```

### 第三步：推理与查询（本体的核心价值）

```
本体论推理                          NetBox 实现
─────────────────────────────────────────────────
"这台交换机故障会影响哪些服务器？"  → API: 查 Cable 关系，追溯连接的 Device
"张三负责的所有资产有哪些？"        → API: Custom Field 过滤
"上海机房还能放多少台服务器？"      → API: Rack 剩余 U 位统计
"这个应用的完整依赖链？"           → GraphQL: 递归查询 Relationships
```

### 第四步：LCP 作为本体的应用层

```
        本体层 (NetBox)
        ┌──────────────────────┐
        │  TBox: 类型/关系定义  │
        │  ABox: 实例/连接数据  │
        └──────────┬───────────┘
                   │ API
        应用层 (LCP)
        ┌──────────▼───────────┐
        │  推理: 影响分析       │  ← 故障时，沿关系链查影响范围
        │  决策: 调度部署       │  ← 部署时，查可用资源
        │  展示: 拓扑可视化     │  ← 前端画关系图
        │  治理: 合规检查       │  ← 校验资产是否符合规范
        └──────────────────────┘
```

---

## 与传统本体工具的对比

| 维度 | Protégé + OWL | NetBox |
|------|--------------|--------|
| 建模灵活性 | 极高（任意类/关系） | 中等（内置模型 + 自定义扩展） |
| 推理能力 | 强（OWL 推理机） | 弱（靠 API 查询组合） |
| 工程可用性 | 低（学术工具） | 高（生产级 REST API） |
| IT 领域适配 | 需从零建模 | 开箱即用 |
| 数据录入 | 痛苦 | 友好（Web UI + API + 导入） |

**NetBox 的核心优势：它是一个已经填好了 IT 领域核心 TBox 的实用本体系统。**

---

## 本体论视角下的设计原则

1. **先建分类体系再填数据** — 先想清楚有哪些 CI 类型和关系类型
2. **关系比属性更重要** — 一台服务器的 CPU 型号不如"它连着哪台交换机"有价值
3. **避免过度分类** — 本体论经典陷阱，分得太细反而不实用
4. **开放世界假设** — 没录入不代表不存在，CMDB 永远是不完整的

---

## 本体与 AI Agent 的关系

### 本体是 Agent 的知识底座，不是 Agent 本身

```
本体 (知识结构)  +  数据 (事实)  +  推理 (Agent)  =  智能决策
     TBox             ABox           LLM/规则引擎
```

具体到 LCP + NetBox 场景：

```
NetBox 本体模型          NetBox 实例数据           LCP AI Agent
─────────────           ─────────────           ─────────────
"服务器有IP、CPU..."     "web-01 在A机房"         "机房A的交换机挂了，
"服务器连交换机..."      "web-01 连 sw-01"         影响了web-01和web-02，
"VM跑在服务器上..."      "app-vm 跑在 web-01"      上面跑着订单服务，
                                                  需要通知张三"
 知道有什么              知道具体有哪些             能推理、决策、行动
```

### 没有本体的 Agent vs 有本体的 Agent

**没有本体**（直接扔原始数据给 LLM）：
- 不知道交换机和服务器有连接关系，无法推理
- 靠 LLM 猜测，不可靠

**有本体**（结构化知识 + 关系）：
1. 查本体：交换机通过 Cable 连接服务器（TBox 告诉它怎么查）
2. 查数据：sw-01 连着 web-01, web-02（ABox 告诉它具体事实）
3. 继续追：web-01 上跑着 app-vm-01（沿关系链递归）
4. 决策：通知负责人，触发故障预案

### 本体论实践的四个层次

```
第一层  建模 (本体)     → "IT 世界的地图"
第二层  填充 (数据)     → "地图上标注具体位置"
第三层  查询 (API)      → "能按路线查找"
第四层  推理 (Agent)    → "能用地图导航和决策"
```

前三层是基础。没有好的本体（地图），Agent 就是瞎跑。

### 落地架构

```
┌─────────────────────────────────────┐
│           LCP AI Agent              │
│  MCP Server / Function Calling      │
│  "分析故障影响" "推荐部署方案"        │
└──────────┬──────────────┬───────────┘
           │              │
    ┌──────▼──────┐ ┌────▼────────┐
    │   NetBox    │ │   LCP DB    │
    │  基础设施本体 │ │  应用/部署数据 │
    │  (设备/网络) │ │  (IAM/工作空间)│
    └─────────────┘ └─────────────┘
```

Agent 通过 MCP/Tool Use 同时查 NetBox（基础设施知识）和 LCP（业务知识），组合推理。

**一句话总结：本体是给 Agent 准备的结构化知识库，Agent 是本体的消费者。本体做得越好，Agent 越聪明。**
