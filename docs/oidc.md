# OIDC 认证流程

LCP 使用 Authorization Code + PKCE 流程实现 OIDC 认证。

## 流程概览

```
┌──────────┐     ┌──────────┐     ┌──────────────┐
│  Browser  │     │  LCP UI  │     │  LCP Server  │
└─────┬─────┘     └────┬─────┘     └──────┬───────┘
      │                │                   │
      │  1. 访问受保护页面  │                   │
      │───────────────>│                   │
      │                │                   │
      │  2. 生成 PKCE code_verifier + code_challenge
      │                │                   │
      │  3. GET /oidc/authorize            │
      │    ?response_type=code             │
      │    &client_id=lcp-ui               │
      │    &redirect_uri=.../auth/callback │
      │    &scope=openid profile email     │
      │    &state=<random>                 │
      │    &code_challenge=<S256 hash>     │
      │    &code_challenge_method=S256     │
      │───────────────────────────────────>│
      │                │                   │
      │  4. 302 → http://localhost:5173/login?request_id=<id>
      │<───────────────────────────────────│
      │                │                   │
      │  5. 显示登录页面    │                   │
      │───────────────>│                   │
      │                │                   │
      │  6. 用户输入账号密码  │                   │
      │───────────────>│                   │
      │                │  7. POST /oidc/login
      │                │  {identifier, password, requestId}
      │                │──────────────────>│
      │                │                   │
      │                │  8. 返回 redirectUrl  │
      │                │  (含 code + state)  │
      │                │<──────────────────│
      │                │                   │
      │  9. 前端跳转到 redirectUrl            │
      │───────────────────────────────────>│
      │                │                   │
      │  10. /auth/callback?code=<code>&state=<state>
      │───────────────>│                   │
      │                │                   │
      │                │  11. POST /oidc/token
      │                │  {grant_type=authorization_code,
      │                │   code, redirect_uri,
      │                │   client_id, code_verifier}
      │                │──────────────────>│
      │                │                   │
      │                │  12. 返回 tokens     │
      │                │  {access_token,    │
      │                │   id_token,        │
      │                │   refresh_token}   │
      │                │<──────────────────│
      │                │                   │
      │  13. 存储 tokens，进入应用             │
      │<───────────────│                   │
```

## 详细步骤

### 1. 发起授权请求

前端检测到用户未登录时，生成 PKCE 参数并跳转到授权端点：

```typescript
// 生成 PKCE code_verifier (随机字符串)
const codeVerifier = generateRandomString(128);
// 计算 code_challenge = BASE64URL(SHA256(code_verifier))
const codeChallenge = await computeS256Challenge(codeVerifier);

// 保存 code_verifier 到 sessionStorage (token 交换时需要)
sessionStorage.setItem("pkce_code_verifier", codeVerifier);

// 跳转到授权端点
window.location.href = `/oidc/authorize?` + new URLSearchParams({
  response_type: "code",
  client_id: "lcp-ui",
  redirect_uri: "http://localhost:5173/auth/callback",
  scope: "openid profile email phone",
  state: generateRandomString(32),
  code_challenge: codeChallenge,
  code_challenge_method: "S256",
});
```

### 2. 服务端处理授权请求

服务端 `/oidc/authorize` 端点：
1. 验证 `client_id`、`redirect_uri`、`response_type` 等参数
2. 将授权请求存储为 pending 状态，生成 `request_id`
3. 302 重定向到配置的 `loginUrl`（前端登录页），附带 `request_id`

### 3. 用户登录

前端登录页从 URL 获取 `request_id`，用户输入凭据后调用：

```
POST /oidc/login
Content-Type: application/json

{
  "identifier": "admin",
  "password": "password123",
  "requestId": "<request_id>"
}
```

服务端验证凭据后：
- 创建用户会话
- 根据 `request_id` 取出 pending 的授权请求
- 生成授权码（authorization code）
- 返回包含授权码的回调 URL

```json
{
  "redirectUrl": "http://localhost:5173/auth/callback?code=<code>&state=<state>"
}
```

### 4. 授权码换取 Token

前端 `/auth/callback` 页面从 URL 获取 `code`，用保存的 `code_verifier` 换取 token：

```
POST /oidc/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code=<authorization_code>
&redirect_uri=http://localhost:5173/auth/callback
&client_id=lcp-ui
&code_verifier=<code_verifier>
```

服务端验证授权码和 PKCE 后，返回：

```json
{
  "access_token": "<JWT>",
  "id_token": "<JWT>",
  "refresh_token": "<opaque_token>",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile email phone"
}
```

### 5. Token 刷新

Access token 过期前，使用 refresh token 获取新 token：

```
POST /oidc/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=<refresh_token>
&client_id=lcp-ui
```

## 配置

在 `config.yaml` 中配置 OIDC：

```yaml
oidc:
  issuer: "http://localhost:8428"
  algorithm: "EdDSA"
  accessTokenTTL: "1h"
  refreshTokenTTL: "168h"
  authCodeTTL: "5m"
  loginUrl: "http://localhost:5173/login"    # 前端登录页地址
  clients:
    - id: "lcp-ui"
      public: true
      redirectUris:
        - "http://localhost:5173/auth/callback"
      scopes: ["openid", "profile", "email", "phone"]
```

关键配置说明：
- `algorithm`：签名算法，支持 `EdDSA`（默认）、`ES256`、`RS256`。密钥自动生成并存储在 PostgreSQL（`oidc_signing_keys` 表）
- `loginUrl`：授权请求重定向的登录页地址，需指向前端应用
- `issuer`：OIDC 签发者标识，用于 token 签发和发现文档
- `clients`：注册的 OAuth2 客户端，`public: true` 表示公开客户端（无 client_secret）

## 相关端点

| 端点 | 说明 |
|------|------|
| `GET /.well-known/openid-configuration` | OIDC 发现文档 |
| `GET /.well-known/jwks.json` | JSON Web Key Set |
| `GET /oidc/authorize` | 授权端点 |
| `POST /oidc/login` | 登录端点 |
| `POST /oidc/token` | Token 端点 |
| `GET /oidc/userinfo` | 用户信息端点 |
