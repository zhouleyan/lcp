# 连通性探测基础能力设计

## 概述

在 `lib/probe/` 提供通用的网络连通性探测能力，作为后续主机部署（SSH 前置检查）和监控接入（o11y 端点验证）的基础库。

纯函数库，无外部依赖，不暴露 REST API，由上层模块按需调用。

## 核心决策

| 项 | 决定 |
|---|---|
| 协议范围 | TCP 端口探测 + HTTP(S) 端点检查 |
| SSH 认证测试 | 不包含，属于业务层（infra 模块通过 `sshclient.Connect()` 自行处理） |
| 包位置 | `lib/probe/`，扁平包，按协议分文件 |
| 外部依赖 | 无，全部使用 Go 标准库 |
| 测试结果 | 成功/失败 + 总耗时 + 失败阶段 + 错误信息 |
| 触发方式 | 由上层决定：手动测试 / 保存时自动测试均可 |
| 执行模式 | 同步调用（批量/异步由上层编排） |

## 类型定义

```go
package probe

// Result 是所有探测的统一返回结构
type Result struct {
    Success    bool          // 探测是否成功
    Duration   time.Duration // 总耗时
    Phase      Phase         // 失败时卡在哪个阶段（成功时为空）
    Message    string        // 失败时的错误描述
    StatusCode int           // HTTP 状态码（仅 HTTP 探测填充）
}

// Phase 标识失败发生的阶段
type Phase string

const (
    PhaseDNS  Phase = "dns"
    PhaseTCP  Phase = "tcp"
    PhaseTLS  Phase = "tls"
    PhaseHTTP Phase = "http"
)

// Options 是探测的公共选项
type Options struct {
    Timeout time.Duration // 超时时间，默认 5s
}
```

## 函数签名

```go
// TCP 探测指定 host:port 是否可达
func TCP(ctx context.Context, addr string, opts *Options) *Result

// HTTP 探测指定 URL 是否可达并返回正常状态码
func HTTP(ctx context.Context, url string, opts *Options) *Result
```

## 行为定义

| | TCP | HTTP |
|---|---|---|
| 成功条件 | TCP 连接建立成功 | 状态码 < 500 |
| 失败阶段判定 | DNS 解析失败 → `dns`，连接超时/拒绝 → `tcp` | 在 TCP 基础上，TLS 握手失败 → `tls`，状态码 >= 500 → `http` |
| 超时默认值 | 5s | 5s |
| HTTPS | N/A | 自动支持（标准库 http.Client 处理 TLS） |
| ctx 取消 | 立即返回失败 | 立即返回失败 |

HTTP 成功条件用 `< 500` 而非 `== 200`，因为连通性测试关注的是"服务在不在"，3xx/4xx 说明服务活着。

## 错误阶段判定

```go
func classifyError(err error) Phase {
    var dnsErr *net.DNSError
    if errors.As(err, &dnsErr) {
        return PhaseDNS
    }

    var opErr *net.OpError
    if errors.As(err, &opErr) {
        return PhaseTCP
    }

    var certErr *tls.CertificateVerificationError
    if errors.As(err, &certErr) {
        return PhaseTLS
    }
    if strings.Contains(err.Error(), "tls:") {
        return PhaseTLS
    }

    return PhaseTCP // 兜底归为 TCP 阶段
}
```

判定顺序与实际网络握手顺序一致：DNS → TCP → TLS。不需要 custom dialer hook，纯粹靠 error 类型判断。

## 文件结构

```
lib/probe/
├── probe.go        # Result, Phase, Options 类型 + classifyError
├── tcp.go          # TCP() 函数
├── http.go         # HTTP() 函数
├── tcp_test.go     # 启动本地 net.Listener 测连通/拒绝/超时
├── http_test.go    # 启动 httptest.Server 测 200/500/TLS/不可达
└── probe_test.go   # classifyError 单元测试
```

## 测试策略

全部使用标准库在本地起临时服务，不依赖外部网络：

- **TCP 测试**：`net.Listen` 启动本地 listener 测连通，关闭 listener 测拒绝，不可达地址测超时
- **HTTP 测试**：`httptest.NewServer` 测 200/500，`httptest.NewTLSServer` 测 HTTPS，不可达地址测连接失败
- **classifyError 测试**：构造各类 `net.DNSError`、`net.OpError` 等验证阶段判定

## 上层消费场景

```
infra 模块（主机 SSH 连通性）：
  1. probe.TCP(ctx, "192.168.1.10:22", opts)  → 端口通？
  2. sshclient.Connect(ctx)                    → 认证过？（业务层）

o11y 模块（监控端点连通性）：
  probe.HTTP(ctx, "http://vm:8428/health", opts)  → 服务可达？
```

连通性探测与 SSH 认证是两层独立能力，`lib/probe/` 和 `lib/clients/sshclient/` 互不依赖。
