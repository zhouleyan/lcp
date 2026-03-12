# LCP 业务测试用例

## 目录

- [1. OIDC 认证流程](#1-oidc-认证流程)
- [2. 用户管理](#2-用户管理)
- [3. 工作空间管理](#3-工作空间管理)
- [4. 项目（Namespace）管理](#4-项目namespace管理)
- [5. 成员管理](#5-成员管理)
- [6. 角色管理](#6-角色管理)
- [7. 角色绑定管理](#7-角色绑定管理)
- [8. 权限管理](#8-权限管理)
- [9. RBAC 授权](#9-rbac-授权)
- [10. Dashboard 统计](#10-dashboard-统计)
- [11. 主机（Host）管理](#11-主机host管理)
- [12. 环境（Environment）管理](#12-环境environment管理)
- [13. 主机-环境绑定](#13-主机-环境绑定)
- [14. 主机分配（Host Assignment）](#14-主机分配host-assignment)
- [15. Hosts/Env 多层级权限控制](#15-hostsenv-多层级权限控制)

---

## 1. OIDC 认证流程

### 1.1 OIDC Discovery

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| OIDC-001 | 获取 OIDC 发现文档 | 服务正常运行 | GET `/.well-known/openid-configuration` | 200，返回 issuer、endpoints、supported scopes、grant types、algorithms 等完整配置 |
| OIDC-002 | 获取 JWKS | 服务正常运行 | GET `/.well-known/jwks.json` | 200，返回 JSON Web Key Set，包含签名公钥 |

### 1.2 授权请求

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| OIDC-010 | 正常发起授权请求 | 已注册 client `lcp-ui` | GET `/oidc/authorize?response_type=code&client_id=lcp-ui&redirect_uri=http://localhost:5173/auth/callback&scope=openid profile email&state=xyz&code_challenge=abc&code_challenge_method=S256` | 302 重定向到登录页，URL 包含 `request_id` 参数 |
| OIDC-011 | 缺少 response_type 参数 | - | GET `/oidc/authorize?client_id=lcp-ui` | 400，错误提示 response_type 必须为 "code" |
| OIDC-012 | 无效的 client_id | - | GET `/oidc/authorize?response_type=code&client_id=unknown` | 400，客户端未注册 |
| OIDC-013 | redirect_uri 不匹配 | 已注册 client `lcp-ui` | GET `/oidc/authorize?response_type=code&client_id=lcp-ui&redirect_uri=http://evil.com/callback` | 400，redirect_uri 不在注册列表中 |
| OIDC-014 | 公开客户端缺少 PKCE | client `lcp-ui` 为 public 类型 | GET `/oidc/authorize?response_type=code&client_id=lcp-ui&redirect_uri=...`（不带 code_challenge） | 400，公开客户端必须提供 code_challenge |
| OIDC-015 | 不支持的 code_challenge_method | - | GET `/oidc/authorize?...&code_challenge=abc&code_challenge_method=plain` | 400，仅支持 S256 |

### 1.3 用户登录

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| OIDC-020 | 使用用户名密码登录（带 requestId） | 用户 admin 已创建且状态 active，已发起授权请求获得 requestId | POST `/oidc/login` body: `{username: "admin", password: "correct", requestId: "xxx"}` | 200，返回 redirectUri 包含 authorization code 和 state |
| OIDC-021 | 使用邮箱登录 | 用户已创建，邮箱 admin@example.com | POST `/oidc/login` body: `{username: "admin@example.com", password: "correct", requestId: "xxx"}` | 200，正常返回（支持邮箱作为登录标识） |
| OIDC-022 | 密码错误 | 用户 admin 已创建 | POST `/oidc/login` body: `{username: "admin", password: "wrong"}` | 401，认证失败 |
| OIDC-023 | 用户不存在 | - | POST `/oidc/login` body: `{username: "nonexist", password: "any"}` | 401，认证失败 |
| OIDC-024 | 用户状态为 inactive | 用户 admin 状态设为 inactive | POST `/oidc/login` body: `{username: "admin", password: "correct"}` | 401，用户已停用 |
| OIDC-025 | 不带 requestId 直接登录 | 用户 admin 已创建且 active | POST `/oidc/login` body: `{username: "admin", password: "correct"}` | 200，返回 sessionId 和 userId（直接登录模式） |
| OIDC-026 | 登录成功更新 lastLogin | 用户 admin 已创建 | 登录前记录 lastLogin，执行登录，再查询用户 | lastLogin 时间戳已更新 |

### 1.4 Token 交换

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| OIDC-030 | authorization_code 换取 token | 已通过登录获得 auth code | POST `/oidc/token` form: `grant_type=authorization_code&code=xxx&client_id=lcp-ui&code_verifier=yyy` | 200，返回 `{access_token, id_token, refresh_token, token_type: "Bearer", expires_in, scope}` |
| OIDC-031 | auth code 已过期 | auth code 超过 5 分钟 | POST `/oidc/token` 使用过期 code | 400，authorization code 已过期 |
| OIDC-032 | auth code 已使用（重放攻击） | auth code 已成功交换一次 | 再次使用同一 code 调用 `/oidc/token` | 400，authorization code 已被使用 |
| OIDC-033 | PKCE 验证失败 | 已获得 auth code | POST `/oidc/token` form: code_verifier 与原始 code_challenge 不匹配 | 400，PKCE 验证失败 |
| OIDC-034 | client_id 不匹配 | auth code 由 client_id=lcp-ui 生成 | POST `/oidc/token` form: `client_id=other-client&code=xxx` | 400，client_id 不匹配 |
| OIDC-035 | refresh_token 换取新 token | 已有有效 refresh_token | POST `/oidc/token` form: `grant_type=refresh_token&refresh_token=xxx&client_id=lcp-ui` | 200，返回新的 token 组（旧 refresh_token 作废，发放新 refresh_token） |
| OIDC-036 | refresh_token 过期 | refresh_token 已超过 168h | POST `/oidc/token` form: `grant_type=refresh_token&refresh_token=expired` | 400，refresh token 已过期 |
| OIDC-037 | inactive 用户刷新 token | 用户在 token 有效期内被设为 inactive | POST `/oidc/token` form: `grant_type=refresh_token&refresh_token=xxx` | 401，用户已停用，拒绝刷新 |

### 1.5 UserInfo

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| OIDC-040 | 获取用户信息 | 已获取有效 access_token（scope 含 openid profile email phone） | GET `/oidc/userinfo` Header: `Authorization: Bearer {token}` | 200，返回 `{sub, name, email, phone_number}` |
| OIDC-041 | scope 限制返回字段 | access_token scope 仅含 openid | GET `/oidc/userinfo` | 200，仅返回 sub 字段，无 name/email/phone |
| OIDC-042 | 无效 token 访问 userinfo | token 已过期或伪造 | GET `/oidc/userinfo` Header: `Authorization: Bearer invalid` | 401 |
| OIDC-043 | 用户已删除后访问 userinfo | 用户在 token 颁发后被删除 | GET `/oidc/userinfo` | 401，用户不存在 |

---

## 2. 用户管理

### 2.1 创建用户

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| USER-001 | 正常创建用户 | 已认证，有权限 | POST `/api/iam/v1/users` body: `{spec: {username: "testuser", email: "test@example.com", phone: "13800138000", password: "MyPass123"}}` | 201，返回用户对象含 ID、timestamps |
| USER-002 | 创建用户不带密码 | - | POST `/api/iam/v1/users` body: `{spec: {username: "nopwd", email: "nopwd@example.com", phone: "13800138001"}}` | 201，密码字段为空（允许无密码创建） |
| USER-003 | 用户名过短 | - | `username: "ab"` (2 个字符) | 400，用户名长度不符合要求（3-50 位） |
| USER-004 | 用户名包含特殊字符 | - | `username: "test-user"` (含连字符) | 400，用户名仅允许字母、数字和下划线 |
| USER-005 | 用户名过长 | - | `username: "a" * 51` | 400，用户名超过最大长度 |
| USER-006 | 邮箱格式无效 | - | `email: "not-an-email"` | 400，邮箱格式无效 |
| USER-007 | 手机号格式无效 | - | `phone: "12345"` | 400，手机号格式无效 |
| USER-008 | 手机号非法前缀 | - | `phone: "10800138000"` | 400，手机号格式无效（需以 1[3-9] 开头） |
| USER-009 | 密码过短 | - | `password: "Ab1"` | 400，密码长度至少 8 位 |
| USER-010 | 密码缺少大写字母 | - | `password: "mypass123"` | 400，密码须包含大写字母 |
| USER-011 | 密码缺少小写字母 | - | `password: "MYPASS123"` | 400，密码须包含小写字母 |
| USER-012 | 密码缺少数字 | - | `password: "MyPassword"` | 400，密码须包含数字 |
| USER-013 | 用户名重复 | 已存在 username=testuser | 创建 `username: "testuser"` | 409，用户名已存在 |
| USER-014 | 邮箱重复 | 已存在 email=test@example.com | 创建 `email: "test@example.com"` | 409，邮箱已存在 |
| USER-015 | 手机号重复 | 已存在 phone=13800138000 | 创建 `phone: "13800138000"` | 409，手机号已存在 |
| USER-016 | DryRun 模式创建 | - | POST `/api/iam/v1/users?dryRun=true` body: 正常数据 | 200，返回验证结果但不实际写入数据库 |
| USER-017 | 设置用户状态 | - | `status: "active"` | 201，用户状态为 active |
| USER-018 | 无效状态值 | - | `status: "deleted"` | 400，状态仅允许 active 或 inactive |

### 2.2 查询用户

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| USER-020 | 获取用户详情 | 用户 ID=1 存在 | GET `/api/iam/v1/users/1` | 200，返回用户信息 |
| USER-021 | 用户不存在 | - | GET `/api/iam/v1/users/99999` | 404 |
| USER-022 | 用户列表（默认分页） | 存在多个用户 | GET `/api/iam/v1/users` | 200，返回 UserList，默认 page=1, pageSize=20 |
| USER-023 | 用户列表自定义分页 | 存在 50 个用户 | GET `/api/iam/v1/users?page=2&pageSize=10` | 200，返回第 2 页 10 条数据，totalCount=50 |
| USER-024 | 用户列表排序 | 存在多个用户 | GET `/api/iam/v1/users?sortBy=username&sortOrder=asc` | 200，按用户名升序排列 |
| USER-025 | pageSize 超过最大值 | - | GET `/api/iam/v1/users?pageSize=200` | 200，自动限制为 pageSize=100 |

### 2.3 更新用户

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| USER-030 | 全量更新用户 | 用户已存在 | PUT `/api/iam/v1/users/{id}` body: 完整 spec | 200，返回更新后的用户 |
| USER-031 | 部分更新用户 | 用户已存在 | PATCH `/api/iam/v1/users/{id}` body: `{spec: {displayName: "新名称"}}` | 200，仅更新指定字段 |
| USER-032 | 更新用户邮箱为已存在值 | 另一用户使用该邮箱 | PUT 更新 email 为重复值 | 409 |

### 2.4 删除用户

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| USER-040 | 删除单个用户 | 用户存在 | DELETE `/api/iam/v1/users/{id}` | 200 |
| USER-041 | 删除不存在的用户 | - | DELETE `/api/iam/v1/users/99999` | 404 |
| USER-042 | 批量删除用户 | 多个用户存在 | DELETE `/api/iam/v1/users` body: `{ids: ["1", "2", "3"]}` | 200，返回 `{successCount, failedCount}` |

### 2.5 修改密码

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| USER-050 | 正常修改密码 | 用户已登录 | POST `/api/iam/v1/users/{id}/change-password` body: `{oldPassword: "OldPass123", newPassword: "NewPass456"}` | 200，密码修改成功 |
| USER-051 | 旧密码错误 | - | `oldPassword: "wrong"` | 400，旧密码不正确 |
| USER-052 | 新密码格式不合规 | - | `newPassword: "short"` | 400，新密码不符合格式要求 |
| USER-053 | 修改密码后旧 refresh_token 失效 | 用户有活跃的 refresh_token | 修改密码后，使用旧 refresh_token 刷新 | 400，refresh_token 已被撤销 |
| USER-054 | 用户自行修改自己的密码 | 以普通用户身份登录 | POST `/api/iam/v1/users/{self}/change-password` | 200，允许（自访问权限） |

---

## 3. 工作空间管理

### 3.1 创建工作空间

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| WS-001 | 正常创建工作空间 | 已认证用户 | POST `/api/iam/v1/workspaces` body: `{spec: {name: "my-workspace"}}` | 201，返回工作空间信息，含 memberCount、namespaceCount |
| WS-002 | 创建工作空间自动注入 ownerId | 已认证用户（ID=1），不传 ownerId | POST 不含 ownerId | 201，ownerId 自动设为当前用户 ID |
| WS-003 | 创建后自动生成默认 Namespace | - | 创建工作空间后查询其 namespaces | 存在一个默认 namespace |
| WS-004 | 创建后自动生成内置角色 | - | 创建工作空间后查询其 roles | 存在 workspace-admin、workspace-viewer、workspace-member 等内置角色 |
| WS-005 | 创建后 owner 自动成为成员 | - | 创建工作空间后查询成员列表 | owner 在成员列表中，且拥有 workspace-admin 角色 |
| WS-006 | 名称过短 | - | `name: "ab"` | 400 |
| WS-007 | 名称包含大写字母 | - | `name: "MyWorkspace"` | 400，仅允许小写字母、数字和连字符 |
| WS-008 | 名称以连字符开头 | - | `name: "-workspace"` | 400 |
| WS-009 | 名称以连字符结尾 | - | `name: "workspace-"` | 400 |
| WS-010 | 指定的 ownerId 用户不存在 | - | `ownerId: "99999"` | 400，用户不存在 |
| WS-011 | 工作空间名称重复 | 已存在 name=my-workspace | 创建同名工作空间 | 409 |

### 3.2 查询工作空间

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| WS-020 | 获取工作空间详情 | 工作空间存在 | GET `/api/iam/v1/workspaces/{id}` | 200，返回含 owner 用户名、统计数据 |
| WS-021 | 工作空间不存在 | - | GET `/api/iam/v1/workspaces/99999` | 404 |
| WS-022 | 平台管理员列出所有工作空间 | 以 platform-admin 身份 | GET `/api/iam/v1/workspaces` | 200，返回所有工作空间 |
| WS-023 | 普通用户列出工作空间（访问过滤） | 用户仅是 ws-1 和 ws-2 的成员 | GET `/api/iam/v1/workspaces` | 200，仅返回 ws-1 和 ws-2 |

### 3.3 更新工作空间

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| WS-030 | 更新工作空间信息 | 工作空间存在 | PUT `/api/iam/v1/workspaces/{id}` body: `{spec: {displayName: "新名称", description: "新描述"}}` | 200 |
| WS-031 | 部分更新工作空间 | - | PATCH `/api/iam/v1/workspaces/{id}` body: `{spec: {status: "inactive"}}` | 200 |
| WS-032 | 更新 ownerId 为不存在的用户 | - | PUT 更新 `ownerId: "99999"` | 400 |

### 3.4 删除工作空间

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| WS-040 | 删除无子 Namespace 的工作空间 | 工作空间下无 namespace（默认 namespace 已删除） | DELETE `/api/iam/v1/workspaces/{id}` | 200 |
| WS-041 | 删除有子 Namespace 的工作空间 | 工作空间下有 namespace | DELETE `/api/iam/v1/workspaces/{id}` | 409，须先删除所有子 namespace |
| WS-042 | 删除后角色绑定级联清理 | 工作空间有角色绑定 | 删除工作空间后查询该工作空间的角色绑定 | 角色绑定已删除 |
| WS-043 | 删除后权限缓存失效 | 用户有该工作空间权限缓存 | 删除工作空间后，用户权限缓存已被清除 | 后续权限检查不再包含该工作空间权限 |
| WS-044 | 批量删除工作空间 | 多个工作空间存在 | DELETE `/api/iam/v1/workspaces` body: `{ids: ["1", "2"]}` | 200，返回成功/失败计数 |
| WS-045 | 批量删除中部分有子 Namespace | ws-1 无子 ns，ws-2 有子 ns | DELETE body: `{ids: ["ws1-id", "ws2-id"]}` | 200，ws-1 删除成功，ws-2 失败，failedCount=1 |

### 3.5 转移工作空间所有权

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| WS-050 | 正常转移所有权 | 当前 owner 操作，新 owner 已是成员 | POST `/api/iam/v1/workspaces/{id}/transfer-ownership` body: `{newOwnerUserId: "2"}` | 200，ownerId 已更新 |
| WS-051 | 新 owner 不是工作空间成员 | 目标用户未加入该工作空间 | POST transfer-ownership | 400，新 owner 必须是工作空间成员 |
| WS-052 | 非 owner 非管理员尝试转移 | 以普通成员身份操作 | POST transfer-ownership | 403 |
| WS-053 | 转移后原 owner 失去 isOwner 标记 | - | 转移后查询原 owner 的角色绑定 | isOwner=false |
| WS-054 | 转移后新 owner 获得 isOwner 标记 | - | 转移后查询新 owner 的角色绑定 | isOwner=true |
| WS-055 | 转移后权限缓存失效 | - | 转移后立即检查新老 owner 权限 | 双方缓存均已刷新 |

---

## 4. 项目（Namespace）管理

### 4.1 创建 Namespace

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| NS-001 | 在工作空间下创建 Namespace | 工作空间存在 | POST `/api/iam/v1/workspaces/{wsId}/namespaces` body: `{spec: {name: "my-project"}}` | 201，返回 namespace 信息 |
| NS-002 | 创建后自动生成内置角色 | - | 创建后查询 namespace roles | 存在 namespace-admin、namespace-viewer 等内置角色 |
| NS-003 | 创建后 owner 自动成为成员 | - | 查询成员列表 | owner 在列表中且有 namespace-admin 角色 |
| NS-004 | 创建后 owner 自动加入父工作空间 | owner 未加入该工作空间 | 创建 namespace 后查询工作空间成员 | owner 已自动成为工作空间成员 |
| NS-005 | 名称验证规则同工作空间 | - | `name: "AB"` / `name: "-ns"` / `name: "n"` | 400（同工作空间名称验证规则） |
| NS-006 | 指定不存在的工作空间 | - | POST 到不存在的 workspaceId 路径 | 400/404 |
| NS-007 | 指定不存在的 owner | - | `ownerId: "99999"` | 400 |
| NS-008 | 设置 visibility | - | `visibility: "private"` | 201，visibility 为 private |
| NS-009 | 无效 visibility 值 | - | `visibility: "secret"` | 400 |
| NS-010 | 设置 maxMembers | - | `maxMembers: 10` | 201，maxMembers=10 |
| NS-011 | maxMembers 为负数 | - | `maxMembers: -1` | 400 |
| NS-012 | 通过顶级路径创建 | - | POST `/api/iam/v1/namespaces` body 含 workspaceId | 201 |

### 4.2 查询 Namespace

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| NS-020 | 获取 Namespace 详情 | namespace 存在 | GET `/api/iam/v1/namespaces/{id}` | 200，含 owner 和 workspace 信息 |
| NS-021 | 通过工作空间路径获取 | - | GET `/api/iam/v1/workspaces/{wsId}/namespaces/{nsId}` | 200 |
| NS-022 | 列出工作空间下所有 Namespace | 工作空间下有多个 namespace | GET `/api/iam/v1/workspaces/{wsId}/namespaces` | 200，仅返回该工作空间下的 namespace |
| NS-023 | 普通用户列表访问过滤 | 用户仅是 ns-1 的成员 | GET `/api/iam/v1/namespaces` | 200，仅返回用户有权限的 namespace |

### 4.3 更新 Namespace

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| NS-030 | 更新 Namespace 信息 | namespace 存在 | PUT `/api/iam/v1/namespaces/{id}` body: `{spec: {displayName: "新项目"}}` | 200 |
| NS-031 | 部分更新 | - | PATCH 更新 description 字段 | 200 |

### 4.4 删除 Namespace

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| NS-040 | 删除无成员的 Namespace | namespace 无成员用户 | DELETE `/api/iam/v1/namespaces/{id}` | 200 |
| NS-041 | 删除有成员的 Namespace | namespace 有成员用户 | DELETE `/api/iam/v1/namespaces/{id}` | 409，须先移除所有成员 |
| NS-042 | 批量删除 | 多个 namespace 存在 | DELETE `/api/iam/v1/namespaces` body: `{ids: [...]}` | 200，返回成功/失败计数 |
| NS-043 | 删除后角色绑定级联清理 | namespace 有角色绑定 | 删除后查询 | 相关角色绑定已清理 |

### 4.5 转移 Namespace 所有权

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| NS-050 | 正常转移 Namespace 所有权 | 新 owner 是 namespace 成员 | POST `/api/iam/v1/workspaces/{wsId}/namespaces/{nsId}/transfer-ownership` body: `{newOwnerUserId: "2"}` | 200 |
| NS-051 | 新 owner 非 namespace 成员 | - | POST transfer-ownership | 400 |

---

## 5. 成员管理

### 5.1 工作空间成员

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| MBR-001 | 列出工作空间成员 | 工作空间有 3 个成员 | GET `/api/iam/v1/workspaces/{wsId}/users` | 200，返回 3 个成员及其角色信息 |
| MBR-002 | 批量添加工作空间成员 | 用户 2、3 存在但未加入 | POST `/api/iam/v1/workspaces/{wsId}/users` body: `{ids: ["2", "3"]}` | 200，成功添加 2 个成员 |
| MBR-003 | 添加成员指定角色 | 存在自定义角色 | POST body: `{ids: ["2"], roleId: "custom-role-id"}` | 200，成员以指定角色加入 |
| MBR-004 | 添加成员不指定角色 | - | POST body: `{ids: ["2"]}` | 200，使用默认 workspace-viewer 角色 |
| MBR-005 | 添加不存在的用户 | 用户 99999 不存在 | POST body: `{ids: ["99999"]}` | 400，用户不存在 |
| MBR-006 | 批量移除工作空间成员 | 用户 2、3 是成员 | DELETE `/api/iam/v1/workspaces/{wsId}/users` body: `{ids: ["2", "3"]}` | 200，成员已移除 |
| MBR-007 | 移除后角色绑定清理 | - | 移除成员后查询该用户在该工作空间的角色绑定 | 角色绑定已删除 |

### 5.2 Namespace 成员

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| MBR-010 | 列出 Namespace 成员 | namespace 有成员 | GET `/api/iam/v1/workspaces/{wsId}/namespaces/{nsId}/users` | 200，返回成员列表 |
| MBR-011 | 批量添加 Namespace 成员 | 用户已存在 | POST `/api/iam/v1/workspaces/{wsId}/namespaces/{nsId}/users` body: `{ids: ["2", "3"]}` | 200 |
| MBR-012 | 添加成员自动加入父工作空间 | 用户 2 未加入工作空间 | 将用户 2 添加到 namespace | 用户 2 自动成为工作空间成员 |
| MBR-013 | 超过 maxMembers 限制 | namespace maxMembers=2，已有 2 个成员 | 添加第 3 个成员 | 400，成员数量超过限制 |
| MBR-014 | maxMembers=0 表示不限制 | namespace maxMembers=0 | 添加任意数量成员 | 200，无限制 |
| MBR-015 | 批量移除 Namespace 成员 | 成员存在 | DELETE body: `{ids: ["2", "3"]}` | 200 |

### 5.3 用户视角查询

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| MBR-020 | 查询用户关联的工作空间 | 用户加入了 2 个工作空间 | GET `/api/iam/v1/users/{userId}:workspaces` | 200，返回 2 个工作空间，含 role、joinedAt |
| MBR-021 | 查询用户关联的 Namespace | 用户加入了 3 个 namespace | GET `/api/iam/v1/users/{userId}:namespaces` | 200，返回 3 个 namespace，含角色信息 |

---

## 6. 角色管理

### 6.1 平台级角色

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ROLE-001 | 列出平台角色 | 系统启动后已有内置角色 | GET `/api/iam/v1/roles` | 200，包含 platform-admin、platform-viewer 等内置角色 |
| ROLE-002 | 创建自定义平台角色 | - | POST `/api/iam/v1/roles` body: `{spec: {name: "custom-role", scope: "platform", rules: [{resource: "iam:users:*"}]}}` | 201 |
| ROLE-003 | 角色名称格式验证 | - | `name: "INVALID"` (大写) | 400 |
| ROLE-004 | scope 无效 | - | `scope: "global"` | 400，scope 须为 platform/workspace/namespace |
| ROLE-005 | rules 为空 | - | `rules: []` | 400，规则不能为空 |
| ROLE-006 | 无效的权限规则模式 | - | `rules: [{resource: ":::invalid"}]` | 400，权限模式格式无效 |
| ROLE-007 | 规则无法匹配任何权限 | - | `rules: [{resource: "nonexist:module:*"}]` | 400，规则须匹配至少一个已注册权限 |
| ROLE-008 | 更新自定义角色 | 自定义角色已存在 | PUT `/api/iam/v1/roles/{id}` | 200 |
| ROLE-009 | 不能修改内置角色 | - | PUT 更新 platform-admin 角色 | 403/409，不允许修改内置角色 |
| ROLE-010 | 删除自定义角色（无绑定） | 角色无活跃绑定 | DELETE `/api/iam/v1/roles/{id}` | 200 |
| ROLE-011 | 删除有活跃绑定的角色 | 角色已绑定给用户 | DELETE `/api/iam/v1/roles/{id}` | 409，角色有活跃绑定无法删除 |
| ROLE-012 | 不能删除内置角色 | - | DELETE platform-admin 角色 | 403/409 |

### 6.2 工作空间级角色

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ROLE-020 | 列出工作空间角色 | 工作空间已创建（含内置角色） | GET `/api/iam/v1/workspaces/{wsId}/roles` | 200，包含 workspace-admin、workspace-viewer 等 |
| ROLE-021 | 创建工作空间自定义角色 | - | POST `/api/iam/v1/workspaces/{wsId}/roles` body: `{spec: {name: "ws-custom", scope: "workspace", rules: [...]}}` | 201 |
| ROLE-022 | 更新工作空间自定义角色 | 角色存在 | PUT `/api/iam/v1/workspaces/{wsId}/roles/{id}` | 200 |
| ROLE-023 | 删除工作空间自定义角色 | 角色无绑定 | DELETE `/api/iam/v1/workspaces/{wsId}/roles/{id}` | 200 |

### 6.3 Namespace 级角色

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ROLE-030 | 列出 Namespace 角色 | namespace 已创建 | GET `/api/iam/v1/workspaces/{wsId}/namespaces/{nsId}/roles` | 200 |
| ROLE-031 | 创建 Namespace 自定义角色 | - | POST `.../namespaces/{nsId}/roles` | 201 |
| ROLE-032 | 通过顶级路径操作 | - | GET `/api/iam/v1/namespaces/{nsId}/roles` | 200 |

---

## 7. 角色绑定管理

### 7.1 平台级角色绑定

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| RB-001 | 列出平台级角色绑定 | 存在角色绑定 | GET `/api/iam/v1/rolebindings` | 200 |
| RB-002 | 创建平台级角色绑定 | 用户和角色存在 | POST `/api/iam/v1/rolebindings` body: `{spec: {userId: "1", roleId: "2", scope: "platform"}}` | 201 |
| RB-003 | 角色必须为平台 scope | 使用 workspace scope 的角色 | POST 创建绑定 | 400，角色 scope 不匹配 |
| RB-004 | 删除角色绑定 | 绑定存在且非 owner | DELETE `/api/iam/v1/rolebindings/{id}` | 200 |
| RB-005 | 不能删除 owner 角色绑定 | 绑定的 isOwner=true | DELETE 该绑定 | 409，不允许删除 owner 绑定 |
| RB-006 | 创建后权限缓存失效 | 用户有缓存 | 创建绑定后检查该用户权限 | 缓存已刷新，新权限立即生效 |

### 7.2 工作空间级角色绑定

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| RB-010 | 列出工作空间级角色绑定 | - | GET `/api/iam/v1/workspaces/{wsId}/rolebindings` | 200 |
| RB-011 | 创建工作空间级角色绑定 | 角色属于该工作空间 | POST `.../workspaces/{wsId}/rolebindings` | 201 |
| RB-012 | 角色不属于该工作空间 | 使用其他工作空间的角色 | POST 创建绑定 | 400 |

### 7.3 Namespace 级角色绑定

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| RB-020 | 列出 Namespace 级角色绑定 | - | GET `.../namespaces/{nsId}/rolebindings` | 200 |
| RB-021 | 创建 Namespace 级角色绑定 | 角色属于该 namespace | POST `.../namespaces/{nsId}/rolebindings` | 201 |

### 7.4 用户视角

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| RB-030 | 查询用户的所有角色绑定 | 用户有多级角色绑定 | GET `/api/iam/v1/users/{userId}:rolebindings` | 200，返回所有级别的角色绑定 |
| RB-031 | 查询用户的展开权限 | 用户有 platform-admin 角色 | GET `/api/iam/v1/users/{userId}:permissions` | 200，`isPlatformAdmin=true`，platform 包含所有权限码 |
| RB-032 | 工作空间级权限展开 | 用户有 workspace-admin 角色 | GET 用户权限 | workspaces 字段含该工作空间的展开权限列表 |
| RB-033 | 通配符权限展开 | 角色 rules 含 `*:*` | GET 用户权限 | 展开为所有已注册的具体权限码 |

---

## 8. 权限管理

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| PERM-001 | 列出所有权限 | 系统启动后权限已同步 | GET `/api/iam/v1/permissions` | 200，返回自动注册的权限列表 |
| PERM-002 | 按模块筛选 | - | GET `/api/iam/v1/permissions?module=iam` | 200，仅返回 iam 模块权限 |
| PERM-003 | 按 scope 筛选 | - | GET `/api/iam/v1/permissions?scope=platform` | 200，仅返回平台级权限 |
| PERM-004 | 关键字搜索 | - | GET `/api/iam/v1/permissions?search=users` | 200，返回包含 "users" 的权限 |
| PERM-005 | 分页查询 | 权限数量 > 20 | GET `/api/iam/v1/permissions?page=2&pageSize=10` | 200，返回第 2 页 |
| PERM-006 | 权限码格式验证 | - | 检查返回的权限码 | 格式为 `{module}:{resourceChain}:{verb}`，如 `iam:users:list` |
| PERM-007 | 权限为只读 | - | POST `/api/iam/v1/permissions` | 405，权限不可手动创建 |

---

## 9. RBAC 授权

### 9.1 认证中间件

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| AUTH-001 | 无 token 访问业务 API | - | GET `/api/iam/v1/users` 不带 Authorization header | 401 |
| AUTH-002 | 无效 token 访问 | - | `Authorization: Bearer invalid-token` | 401 |
| AUTH-003 | 过期 token 访问 | token 已过期 | 使用过期 token 访问 | 401 |
| AUTH-004 | inactive 用户使用有效 token | 用户在 token 有效期内被设为 inactive | 使用之前获取的有效 token 访问 API | 401，用户已停用 |
| AUTH-005 | OIDC 端点不需要认证 | - | GET `/.well-known/openid-configuration` 不带 token | 200 |

### 9.2 授权中间件

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| AUTH-010 | 平台管理员访问任意资源 | 用户有 platform-admin 角色（rules: `*:*`） | 访问任意 API 端点 | 200，直接放行（短路逻辑） |
| AUTH-011 | 用户访问自己的资源 | 普通用户（无特殊权限） | GET `/api/iam/v1/users/{self}` | 200，自访问允许 |
| AUTH-012 | 用户修改自己的密码 | 普通用户 | POST `/api/iam/v1/users/{self}/change-password` | 200，自访问允许 |
| AUTH-013 | 用户访问他人资源 | 无 iam:users:get 权限 | GET `/api/iam/v1/users/{other}` | 403 |
| AUTH-014 | 工作空间级权限检查 | 用户有 ws-1 的 workspace-admin 角色 | GET `/api/iam/v1/workspaces/{ws-1}/users` | 200 |
| AUTH-015 | 工作空间级权限不跨工作空间 | 用户有 ws-1 权限但无 ws-2 权限 | GET `/api/iam/v1/workspaces/{ws-2}/users` | 403 |
| AUTH-016 | 平台权限向下继承 | 用户有平台级 `iam:workspaces:*` 权限 | 访问任意工作空间资源 | 200，平台权限向下继承 |
| AUTH-017 | 工作空间权限向 Namespace 继承 | 用户有 ws-1 的 workspace-admin 角色 | 访问 ws-1 下任意 namespace 资源 | 200 |
| AUTH-018 | Namespace 权限不向上继承 | 用户仅有 ns-1 的 namespace-admin 角色 | 访问 ns-1 所属工作空间级 API | 403 |
| AUTH-019 | 非管理员列出工作空间（访问过滤） | 普通用户是 ws-1 成员 | GET `/api/iam/v1/workspaces` | 200，仅返回 ws-1（不返回无权限的工作空间） |
| AUTH-020 | 非管理员列出 Namespace（访问过滤） | 普通用户是 ns-1、ns-2 成员 | GET `/api/iam/v1/namespaces` | 200，仅返回 ns-1、ns-2 |

### 9.3 权限匹配

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| AUTH-030 | 全通配 `*:*` 匹配所有权限 | 角色规则 `*:*` | 检查任意权限码 | 匹配成功 |
| AUTH-031 | 动词通配 `*:list` 匹配所有 list 操作 | 角色规则 `*:list` | 检查 `iam:users:list` | 匹配成功 |
| AUTH-032 | 动词通配不匹配其他动词 | 角色规则 `*:list` | 检查 `iam:users:create` | 匹配失败 |
| AUTH-033 | 模块通配 `iam:*` 匹配所有 IAM 操作 | 角色规则 `iam:*` | 检查 `iam:users:delete` | 匹配成功 |
| AUTH-034 | 模块通配不匹配其他模块 | 角色规则 `iam:*` | 检查 `dashboard:overview:get` | 匹配失败 |
| AUTH-035 | 资源通配 `iam:users:*` | 角色规则 `iam:users:*` | 检查 `iam:users:create` | 匹配成功 |
| AUTH-036 | 资源通配不匹配其他资源 | 角色规则 `iam:users:*` | 检查 `iam:workspaces:list` | 匹配失败 |
| AUTH-037 | 精确匹配 | 角色规则 `iam:users:list` | 检查 `iam:users:list` | 匹配成功 |
| AUTH-038 | 精确匹配不匹配其他 | 角色规则 `iam:users:list` | 检查 `iam:users:get` | 匹配失败 |

### 9.4 权限缓存

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| AUTH-040 | 缓存生效减少 DB 查询 | 用户首次请求后缓存建立 | 短时间内再次请求 | 使用缓存，无额外 DB 查询 |
| AUTH-041 | 角色绑定变更后缓存失效 | 用户有缓存 | 为用户添加新角色绑定 | 缓存被清除，下次请求重新加载 |
| AUTH-042 | 并发请求 singleflight 去重 | 缓存未命中 | 同一用户并发 10 个请求 | 仅 1 次 DB 查询（singleflight 去重） |
| AUTH-043 | TTL 过期后重新加载 | 缓存已建立 | 等待缓存 TTL 过期后请求 | 触发重新加载 |

---

## 10. Dashboard 统计

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| DASH-001 | 平台概览统计 | 存在用户、工作空间、namespace、角色等 | GET `/api/dashboard/v1/overview` | 200，返回 `{workspaceCount, namespaceCount, userCount, roleCount, roleBindingCount}` |
| DASH-002 | 工作空间概览统计 | 工作空间下有成员和 namespace | GET `/api/dashboard/v1/workspaces/{wsId}/overview` | 200，返回 `{namespaceCount, memberCount, roleCount, roleBindingCount}` |
| DASH-003 | Namespace 概览统计 | namespace 下有成员 | GET `/api/dashboard/v1/workspaces/{wsId}/namespaces/{nsId}/overview` | 200，返回 `{memberCount, roleCount, roleBindingCount}` |
| DASH-004 | 统计数据准确性 | 创建 2 个工作空间、3 个用户 | 查询平台概览 | workspaceCount=2, userCount=3（与实际一致） |
| DASH-005 | 不存在的工作空间统计 | - | GET `/api/dashboard/v1/workspaces/99999/overview` | 404 |

---

## 11. 主机（Host）管理

### 11.1 创建主机

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HOST-001 | 创建平台级主机 | 已认证，有权限 | POST `/api/infra/v1/hosts` body: `{spec: {name: "host-01", hostname: "srv01.example.com", ipAddress: "10.0.0.1", scope: "platform"}}` | 201，返回主机对象，scope=platform |
| HOST-002 | 创建工作空间级主机 | 工作空间存在 | POST `/api/infra/v1/workspaces/{wsId}/hosts` body: `{spec: {name: "ws-host-01", hostname: "ws-srv01", ipAddress: "10.0.1.1", scope: "workspace"}}` | 201，scope=workspace，workspaceId 已设置 |
| HOST-003 | 创建 Namespace 级主机 | namespace 存在 | POST `/api/infra/v1/workspaces/{wsId}/namespaces/{nsId}/hosts` body: `{spec: {name: "ns-host-01", hostname: "ns-srv01", ipAddress: "10.0.2.1", scope: "namespace"}}` | 201，scope=namespace，namespaceId 已设置 |
| HOST-004 | 创建主机含完整硬件信息 | - | POST 含 `os: "linux", arch: "amd64", cpuCores: 8, memoryMB: 16384, diskGB: 500` | 201，所有硬件信息正确存储 |
| HOST-005 | 创建主机含 labels | - | POST 含 `labels: {"env": "prod", "region": "cn-east"}` | 201，JSONB labels 正确存储 |
| HOST-006 | 名称格式验证 | - | `name: "INVALID"` (大写) | 400，名称仅允许小写字母、数字和连字符 |
| HOST-007 | 名称过短 | - | `name: "ab"` | 400 |
| HOST-008 | 名称以连字符开头 | - | `name: "-host"` | 400 |
| HOST-009 | scope 无效 | - | `scope: "global"` | 400，scope 须为 platform/workspace/namespace |
| HOST-010 | status 无效 | - | `status: "deleted"` | 400，status 须为 active/inactive |
| HOST-011 | 默认 status 为 active | - | 创建主机不传 status | 201，status 默认为 active |
| HOST-012 | 同 scope 下名称重复 | 已存在平台级 host-01 | 再创建平台级 `name: "host-01"` | 409 |

### 11.2 查询主机

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HOST-020 | 获取主机详情 | 主机存在 | GET `/api/infra/v1/hosts/{id}` | 200，返回主机信息含环境名称（如已绑定） |
| HOST-021 | 主机不存在 | - | GET `/api/infra/v1/hosts/99999` | 404 |
| HOST-022 | 列出平台级主机 | 存在多台平台级主机 | GET `/api/infra/v1/hosts` | 200，返回平台级主机列表 |
| HOST-023 | 列出工作空间下主机（含 owned 和 assigned） | ws-1 有自有主机和被分配的平台主机 | GET `/api/infra/v1/workspaces/{wsId}/hosts` | 200，返回两类主机，origin 字段分别为 "owned" 和 "assigned" |
| HOST-024 | 列出 Namespace 下主机（含 owned 和 assigned） | ns-1 有自有主机和被分配的主机 | GET `.../namespaces/{nsId}/hosts` | 200，origin 区分 "owned" 和 "assigned" |
| HOST-025 | 按 status 筛选 | 存在 active 和 inactive 主机 | GET `/api/infra/v1/hosts?status=active` | 200，仅返回 active 主机 |
| HOST-026 | 按 environmentId 筛选 | 部分主机绑定了环境 | GET `/api/infra/v1/hosts?environmentId=1` | 200，仅返回绑定该环境的主机 |
| HOST-027 | 关键字搜索 | - | GET `/api/infra/v1/hosts?search=srv01` | 200，按名称/hostname 模糊匹配 |
| HOST-028 | 分页与排序 | 存在 30 台主机 | GET `/api/infra/v1/hosts?page=2&pageSize=10&sortBy=name&sortOrder=asc` | 200，第 2 页 10 条，按名称升序 |
| HOST-029 | 按 IP 排序 | - | GET `/api/infra/v1/hosts?sortBy=ip_address` | 200，按 IP 地址排序 |

### 11.3 更新/删除主机

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HOST-030 | 全量更新主机 | 主机已存在 | PUT `/api/infra/v1/hosts/{id}` | 200 |
| HOST-031 | 部分更新主机 | 主机已存在 | PATCH `/api/infra/v1/hosts/{id}` body: `{spec: {status: "inactive"}}` | 200，仅更新 status |
| HOST-032 | 删除主机 | 主机存在 | DELETE `/api/infra/v1/hosts/{id}` | 200 |
| HOST-033 | 批量删除主机 | 多台主机存在 | DELETE `/api/infra/v1/hosts` body: `{ids: ["1", "2"]}` | 200，返回成功/失败计数 |

---

## 12. 环境（Environment）管理

### 12.1 创建环境

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ENV-001 | 创建平台级环境 | 已认证，有权限 | POST `/api/infra/v1/environments` body: `{spec: {name: "production", envType: "production", scope: "platform"}}` | 201，scope=platform |
| ENV-002 | 创建工作空间级环境 | 工作空间存在 | POST `/api/infra/v1/workspaces/{wsId}/environments` body: `{spec: {name: "staging", envType: "staging", scope: "workspace"}}` | 201，scope=workspace，workspaceId 已设置 |
| ENV-003 | 创建 Namespace 级环境 | namespace 存在 | POST `.../namespaces/{nsId}/environments` body: `{spec: {name: "dev", envType: "development", scope: "namespace"}}` | 201，scope=namespace，namespaceId 已设置 |
| ENV-004 | 所有合法 envType | - | 分别使用 development、testing、staging、production、custom | 均 201 |
| ENV-005 | 无效 envType | - | `envType: "preview"` | 400 |
| ENV-006 | 名称格式验证 | - | `name: "PROD"` (大写) | 400 |
| ENV-007 | scope 无效 | - | `scope: "global"` | 400 |
| ENV-008 | 同 scope 下名称重复 | 平台级已有 production | 再创建平台级 `name: "production"` | 409 |
| ENV-009 | 不同 scope 同名允许 | 平台级有 production | 在 ws-1 下创建 `name: "production"` | 201（不同 scope 不冲突） |
| ENV-010 | 默认 status 为 active | - | 创建环境不传 status | 201，status=active |

### 12.2 查询环境

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ENV-020 | 获取环境详情 | 环境存在 | GET `/api/infra/v1/environments/{id}` | 200，返回含 hostCount（已绑定主机数）的环境信息 |
| ENV-021 | 列出平台级环境 | 存在多个平台级环境 | GET `/api/infra/v1/environments` | 200 |
| ENV-022 | 列出工作空间下环境（含子 namespace 环境） | ws-1 下有 workspace 级和 namespace 级环境 | GET `/api/infra/v1/workspaces/{wsId}/environments` | 200，包含该工作空间及其下属 namespace 的环境 |
| ENV-023 | 列出 Namespace 下环境 | namespace 有环境 | GET `.../namespaces/{nsId}/environments` | 200，仅返回该 namespace 的环境 |
| ENV-024 | 按 envType 筛选 | 存在多种类型环境 | GET `/api/infra/v1/environments?envType=production` | 200，仅返回 production 类型 |
| ENV-025 | 分页与排序 | 存在多个环境 | GET `/api/infra/v1/environments?sortBy=env_type&sortOrder=asc` | 200，按环境类型排序 |
| ENV-026 | 查看环境下的主机列表 | 环境有绑定主机 | GET `/api/infra/v1/environments/{envId}:hosts` | 200，返回绑定到该环境的主机列表 |
| ENV-027 | 环境无绑定主机 | 环境 hostCount=0 | GET `/api/infra/v1/environments/{envId}:hosts` | 200，返回空列表 |

### 12.3 更新/删除环境

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ENV-030 | 全量更新环境 | 环境已存在 | PUT `/api/infra/v1/environments/{id}` | 200 |
| ENV-031 | 部分更新环境 | 环境已存在 | PATCH body: `{spec: {description: "新描述"}}` | 200 |
| ENV-032 | 删除环境 | 环境存在 | DELETE `/api/infra/v1/environments/{id}` | 200 |
| ENV-033 | 删除环境后主机绑定自动清除 | 主机 H1 绑定了环境 E1 | 删除 E1 后查询 H1 | H1 的 environmentId 为空（ON DELETE SET NULL） |
| ENV-034 | 批量删除环境 | 多个环境存在 | DELETE `/api/infra/v1/environments` body: `{ids: ["1", "2"]}` | 200 |

---

## 13. 主机-环境绑定

### 13.1 绑定环境

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| BIND-001 | 平台级主机绑定平台级环境 | 平台主机 H1 和平台环境 E1 均存在，H1 未绑定环境 | POST `/api/infra/v1/hosts/{H1}:bind-environment` body: `{environmentId: "E1"}` | 200，H1.environmentId=E1 |
| BIND-002 | 工作空间级主机绑定工作空间级环境 | ws-1 下主机 H2 和环境 E2 | POST `.../workspaces/{wsId}/hosts/{H2}:bind-environment` body: `{environmentId: "E2"}` | 200 |
| BIND-003 | Namespace 级主机绑定 Namespace 级环境 | ns-1 下主机 H3 和环境 E3 | POST `.../namespaces/{nsId}/hosts/{H3}:bind-environment` body: `{environmentId: "E3"}` | 200 |
| BIND-004 | 主机已绑定环境时再次绑定 | H1 已绑定 E1 | POST bind-environment body: `{environmentId: "E2"}` | 幂等处理：若当前已有绑定则不覆盖（仅当 environment_id IS NULL 时生效） |
| BIND-005 | 缺少 environmentId | - | POST bind-environment body: `{}` | 400，environmentId 为必填 |
| BIND-006 | 绑定后查询主机详情 | H1 已绑定 E1 | GET `/api/infra/v1/hosts/{H1}` | 200，spec 中 environmentId 和 environmentName 已填充 |
| BIND-007 | 绑定后环境 hostCount 增加 | E1 原有 0 台主机 | 绑定 H1 到 E1 后查询 E1 | hostCount=1 |

### 13.2 解绑环境

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| BIND-010 | 正常解绑环境 | H1 已绑定 E1 | POST `/api/infra/v1/hosts/{H1}:unbind-environment` | 200，H1.environmentId 为空 |
| BIND-011 | 未绑定时解绑 | H1 未绑定任何环境 | POST unbind-environment | 200（幂等操作，不报错） |
| BIND-012 | 解绑后环境 hostCount 减少 | E1 有 2 台主机 | 解绑 1 台后查询 E1 | hostCount=1 |

### 13.3 跨层级绑定场景

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| BIND-020 | 平台主机绑定工作空间级环境 | 平台主机 H1，ws-1 下环境 E2 | POST bind-environment body: `{environmentId: "E2"}` | 根据业务规则验证是否允许跨层级绑定 |
| BIND-021 | 工作空间主机绑定 Namespace 级环境 | ws-1 主机 H2，ns-1 下环境 E3 | POST bind-environment | 根据业务规则验证 |
| BIND-022 | Namespace 主机绑定平台级环境 | ns-1 主机 H3，平台环境 E1 | POST bind-environment | 根据业务规则验证 |
| BIND-023 | 同一环境绑定多台主机 | 环境 E1 已绑定 H1 | 将 H2 也绑定到 E1 | 200，环境支持绑定多台主机（一对多） |
| BIND-024 | 删除环境后所有绑定主机自动解绑 | E1 绑定了 H1、H2、H3 | 删除 E1 后查询 H1/H2/H3 | 所有主机 environmentId 为空（FK ON DELETE SET NULL） |

---

## 14. 主机分配（Host Assignment）

### 14.1 分配主机

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ASSIGN-001 | 平台主机分配到工作空间 | 平台主机 H1 存在，ws-1 存在 | POST `/api/infra/v1/hosts/{H1}:assign` body: `{workspaceId: "ws-1"}` | 200，H1 出现在 ws-1 的主机列表中（origin=assigned） |
| ASSIGN-002 | 平台主机分配到 Namespace | 平台主机 H1 存在，ns-1 存在 | POST assign body: `{namespaceId: "ns-1"}` | 200，H1 出现在 ns-1 的主机列表中（origin=assigned） |
| ASSIGN-003 | 工作空间主机分配到同工作空间下的 Namespace | ws-1 主机 H2，ns-1 属于 ws-1 | POST assign body: `{namespaceId: "ns-1"}` | 200 |
| ASSIGN-004 | 工作空间主机分配到其他工作空间的 Namespace | ws-1 主机 H2，ns-2 属于 ws-2 | POST assign body: `{namespaceId: "ns-2"}` | 400，工作空间主机只能分配到同工作空间下的 namespace |
| ASSIGN-005 | Namespace 级主机不可分配 | ns-1 主机 H3 | POST assign body: `{workspaceId: "ws-1"}` | 400，namespace 级主机不允许分配 |
| ASSIGN-006 | 同时指定 workspaceId 和 namespaceId | - | POST assign body: `{workspaceId: "1", namespaceId: "1"}` | 400，须且仅须指定其一 |
| ASSIGN-007 | 未指定 workspaceId 和 namespaceId | - | POST assign body: `{}` | 400，须指定 workspaceId 或 namespaceId |
| ASSIGN-008 | 平台主机同时分配到多个工作空间 | 平台主机 H1 | 分别 assign 到 ws-1 和 ws-2 | 200，H1 同时出现在 ws-1 和 ws-2 的列表中 |
| ASSIGN-009 | 平台主机同时分配到多个 Namespace | 平台主机 H1 | 分别 assign 到 ns-1 和 ns-2 | 200，H1 同时出现在两个 namespace 列表中 |

### 14.2 取消分配

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ASSIGN-010 | 取消工作空间分配 | H1 已分配到 ws-1 | POST `/api/infra/v1/hosts/{H1}:unassign` body: `{workspaceId: "ws-1"}` | 200，H1 不再出现在 ws-1 的主机列表中 |
| ASSIGN-011 | 取消 Namespace 分配 | H1 已分配到 ns-1 | POST unassign body: `{namespaceId: "ns-1"}` | 200 |

### 14.3 查看分配记录

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ASSIGN-020 | 查看主机的所有分配 | H1 分配到 ws-1 和 ns-2 | GET `/api/infra/v1/hosts/{H1}:assignments` | 200，返回 2 条分配记录，含工作空间名/namespace 名 |
| ASSIGN-021 | 未分配的主机 | H1 无任何分配 | GET `/api/infra/v1/hosts/{H1}:assignments` | 200，返回空列表 |

### 14.4 分配与列表展示

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| ASSIGN-030 | 工作空间列表区分 owned 和 assigned | ws-1 有自有主机 H2（scope=workspace），平台主机 H1 分配到 ws-1 | GET `.../workspaces/{wsId}/hosts` | H2.origin="owned"，H1.origin="assigned" |
| ASSIGN-031 | Namespace 列表区分 owned 和 assigned | ns-1 有自有主机 H3（scope=namespace），ws-1 主机 H2 分配到 ns-1 | GET `.../namespaces/{nsId}/hosts` | H3.origin="owned"，H2.origin="assigned" |
| ASSIGN-032 | 工作空间列表含下属 namespace 的主机 | ws-1 下 ns-1 有主机 H3 | GET `.../workspaces/{wsId}/hosts` | 列表中包含 H3（scope=namespace 且属于 ws-1 下的 namespace） |

---

## 15. Hosts/Env 多层级权限控制

### 15.1 平台级权限

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HPERM-001 | 平台管理员操作所有 hosts/env | 用户有 platform-admin 角色（`*:*`） | 对平台/工作空间/namespace 级别的 hosts 和 environments 执行 CRUD | 全部 200 |
| HPERM-002 | 拥有 `infra:*` 平台权限的用户 | 自定义角色规则 `infra:*`，scope=platform | 操作所有层级 infra 资源 | 全部 200（平台权限向下继承） |
| HPERM-003 | 拥有 `infra:hosts:*` 平台权限 | 规则 `infra:hosts:*`，scope=platform | 操作所有层级主机（CRUD + assign + bind） | 200 |
| HPERM-004 | 拥有 `infra:hosts:*` 但无 environments 权限 | 规则仅 `infra:hosts:*` | 操作环境 API：GET `/api/infra/v1/environments` | 403，无 `infra:environments:list` 权限 |
| HPERM-005 | 拥有 `infra:environments:*` 但无 hosts 权限 | 规则仅 `infra:environments:*` | 操作主机 API：GET `/api/infra/v1/hosts` | 403，无 `infra:hosts:list` 权限 |
| HPERM-006 | 平台只读用户 | 规则 `*:list` + `*:get`，scope=platform | 列出/查看所有层级 hosts 和 environments | 200 |
| HPERM-007 | 平台只读用户尝试创建 | 同上 | POST 创建主机 | 403，无 `infra:hosts:create` 权限 |
| HPERM-008 | 平台只读用户尝试绑定环境 | 同上 | POST bind-environment | 403，无 `infra:hosts:bind-environment` 权限 |
| HPERM-009 | 平台只读用户尝试分配主机 | 同上 | POST assign | 403，无 `infra:hosts:assign` 权限 |

### 15.2 工作空间级权限

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HPERM-010 | 工作空间管理员操作本工作空间 hosts | 用户有 ws-1 的 workspace-admin 角色 | CRUD ws-1 下的 hosts | 全部 200 |
| HPERM-011 | 工作空间管理员操作本工作空间 environments | 同上 | CRUD ws-1 下的 environments | 全部 200 |
| HPERM-012 | 工作空间管理员操作下属 namespace 的 hosts | 用户有 ws-1 的 workspace-admin 角色，ns-1 属于 ws-1 | CRUD ns-1 下的 hosts | 200（工作空间权限向 namespace 继承） |
| HPERM-013 | 工作空间管理员操作下属 namespace 的 environments | 同上 | CRUD ns-1 下的 environments | 200 |
| HPERM-014 | 工作空间管理员不能操作平台级 hosts | 用户仅有 ws-1 的 workspace-admin 角色 | GET `/api/infra/v1/hosts` | 取决于访问过滤逻辑：可能返回受限列表或 403 |
| HPERM-015 | 工作空间管理员不能操作其他工作空间 hosts | 用户有 ws-1 权限，无 ws-2 权限 | GET `/api/infra/v1/workspaces/{ws-2}/hosts` | 403 |
| HPERM-016 | 工作空间管理员不能操作其他工作空间 environments | 同上 | GET `/api/infra/v1/workspaces/{ws-2}/environments` | 403 |
| HPERM-017 | 工作空间自定义角色（仅 hosts 读权限） | 自定义角色：`infra:hosts:list` + `infra:hosts:get`，scope=workspace，绑定 ws-1 | 列出/查看 ws-1 的 hosts | 200 |
| HPERM-018 | 工作空间自定义角色不可创建 hosts | 同上 | POST 创建 ws-1 下的 host | 403 |
| HPERM-019 | 工作空间自定义角色不可操作 environments | 同上（无 environments 权限） | GET ws-1 environments | 403 |

### 15.3 Namespace 级权限

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HPERM-020 | Namespace 管理员操作本 namespace hosts | 用户有 ns-1 的 namespace-admin 角色 | CRUD ns-1 下的 hosts | 全部 200 |
| HPERM-021 | Namespace 管理员操作本 namespace environments | 同上 | CRUD ns-1 下的 environments | 全部 200 |
| HPERM-022 | Namespace 权限不向上继承到工作空间 | 用户仅有 ns-1 的 namespace-admin 角色 | GET `/api/infra/v1/workspaces/{ws-1}/hosts` | 403，namespace 权限不能操作工作空间级资源 |
| HPERM-023 | Namespace 权限不向上继承到平台 | 同上 | GET `/api/infra/v1/hosts` | 403 或返回空列表（取决于访问过滤逻辑） |
| HPERM-024 | Namespace 权限不跨 namespace | 用户有 ns-1 权限无 ns-2 权限 | GET `.../namespaces/{ns-2}/hosts` | 403 |
| HPERM-025 | Namespace 自定义角色（仅 environments 读权限） | 自定义角色：`infra:environments:list` + `infra:environments:get`，scope=namespace | 列出/查看 ns-1 的 environments | 200 |
| HPERM-026 | Namespace 自定义角色不可创建 environments | 同上 | POST 创建 ns-1 下的 environment | 403 |

### 15.4 权限继承链（infra 资源完整验证）

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HPERM-030 | 平台 infra 权限覆盖所有层级 | 用户有平台 `infra:hosts:*` 权限 | 分别操作平台级/ws-1/ns-1 的 hosts | 全部 200 |
| HPERM-031 | 工作空间 infra 权限覆盖下属 namespace | 用户有 ws-1 的 `infra:environments:*` 权限 | 操作 ws-1 级 environments 和 ws-1 下 ns-1 的 environments | 全部 200 |
| HPERM-032 | 工作空间 infra 权限不覆盖平台 | 用户有 ws-1 的 `infra:hosts:*` 权限 | 操作平台级 hosts | 403 |
| HPERM-033 | namespace infra 权限仅限本 namespace | 用户有 ns-1 的 `infra:*` 权限 | 操作 ns-1 的 hosts/environments | 200 |
| HPERM-034 | namespace infra 权限不覆盖父工作空间 | 同上 | 操作 ws-1 级 hosts | 403 |
| HPERM-035 | namespace infra 权限不覆盖兄弟 namespace | 用户有 ns-1 的 `infra:*` 权限，ns-2 与 ns-1 同属 ws-1 | 操作 ns-2 的 hosts | 403 |

### 15.5 特殊操作权限

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HPERM-040 | assign 操作需要 `infra:hosts:assign` 权限 | 用户有 `infra:hosts:list` + `infra:hosts:get` 但无 assign | POST assign | 403 |
| HPERM-041 | bind-environment 需要 `infra:hosts:bind-environment` 权限 | 用户有 hosts CRUD 但无 bind-environment | POST bind-environment | 403 |
| HPERM-042 | unbind-environment 需要 `infra:hosts:unbind-environment` 权限 | 同理 | POST unbind-environment | 403 |
| HPERM-043 | unassign 需要 `infra:hosts:unassign` 权限 | 同理 | POST unassign | 403 |
| HPERM-044 | 查看 assignments 需要 `infra:hosts:assignments` 权限 | 用户有 hosts CRUD 但无 assignments | GET `{hostId}:assignments` | 403 |
| HPERM-045 | 查看环境下主机需要 `infra:environments:hosts` 权限 | 用户有 environments CRUD 但无 hosts verb | GET `{envId}:hosts` | 403 |
| HPERM-046 | 批量删除需要 `infra:hosts:deleteCollection` 权限 | 用户有 `infra:hosts:delete` 但无 deleteCollection | DELETE 批量删除 | 403 |

### 15.6 跨模块权限隔离

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HPERM-050 | IAM 权限不影响 infra | 用户有 `iam:*` 平台权限但无 `infra:*` | 操作 infra 资源（hosts/environments） | 403 |
| HPERM-051 | infra 权限不影响 IAM | 用户有 `infra:*` 平台权限但无 `iam:*` | 操作 IAM 资源（users/workspaces） | 403 |
| HPERM-052 | 同时拥有 IAM 和 infra 权限 | 角色规则 `iam:users:list` + `infra:hosts:list` | 分别操作两模块的 list | 均 200 |
| HPERM-053 | `*:*` 覆盖所有模块 | platform-admin 角色 | 同时操作 iam 和 infra 资源 | 均 200 |

### 15.7 角色绑定变更对 infra 资源的影响

| 编号 | 用例名称 | 前置条件 | 操作步骤 | 预期结果 |
|------|---------|---------|---------|---------|
| HPERM-060 | 新增 infra 角色绑定后立即生效 | 用户原无 infra 权限 | 为用户绑定含 `infra:hosts:*` 的角色，立即操作 hosts | 200（缓存已失效，新权限生效） |
| HPERM-061 | 移除 infra 角色绑定后立即失效 | 用户有 infra 权限 | 解除角色绑定后立即操作 hosts | 403（缓存已失效，权限已移除） |
| HPERM-062 | 角色规则更新后影响 infra 权限 | 用户有自定义角色含 `infra:*` | 将角色规则改为 `iam:*`（移除 infra），用户操作 hosts | 403（全局缓存失效） |
| HPERM-063 | 删除工作空间后相关 infra 权限清理 | 用户有 ws-1 的 infra 权限 | 删除 ws-1 | 用户在 ws-1 的 infra 权限缓存被清除 |

---

## 附录：验证规则速查

| 字段 | 正则/规则 | 有效示例 | 无效示例 |
|------|----------|---------|---------|
| username | `^[a-zA-Z0-9_]{3,50}$` | `user_123`, `Admin01` | `ab`, `test-user`, `a@b` |
| email | RFC 5322 | `user@example.com` | `not-email`, `@no.com` |
| phone | `^1[3-9]\d{9}$` | `13800138000` | `12345`, `10800138000` |
| password | 8-128 位，含大写+小写+数字 | `MyPass123` | `short`, `alllowercase1`, `ALLUPPERCASE1` |
| workspace/namespace/role name | `^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$` | `my-workspace`, `ns01` | `AB`, `-start`, `end-`, `a` |
| status | enum: active, inactive | `active` | `deleted`, `pending` |
| visibility | enum: public, private | `private` | `secret`, `internal` |
| scope | enum: platform, workspace, namespace | `platform` | `global`, `tenant` |
| envType | enum: development, testing, staging, production, custom | `production` | `preview`, `beta` |
| host/env name | `^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$` | `host-01`, `staging` | `HOST`, `-host`, `h` |
