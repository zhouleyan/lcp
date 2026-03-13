# SSH ProxyJump（堡垒机跳转）设计

## 概述

为 `lib/clients/sshclient` 添加堡垒机跳转支持，等价于 `ssh -J bastion target`。通过在 `Config` 中添加 `ProxyJump *Config` 字段实现，支持多级链式跳转，调用方无感知。

## 核心决策

| 项 | 决定 |
|---|---|
| 跳转级数 | 多级链式（`ProxyJump *Config` 递归嵌套，零额外代码） |
| 生命周期 | 自动管理：`Connect()` 建立整条链路，`Close()` 级联关闭 |
| 影响范围 | 仅改 `client.go`（Config + Connect + Close），command.go / sftp.go 零改动 |
| 外部依赖 | 无新增，复用现有 `golang.org/x/crypto/ssh` |

## 数据结构变更

### Config

```go
type Config struct {
    Host              string
    Port              int
    User              string
    Password          string
    PrivateKey        string
    PrivateKeyContent string
    Timeout           time.Duration
    ProxyJump         *Config // 堡垒机配置，nil 则直连
}
```

### Client

```go
type Client struct {
    config      Config
    client      *ssh.Client
    proxyClient *Client   // 堡垒机 Client（自动管理）
    proxyConn   net.Conn  // 通过堡垒机建立的隧道连接
    mu          sync.Mutex
}
```

## Connect() 逻辑

```go
func (c *Client) Connect(ctx context.Context) error {
    // ... 现有校验、buildAuthMethods、sshConfig 构建 ...

    if c.config.ProxyJump != nil {
        // 1. 递归连接堡垒机
        proxy := New(*c.config.ProxyJump)
        if err := proxy.Connect(ctx); err != nil {
            return fmt.Errorf("sshclient: failed to connect proxy %s: %w",
                c.config.ProxyJump.Host, err)
        }
        c.proxyClient = proxy

        // 2. 通过堡垒机开 TCP 隧道到目标
        tunnelConn, err := proxy.client.Dial("tcp", addr)
        if err != nil {
            proxy.Close()
            return fmt.Errorf("sshclient: failed to tunnel to %s: %w", addr, err)
        }
        c.proxyConn = tunnelConn

        // 3. 在隧道上建立 SSH 连接
        ncc, chans, reqs, err := ssh.NewClientConn(tunnelConn, addr, sshConfig)
        if err != nil {
            tunnelConn.Close()
            proxy.Close()
            return fmt.Errorf("sshclient: failed to dial %s via proxy: %w", addr, err)
        }
        c.client = ssh.NewClient(ncc, chans, reqs)
    } else {
        // 现有直连逻辑不变
        sshClient, err := ssh.Dial("tcp", addr, sshConfig)
        // ...
    }
}
```

## Close() 逻辑

按反向顺序关闭：目标连接 → 隧道 → 堡垒机（递归）。

```go
func (c *Client) Close() error {
    if c.client != nil {
        c.client.Close()
        c.client = nil
    }
    if c.proxyConn != nil {
        c.proxyConn.Close()
        c.proxyConn = nil
    }
    if c.proxyClient != nil {
        c.proxyClient.Close()
        c.proxyClient = nil
    }
    return nil
}
```

## 使用方式

```go
// 直连（不变）
client := sshclient.New(sshclient.Config{
    Host: "192.168.1.10", Password: "secret",
})

// 通过堡垒机
client := sshclient.New(sshclient.Config{
    Host: "10.0.0.5", Password: "target-pass",
    ProxyJump: &sshclient.Config{
        Host: "bastion.example.com", Password: "bastion-pass",
    },
})

// 多级跳转
client := sshclient.New(sshclient.Config{
    Host: "10.0.0.5", Password: "target-pass",
    ProxyJump: &sshclient.Config{
        Host: "bastion2", Password: "pass2",
        ProxyJump: &sshclient.Config{
            Host: "bastion1", Password: "pass1",
        },
    },
})

// 后续操作完全一致
client.Connect(ctx)
defer client.Close()
result, _ := client.Exec(ctx, "uname -r")
```

## 测试策略

- 现有测试不受影响（ProxyJump 为 nil 走原逻辑）
- 新增测试用 `golang.org/x/crypto/ssh` 启动本地 mock SSH server 模拟堡垒机和目标机
- 测试用例：单级跳转成功、堡垒机连接失败、隧道建立失败、Close 级联清理
