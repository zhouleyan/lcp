# probe

通用网络连通性探测库，支持 TCP 端口探测和 HTTP(S) 端点检查。

纯 Go 标准库实现，无外部依赖。

## 使用方式

### TCP 探测

检测 `host:port` 是否可达：

```go
import "lcp.io/lcp/lib/probe"

// 默认超时 5s
result := probe.TCP(ctx, "192.168.1.10:22", nil)

// 自定义超时
result := probe.TCP(ctx, "192.168.1.10:22", &probe.Options{
    Timeout: 3 * time.Second,
})

if result.Success {
    fmt.Printf("连通，耗时 %v\n", result.Duration)
} else {
    fmt.Printf("失败，阶段: %s，原因: %s\n", result.Phase, result.Message)
}
```

### HTTP 探测

检测 URL 是否可达（状态码 < 500 视为成功）：

```go
result := probe.HTTP(ctx, "http://vm:8428/health", nil)

if result.Success {
    fmt.Printf("可达，状态码: %d，耗时 %v\n", result.StatusCode, result.Duration)
} else {
    fmt.Printf("失败，阶段: %s，原因: %s\n", result.Phase, result.Message)
}
```

HTTPS 自动支持，跳过证书验证（连通性测试，非安全校验）。不跟随重定向，3xx 视为可达。

## 返回结果

```go
type Result struct {
    Success    bool          // 是否成功
    Duration   time.Duration // 总耗时
    Phase      Phase         // 失败阶段（成功时为空）
    Message    string        // 失败原因
    StatusCode int           // HTTP 状态码（仅 HTTP 探测）
}
```

## 失败阶段

失败时 `Phase` 标识卡在哪一步：

| Phase | 含义 | 典型原因 |
|-------|------|---------|
| `dns` | DNS 解析失败 | 域名不存在、DNS 不可达 |
| `tcp` | TCP 连接失败 | 端口未开放、超时、网络不可达 |
| `tls` | TLS 握手失败 | 证书错误、协议不匹配 |
| `http` | HTTP 服务端错误 | 状态码 >= 500 |

## 典型场景

```go
// 主机 SSH 前置检查：先测端口，再走 SSH 认证
result := probe.TCP(ctx, host+":22", nil)
if !result.Success {
    return fmt.Errorf("SSH 端口不通: %s", result.Message)
}
// 端口通了，再调 sshclient.Connect() ...

// 监控端点验证：检测 VictoriaMetrics 是否可达
result := probe.HTTP(ctx, metricsURL+"/health", nil)
if !result.Success {
    return fmt.Errorf("监控端点不可达: %s", result.Message)
}
```
