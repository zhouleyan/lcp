# lib/ipam — IP 地址管理库

基于位图的 IP 地址管理库，支持多命名池、多 CIDR、自动/手动分配、Owner 追踪和可插拔持久化。

零外部依赖，仅使用 Go 标准库。所有操作线程安全。

## 架构

```
Manager (命名池管理 + Owner 追踪 + Store 持久化)
  └── managedPool
        ├── CIDRPool (多 CIDR 管理)
        │     └── Range (单 CIDR 分配器)
        │           └── AllocationBitmap (位图追踪)
        └── owners: map[ip]owner
```

## 快速开始

### 纯内存模式

```go
import "lcp.io/lcp/lib/ipam"

// 创建 Manager（NoopStore = 纯内存，不持久化）
mgr := ipam.NewManager(ipam.NoopStore{})

// 创建 IP 池，可包含多个 CIDR
err := mgr.CreatePool("vpc-prod", []string{
    "10.0.1.0/24",   // 254 个可用 IP
    "10.0.2.0/24",   // 254 个可用 IP
})

// 自动分配下一个可用 IP
alloc, err := mgr.AllocateNext("vpc-prod", "host-web-01")
// alloc.IP    = 10.0.1.x
// alloc.CIDR  = "10.0.1.0/24"
// alloc.Owner = "host-web-01"
// alloc.Pool  = "vpc-prod"

// 手动分配指定 IP
alloc, err = mgr.Allocate("vpc-prod", net.ParseIP("10.0.1.100"), "host-db-01")

// 释放 IP
err = mgr.Release("vpc-prod", net.ParseIP("10.0.1.100"))
```

### 带持久化

实现 `ipam.Store` 接口，将池状态和分配记录存入数据库：

```go
type Store interface {
    SavePool(ctx context.Context, state *PoolState) error
    DeletePool(ctx context.Context, name string) error
    LoadPools(ctx context.Context) ([]*PoolState, error)
    SaveAllocation(ctx context.Context, alloc *Allocation) error
    DeleteAllocation(ctx context.Context, pool string, ip net.IP) error
    LoadAllocations(ctx context.Context, pool string) ([]*Allocation, error)
}
```

Manager 在分配/释放成功后同步调用 Store 方法。Store 失败时自动回滚内存操作。

```go
mgr := ipam.NewManager(myPostgresStore)
```

## API 参考

### 池管理

```go
// 创建命名池，cidrs 为 CIDR 字符串列表
mgr.CreatePool(name string, cidrs []string) error

// 删除池（池内有已分配 IP 时返回 ErrPoolInUse）
mgr.DeletePool(name string) error

// 查询池信息
mgr.GetPool(name string) (*PoolInfo, error)

// 列出所有池（按名称排序）
mgr.ListPools() []*PoolInfo

// 动态添加 CIDR 到已有池
mgr.AddCIDR(pool, cidr string) error

// 从池中移除 CIDR（该 CIDR 有已分配 IP 时返回 ErrCIDRInUse）
mgr.RemoveCIDR(pool, cidr string) error
```

### IP 分配

```go
// 手动分配指定 IP
mgr.Allocate(pool string, ip net.IP, owner string) (*Allocation, error)

// 自动分配下一个可用 IP
mgr.AllocateNext(pool, owner string) (*Allocation, error)

// 释放 IP
mgr.Release(pool string, ip net.IP) error
```

### 查询

```go
// 查询单个 IP 的分配信息
mgr.GetAllocation(pool string, ip net.IP) (*Allocation, error)

// 列出池内所有分配（按 IP 排序）
mgr.ListAllocations(pool string) ([]*Allocation, error)

// 按 Owner 查询所有池中的分配
mgr.ListAllocationsByOwner(owner string) []*Allocation
```

### 返回类型

```go
type Allocation struct {
    IP    net.IP `json:"ip"`     // 分配的 IP 地址
    CIDR  string `json:"cidr"`   // 所属 CIDR 段
    Owner string `json:"owner"`  // 持有者标识
    Pool  string `json:"pool"`   // 所属池名称
}

type PoolInfo struct {
    Name     string   `json:"name"`     // 池名称
    CIDRs    []string `json:"cidrs"`    // CIDR 列表
    UsedIPs  int      `json:"usedIPs"`  // 已分配数
    FreeIPs  int      `json:"freeIPs"`  // 可用数
    TotalIPs int      `json:"totalIPs"` // 总数
}
```

## 错误处理

所有错误为 sentinel error，可用 `errors.Is` 判断：

```go
alloc, err := mgr.AllocateNext("vpc-prod", "host-01")
if errors.Is(err, ipam.ErrFull) {
    // 池已满，无可用 IP
}
if errors.Is(err, ipam.ErrPoolNotFound) {
    // 池不存在
}
```

| 错误 | 含义 |
|------|------|
| `ErrFull` | 所有地址已分配 |
| `ErrAllocated` | 地址已被分配 |
| `ErrNotInRange` | 地址不在有效范围内 |
| `ErrNotAllocated` | 地址未被分配（释放/查询时） |
| `ErrPoolNotFound` | 池不存在 |
| `ErrPoolExists` | 池已存在 |
| `ErrPoolInUse` | 池有已分配地址，不能删除 |
| `ErrCIDRExists` | CIDR 已在池中 |
| `ErrCIDRInUse` | CIDR 有已分配地址，不能移除 |

## CIDR 地址计算规则

- `/24` (256 地址) → 排除网络地址和广播地址 → **254 个可用**
- `/30` (4 地址) → 排除网络地址和广播地址 → **2 个可用**
- `/32` (单地址) → 不排除 → **1 个可用**
- IPv6 `/120` (256 地址) → 排除网络地址和广播地址 → **254 个可用**
- IPv6 大于 `/112` 的子网，可用地址数上限为 **65536**

## 直接使用底层组件

Manager 是推荐的入口，但底层组件也可独立使用：

### CIDRPool — 多 CIDR 分配（无 Owner 追踪）

```go
pool := ipam.NewCIDRPool()
_, cidr, _ := net.ParseCIDR("10.0.0.0/24")
pool.AddCIDR(cidr)

ip, _ := pool.AllocateNext()
pool.Release(ip)
fmt.Println(pool.Free(), pool.Used())
```

### Range — 单 CIDR 分配

```go
_, cidr, _ := net.ParseCIDR("10.0.0.0/24")
r, _ := ipam.NewCIDRRange(cidr)

r.Allocate(net.ParseIP("10.0.0.1"))
ip, _ := r.AllocateNext()
r.ForEach(func(ip net.IP) {
    fmt.Println("allocated:", ip)
})
```

### AllocationBitmap — 纯位图

```go
bm := ipam.NewAllocationBitmap(254, "10.0.0.0/24")
bm.Allocate(0)                    // 分配 offset 0
offset, ok := bm.AllocateNext()   // 自动分配
bm.Release(0)                     // 释放
bm.ForEach(func(offset int) {     // 遍历已分配
    fmt.Println(offset)
})

// 快照与恢复
spec, data := bm.Snapshot()
bm2 := ipam.NewAllocationBitmap(254, "10.0.0.0/24")
bm2.Restore(spec, data)
```

## 并发安全

所有层级均为 goroutine 安全：

- `AllocationBitmap` — `sync.Mutex`
- `CIDRPool` — `sync.RWMutex`
- `Manager` — `sync.RWMutex`（池 map）+ 每个池独立 `sync.Mutex`（Owner map）

## 测试

```bash
go test ./lib/ipam/... -v           # 运行所有测试
go test ./lib/ipam/... -v -race     # 带竞态检测
```
