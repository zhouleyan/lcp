# CMDB 设计讨论记录

## CMDB 管理的资源范围

### 基础设施层
- 数据中心、机房、机柜、电力、制冷
- 物理服务器（品牌、型号、SN、CPU/内存/磁盘、位置）
- 网络设备（交换机、路由器、防火墙、负载均衡器）
- 存储设备（SAN、NAS、磁盘阵列）

### 虚拟化与云资源
- 虚拟机（宿主机关系、规格、镜像）
- 容器/Pod（K8s集群、节点、命名空间）
- 云账号/区域/VPC

### 网络资源
- IP地址/子网（IPAM，项目已有 `lib/ipam`）
- 域名/DNS记录、VIP/浮动IP、SSL证书

### 应用层
- 应用/服务、中间件实例、配置项

### 关系与拓扑
- 依赖关系（服务间调用链）
- 部署关系（应用部署在哪些主机/容器上）
- 网络连接（端口与服务的开放关系）

### 运维相关
- 变更记录、责任人/团队

---

## 基础设施层资源关联关系

### 物理包含层级（树形）

```
Region (地域/区域)
 └── DataCenter (数据中心/机房)
      ├── Room (机房楼层/房间)
      │    └── Rack (机柜)
      │         ├── Server (物理服务器)
      │         │    ├── NIC (网卡)
      │         │    ├── Disk (磁盘)
      │         │    └── GPU (加速卡)
      │         ├── NetworkDevice (网络设备)
      │         │    └── Port (端口) ← 与网卡链路连接
      │         └── StorageDevice (存储设备)
      │              └── Volume (存储卷) ← Server 挂载
      ├── PowerSystem (供电系统)
      │    └── PDU (配电单元) → Rack 供电
      └── CoolingSystem (制冷系统)
```

### 关系类型

| 关系       | 示例                       | 类型     |
|------------|----------------------------|----------|
| 物理包含   | 数据中心 → 机柜 → 服务器   | 树形层级 |
| 网络连接   | 服务器网卡 ↔ 交换机端口    | 多对多   |
| 存储挂载   | 服务器 ↔ 存储卷            | 多对多   |
| 供电       | PDU → 机柜                 | 一对多   |
| 冗余/备份  | 主服务器 ↔ 备服务器        | 对等     |

---

## 虚拟机到数据中心的关系链

```
VM ──runs_on──→ Server ──located_in──→ Rack ──located_in──→ DataCenter ──located_in──→ Region
```

集群化部署（VMware/OpenStack）会多一层逻辑抽象：

```
Region → DataCenter → Cluster(逻辑分组) → Server → VM
```

VM 迁移会改变 VM → Server 的关系，用 `host_server_id` 外键表达当前归属。

---

## 实现顺序

```
第一步  Server（物理服务器）   ← 核心节点，先跑通
第二步  DataCenter             ← 给 Server 挂物理位置
第三步  Rack                   ← 细化位置层级
第四步  NetworkDevice          ← 交换机、防火墙
第五步  VM                     ← 虚拟化层
```

---

## 设计决策：物理机与虚拟机使用单表（Host）

物理机和虚拟机共享大部分属性，采用单表 + `type` 字段 + `parent_id` 自引用的方式。

### 表结构

```
Host {
  id
  name
  type              // physical | virtual
  ip
  cpu
  memory
  disk
  os
  status            // online | offline | maintenance

  // 物理机专属
  sn                // 虚拟机为空
  manufacturer
  model
  rack_position

  // 虚拟机专属
  parent_id → Host  // 宿主机（自引用，物理机为空）
  hypervisor        // KVM/VMware/Hyper-V
}
```

### 好处
- 查询简单，不用 UNION 两张表
- 关系清晰，`WHERE parent_id = ?` 查出物理机上的所有 VM
- API 统一，一套 `/hosts` 的 CRUD，用 `type` 过滤
- 统计方便，物理机总数、VM 总数、资源利用率都在一张表里算

### 验证约束

用验证层（而非数据库约束）区分类型差异：
- `physical`: sn 必填，parent_id 必须为空
- `virtual`: parent_id 必填且必须指向 physical 类型的 host，sn 可为空

---

## 网络设备详细分类

### 核心设备

| 设备 | 作用 | 典型厂商 |
|------|------|----------|
| **交换机 (Switch)** | 二层/三层转发，连接同网段设备 | Cisco Catalyst、华为 CloudEngine、H3C |
| **路由器 (Router)** | 跨网段路由，连接不同网络/出口 | Cisco ISR、华为 NetEngine、Juniper |
| **防火墙 (Firewall)** | 安全策略，流量过滤，区域隔离 | Palo Alto、Fortinet、深信服、华为 USG |
| **负载均衡器 (LB)** | 流量分发到后端服务器集群 | F5 BIG-IP、A10、Nginx(软件)、LVS(软件) |

