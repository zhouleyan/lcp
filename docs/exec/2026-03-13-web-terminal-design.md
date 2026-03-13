# Web Terminal (Host Exec) 设计文档

日期：2026-03-13

## 概述

在 LCP 平台中实现 Web 终端功能，用户可以通过浏览器对已纳管的主机发起 SSH 连接，进行交互式命令行操作。

## 需求总结

| 项目 | 决策 |
|------|------|
| 使用场景 | 通用 shell 终端，后续可扩展命令限制 |
| SSH 凭据来源 | 当前仅实现用户手动输入，未来结合 PKI 模块 |
| 连接架构 | 代理模式，嵌入 lcp-server（浏览器 ↔ WebSocket ↔ LCP ↔ SSH ↔ 主机） |
| 会话管理 | 先无状态（关闭即断开），预留审计和会话管理扩展点 |
| 前端入口 | 主机列表"终端"按钮 + 独立 exec 页面（`/exec?hostId=xxx`） |
| 终端组件 | xterm.js，内嵌在 LCP 前端 |
| 断线处理 | WebSocket 自动重连（指数退避，最多 3 次），SSH 会话不保持 |
| 连接限制 | 单用户最多 5 个并发会话，空闲 30 分钟自动断开 |
| 凭据存储 | 暂不实现，未来结合 PKI 模块 |
| API 命名 | exec（参考 K8s pod exec 语义） |

## 整体架构

```
浏览器 (xterm.js)
  │
  │  WebSocket (wss://host/api/infra/v1/hosts/{hostId}/exec)
  │  Auth: Bearer token (复用现有认证)
  │
  ▼
lcp-server
  │
  ├─ Filter Chain (requestInfo → auth → authz)
  │   RBAC 权限码: infra:hosts:exec
  │
  ├─ ExecHandler:
  │   1. WebSocket upgrade (lib/websocket/)
  │   2. 接收 connect 消息（SSH 凭据 + 终端尺寸）
  │   3. 建立 SSH 连接 (复用 lib/clients/sshclient/)
  │   4. 分配 PTY (ssh.Session.RequestPty)
  │   5. 双向桥接: WebSocket ↔ SSH stdin/stdout
  │   6. 监听: 断开 / 空闲超时 / 窗口 resize
  │
  ▼
目标主机 (SSH)
```

## 分层架构

```
lib/rest/websocket.go          WebSocketHandler 类型 + HandleWebSocket 包装器
                                框架层，和 HandlerFunc 平级

lib/websocket/                  通用 WebSocket 基础设施（封装 github.com/coder/websocket）
  ├── conn.go                   Conn 接口封装（对外暴露，隔离第三方依赖）
  ├── message.go                消息协议（类型前缀常量、读写辅助函数）
  ├── session.go                SessionManager（并发限制、空闲超时、会话追踪）
  └── upgrader.go               HTTP → WebSocket 升级

pkg/apis/infra/exec.go          SSH PTY 桥接业务逻辑（只依赖 lib/websocket.Conn）
```

第三方依赖 `github.com/coder/websocket` 封装在 `lib/websocket/` 内部，其他模块零感知。

选择 `coder/websocket` 的理由：
- io.Reader/Writer 接口 — SSH stdin/stdout 桥接可直接 io.Copy
- Context 原生支持 — 和 LCP context 体系自然衔接
- Coder 自身就做 Web 终端产品，生产验证过
- 标准 net/http 模式 — 和现有 handler chain 无缝配合

## REST 框架扩展

### WebSocketHandler 类型

```go
// lib/rest/websocket.go

// WebSocketHandler 处理已升级的 WebSocket 连接。
// 框架负责升级和错误处理，handler 只关注业务逻辑。
type WebSocketHandler func(ctx context.Context, params map[string]string, conn *websocket.Conn)
```

### ActionInfo 扩展

```go
type ActionInfo struct {
    Name             string
    Method           string
    StatusCode       int
    Handler          HandlerFunc        // JSON 请求/响应（互斥）
    WebSocketHandler WebSocketHandler   // WebSocket 连接（互斥）
}
```

### HandleWebSocket 包装器

框架内部提供，负责升级 + 参数解析，handler 不碰 HTTP 细节：

```go
func HandleWebSocket(handler WebSocketHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        conn, err := websocket.Accept(w, req, nil)  // lib/websocket 封装
        if err != nil {
            return
        }
        defer conn.Close()
        params := PathParams(req)
        mergeQueryParams(params, req)
        handler(req.Context(), params, conn)
    }
}
```

## WebSocket 消息协议

用第一个字节区分消息类型，参考 K8s exec channel ID 前缀风格：

```
消息格式: [1 byte type] [payload]

类型定义:
  0x00 = stdin/stdout 数据 (二进制，payload 即原始终端数据)
  0x01 = resize         (JSON payload: {"cols": 120, "rows": 40})
  0x02 = connect        (JSON payload: 首条消息，携带连接参数)
  0x03 = status         (JSON payload: 服务端→客户端状态通知)
```

### 连接流程

```
1. 客户端发起 WebSocket 连接
   GET /api/infra/v1/hosts/{hostId}/exec
   Header: Authorization: Bearer <token>

2. 连接建立后，客户端发送 connect 消息 (0x02):
   {
     "cols": 120,
     "rows": 40,
     "user": "root",
     "password": "xxx",
     "privateKey": "-----BEGIN..."   // 可选，与 password 二选一
     "port": 22                      // 可选，默认 22
   }

3. 服务端建立 SSH 连接，成功后回复 status (0x03):
   {"status": "connected", "message": "Connected to 10.0.1.5"}

4. 双向数据传输 (0x00):
   客户端 → 服务端: 键盘输入
   服务端 → 客户端: 终端输出

5. 窗口 resize 时客户端发送 (0x01):
   {"cols": 150, "rows": 50}

6. 断开时服务端发送 status (0x03):
   {"status": "closed", "message": "Session closed: idle timeout"}
```

## SSH PTY 桥接

```go
// pkg/apis/infra/exec.go

func NewExecHandler(sessionMgr *websocket.SessionManager, hostStore HostStore) rest.WebSocketHandler {
    return func(ctx context.Context, params map[string]string, conn *websocket.Conn) {
        // 1. 等待 connect 消息 (0x02)
        // 2. 查询 Host 记录，拿到 IP
        // 3. SessionManager.Acquire（并发限制）
        // 4. 建立 SSH 连接 (sshclient.New + Connect)
        // 5. 分配 PTY (RequestPty "xterm-256color")
        // 6. 双向桥接（两个 goroutine）:
        //      WebSocket → SSH stdin (0x00 数据 + 0x01 resize)
        //      SSH stdout → WebSocket (0x00 数据)
        // 7. 任一方断开 → cancel context → 清理资源
    }
}
```

### 双向桥接

```go
// WebSocket → SSH stdin
go func() {
    for {
        msgType, payload := conn.Read()
        switch msgType {
        case 0x00: stdin.Write(payload)         // 终端输入
        case 0x01: session.WindowChange(r, c)   // 窗口 resize
        }
        idleTimer.Reset(30 * time.Minute)
    }
}()

// SSH stdout → WebSocket
go func() {
    buf := make([]byte, 4096)
    for {
        n := stdout.Read(buf)
        conn.Write(append([]byte{0x00}, buf[:n]...))
        idleTimer.Reset(30 * time.Minute)
    }
}()
```

### 资源清理顺序

```
任一方断开 → cancel context
  → SSH session.Close()
  → SSH client.Close()（含 ProxyJump 级联清理）
  → WebSocket conn.Close()
  → SessionManager.Release()
```

## SessionManager（通用）