### 安全设备

| 设备 | 作用 |
|------|------|
| **WAF** | Web应用防火墙，防SQL注入/XSS等 |
| **IDS/IPS** | 入侵检测/防御，分析异常流量 |
| **VPN网关** | 远程接入、站点互联加密隧道 |
| **堡垒机** | 运维审计，管控服务器访问入口 |

### 辅助设备

| 设备 | 作用 |
|------|------|
| **光纤收发器** | 电信号 ↔ 光信号转换 |
| **配线架 (Patch Panel)** | 线缆集中管理，整理机柜布线 |
| **带外管理 (OOB)** | 独立管理网络，服务器宕机时仍可远程操控 |

### 在数据中心里的位置

```
互联网
  │
路由器 (出口)
  │
防火墙 (安全边界)
  │
核心交换机 (三层，跨VLAN路由)
  ├── 汇聚交换机
  │    ├── 接入交换机 ── 服务器群
  │    └── 接入交换机 ── 服务器群
  └── 负载均衡器 ── 应用服务器集群
```

### CMDB 管理网络设备的价值

- **故障定位** — 服务器不通时，查 CMDB 知道它接在哪台交换机的哪个端口，快速排查
- **变更影响分析** — 升级核心交换机前，查出下挂的服务器和业务，评估影响范围
- **容量规划** — 交换机端口剩余数、防火墙吞吐量是否到瓶颈

### 第一期建议只管交换机和防火墙

- 交换机直接关联服务器（有链路关系），是网络拓扑的核心
- 防火墙关联安全策略，运维高频查询
- 其他设备后续扩展

---

## 存储设备详细分类

### 按架构分

| 设备 | 全称 | 特点 | 典型场景 |
|------|------|------|----------|
| **SAN** | Storage Area Network | 块存储，光纤/iSCSI连接，高性能 | 数据库、虚拟化存储 |
| **NAS** | Network Attached Storage | 文件存储，NFS/SMB协议，共享方便 | 文件共享、日志归档、备份 |
| **DAS** | Direct Attached Storage | 直连服务器，无网络开销 | 单机高IO场景 |
| **分布式存储** | Ceph/MinIO/GlusterFS | 软件定义，横向扩展 | 云平台、对象存储、大数据 |

### 按存储介质分

| 类型 | 特点 |
|------|------|
| **全闪存阵列 (AFA)** | 全SSD，高IOPS低延迟，贵 |
| **混合阵列** | SSD做缓存 + HDD做容量，性价比 |
| **机械盘阵列** | 大容量低成本，冷数据/备份 |

### 典型厂商

| 厂商 | 产品线 |
|------|--------|
| Dell EMC | PowerStore、Unity、Isilon(NAS) |
| 华为 | OceanStor Dorado(闪存)、OceanStor(混合) |
| NetApp | AFF(闪存)、FAS(混合)、ONTAP |
| HPE | 3PAR、Nimble |
| 开源 | Ceph、MinIO、GlusterFS |

### 在数据中心里的位置

```
Server
  ├── DAS (直连本地磁盘)
  ├── SAN (光纤/iSCSI) ── 存储阵列 ── 磁盘组 ── LUN/Volume
  └── NAS (以太网) ──── NAS设备 ── 文件系统 ── Share/Export

分布式存储:
  多台 Server 的本地磁盘 ── Ceph/MinIO 集群 ── Pool ── Volume/Bucket
```

### CMDB 需要管理的存储层级

```
StorageDevice (存储设备)
  ├── model, vendor, sn, firmware
  ├── type: san | nas | das | distributed
  ├── rack_id → Rack (物理位置)
  ├── total_capacity, used_capacity
  │
  └── Volume (存储卷/LUN)
       ├── name, capacity, type(ssd/hdd)
       └── mount → Host (挂载关系，多对多)
```

### CMDB 管理存储设备的价值

- **容量管理** — 总容量、已用、剩余，提前预警扩容
- **挂载关系** — 哪台服务器用了哪个卷，故障时快速定位影响范围
- **性能分级** — SSD卷给数据库，HDD卷给备份，资源合理分配

### 第一期建议

先管 StorageDevice + Volume + 挂载关系，不区分 SAN/NAS 的协议细节，用 `type` 字段区分即可。和 Host 表的设计思路一致。

---

## 开源 CMDB 项目参考

### 推荐关注

| 项目 | 技术栈 | GitHub Stars | 特点 |
|------|--------|-------------|------|
| **NetBox** | Python/Django | ~19.5k | 网络/数据中心基础设施管理，社区最活跃，插件生态丰富 |
| **VeOps CMDB** | Python + Vue | ~2k+ | 国产，灵活自定义模型，支持自动发现，轻量好上手 |
| **iTop** | PHP/MySQL | — | ITIL 标准，功能全面，企业级，偏重 ITSM |
| **GLPI** | PHP | ~4k+ | 资产管理 + 工单系统，插件多 |
| **Ralph** | Python/Django | ~2k+ | 数据中心资产全生命周期管理 |
| **Snipe-IT** | PHP/Laravel | ~11k+ | IT 资产管理，界面友好，部署简单 |

### 重点参考

**NetBox** (https://github.com/netbox-community/netbox)
- 专注数据中心基础设施：机柜、设备、IP、线缆、电力
- 数据模型设计成熟，和本项目讨论的 CMDB 资源层级高度吻合
- REST API + GraphQL，Go 项目对接方便
- Apache 2.0 协议
- 学习数据模型设计看 `netbox/dcim/models/`（Site → Rack → Device → Interface）

**VeOps CMDB** (https://github.com/veops/cmdb)
- 自定义模型（不写代码定义资源类型和关系）
- 自动发现（服务器、网络设备、存储、中间件、云资源）
- 有 CI 关系图、仪表盘等可视化
- 和本项目 PaaS 平台定位比较接近

---

## 设计决策：集成 NetBox 而非自建 CMDB

LCP 专注 PaaS 云服务管理与应用托管，CMDB 能力通过集成 NetBox 实现，不重复造轮子。

### 整体架构

```
┌─────────────────────────────────────────────────┐
│                   LCP 前端                       │
│   PaaS管理 UI  │  CMDB UI (代理NetBox页面/数据)  │
└────────┬────────────────────┬────────────────────┘
         │                    │
┌────────▼────────┐  ┌───────▼─────────┐
│   LCP Server    │  │  LCP Server     │
│   PaaS 业务API  │  │  CMDB 代理层    │
│   /api/iam/...  │  │  /api/cmdb/...  │
└────────┬────────┘  └───────┬─────────┘
         │                    │ REST API / GraphQL
┌────────▼────────┐  ┌───────▼─────────┐
│   PostgreSQL    │  │    NetBox       │
│   (LCP 库)     │  │  (独立部署)      │
└─────────────────┘  └───────┬─────────┘
                     ┌───────▼─────────┐
                     │   PostgreSQL    │
                     │   (NetBox 库)   │
                     └─────────────────┘
```

### 三种集成深度

**方案一：API 代理（推荐起步）**

```
用户请求 → LCP 认证/鉴权 → /api/cmdb/* → 转发到 NetBox REST API → 返回结果
```

- LCP 负责：统一认证、权限控制、租户隔离
- NetBox 负责：所有 CMDB 数据管理
- 好处：工作量最小，几天上线
- 代码量：一个反向代理 handler + 权限映射

**方案二：数据聚合**

- 应用部署时查 NetBox 的服务器列表
- Dashboard 聚合 NetBox 设备统计 + LCP 应用统计
- 告警时关联 NetBox 设备拓扑定位根因

**方案三：深度集成（后期）**

- NetBox webhook → LCP 事件系统（设备变更触发 LCP 响应）
- LCP 用户/权限同步到 NetBox（统一身份）
- 自定义 NetBox 插件，添加 PaaS 相关字段

### 权限映射

| LCP 概念 | NetBox 概念 |
|----------|------------|
| 工作空间 (Workspace) | Tenant Group |
| 命名空间 (Namespace) | Tenant |
| 用户角色 | NetBox Token 权限 |

### 职责边界

| 关注点 | 负责方 |
|--------|--------|
| 服务器/网络/机柜/IP管理 | NetBox |
| 应用部署/服务编排/容器管理 | LCP |
| 统一认证与权限 | LCP（代理层控制） |
| "应用跑在哪台机器上" | LCP 调 NetBox API 关联 |

### 落地步骤

```
第一步  独立部署 NetBox（Docker 一键启动）
第二步  LCP 加一个 netbox client（调 NetBox REST API）
第三步  LCP 加 /api/cmdb/* 代理路由，统一认证
第四步  前端加 CMDB 页面，消费代理 API
第五步  业务关联（部署选主机、Dashboard 聚合）
```

---

## 待讨论

- NetBox 部署方案（Docker Compose 配置）
- NetBox client Go 封装设计
- /api/cmdb/* 代理路由与权限映射实现
- 前端 CMDB 页面设计
- LCP 与 NetBox 的租户同步机制
- 容器/K8s 到数据中心的关系链设计