```go
// lib/websocket/session.go

type SessionManager struct {
    mu          sync.Mutex
    sessions    map[string][]*Session  // key: userID
    maxPerUser  int                    // 默认 5
    idleTimeout time.Duration          // 默认 30m
}

type Session struct {
    ID         string
    UserID     string
    Resource   string     // "host", "container", "database" ...
    ResourceID string
    Label      string     // 显示用，如 "root@10.0.1.5"
    CreatedAt  time.Time
    cancel     context.CancelFunc  // 预留：外部强制断开
}

// Acquire — 连接前调用，超限返回错误
func (m *SessionManager) Acquire(userID, resource, resourceID, label string, cancel context.CancelFunc) (*Session, error)

// Release — 连接断开时调用
func (m *SessionManager) Release(sessionID string)

// Count — 获取用户当前会话数（预留 API 查询）
func (m *SessionManager) Count(userID string) int
```

## 前端集成

### 技术选型

- `xterm.js` — 终端渲染
- `xterm-addon-fit` — 自适应容器尺寸
- `xterm-addon-web-links` — URL 可点击

### 核心组件

```
WebTerminal (React 组件)
  ├── xterm.js 实例（渲染终端）
  ├── WebSocket 客户端（连接管理 + 消息协议）
  └── 连接表单（用户名、密码/密钥、端口）
```

### UI 入口

- 主机列表页 — 操作列增加"终端"按钮
- 独立 exec 页面 — `/exec?hostId=xxx`

### 自动重连

- WebSocket 异常断开时，延迟 1s/2s/4s 指数退避重连
- 重连后需重新发送 connect 消息（SSH 会话已丢失）
- 最多重试 3 次，之后提示用户手动重连

## 变更范围

| 变更 | 文件/目录 |
|------|----------|
| WebSocketHandler 框架支持 | `lib/rest/websocket.go`, `lib/rest/types.go`, `lib/rest/installer.go` |
| 通用 WebSocket 基础设施 | `lib/websocket/`（新包，封装 `coder/websocket`） |
| SSH PTY 桥接 | `pkg/apis/infra/exec.go` |
| 路由注册 | `pkg/apis/infra/v1/install.go` |
| 依赖引入 | `github.com/coder/websocket`（封装在 lib/websocket 内部） |
| 前端终端组件 | `ui/src/components/WebTerminal/` |
| 前端路由 + 入口 | 主机列表页 + `/exec` 页面 |

## 多实例部署约束

当前 `SessionManager` 是纯内存实现，横向扩容时存在以下约束：

| 约束 | 影响 | 严重程度 |
|------|------|---------|
| 并发限制失效 | 用户可在 N 个实例各开 maxPerUser 个会话 | 中 |
| 会话列表不全 | List/Count 只返回本实例的会话 | 低 |
| 强制断开受限 | Cancel 只能断开本实例上的会话 | 低 |

**WebSocket 连接本身不受影响** — 每个连接自包含（WebSocket ↔ SSH 桥接在同一进程内），断线重连本来就要新建 SSH session。

### 解决方案

| 方案 | 做法 | 复杂度 | 适用阶段 |
|------|------|--------|---------|
| 粘性路由 | 负载均衡器按 userID/token hash 做 sticky session | 低 | 初期 |
| 共享存储 | SessionManager 后端改用 Redis 或 PostgreSQL | 中 | 规模化 |

推荐先粘性路由后共享存储。`SessionManager` 接口已抽象好（Acquire/Release/Count/List），后续替换为 Redis 实现只需改 `NewSessionManager`，业务代码不动。

## 后续扩展（不在本期实现）

- host_credentials 存储 + PKI 模块集成
- 会话审计日志（操作录制/回放）
- 管理员查看/强制断开活跃会话
- 命令白名单/黑名单限制
- 容器 exec、数据库 console 等复用 WebSocket 基础设施
- 多实例部署：SessionManager 共享存储
