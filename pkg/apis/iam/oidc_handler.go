package iam

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"lcp.io/lcp/lib/logger"
	"lcp.io/lcp/lib/oidc"
)

// NewOIDCMux 创建包含所有 OIDC 公开端点的 HTTP 路由。
// 这些端点不经过认证中间件，公开访问：
//   - GET  /.well-known/openid-configuration — OIDC 发现文档
//   - GET  /.well-known/jwks.json            — JSON Web Key Set（公钥集）
//   - GET  /oidc/authorize                   — 授权端点，发起 Authorization Code Flow
//   - POST /oidc/login                       — 用户登录（用户名+密码），完成授权并返回授权码
//   - POST /oidc/token                       — 令牌端点，用授权码或刷新令牌换取访问令牌
//   - GET  /oidc/userinfo                    — 用户信息端点，通过 Bearer Token 获取当前用户信息
//   - POST /oidc/userinfo                    — 同上（支持 POST 方法）
func NewOIDCMux(provider *oidc.Provider) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", handleDiscovery(provider))
	mux.HandleFunc("GET /.well-known/jwks.json", handleJWKS(provider))
	mux.HandleFunc("GET /oidc/authorize", handleAuthorize(provider))
	mux.HandleFunc("POST /oidc/login", handleLogin(provider))
	mux.HandleFunc("POST /oidc/token", handleToken(provider))
	mux.HandleFunc("GET /oidc/userinfo", handleUserInfo(provider))
	mux.HandleFunc("POST /oidc/userinfo", handleUserInfo(provider))
	return mux
}

// handleDiscovery 返回 OIDC 发现文档（RFC 8414），包含授权、令牌、JWKS 等端点地址。
// +openapi:endpoint
// +openapi:method=GET
// +openapi:path=/.well-known/openid-configuration
// +openapi:summary=OIDC 发现文档
// +openapi:description=返回 OpenID Connect 发现文档，包含授权、令牌、JWKS 等端点地址
// +openapi:tag=OIDC
// +openapi:operationId=getOIDCDiscovery
// +openapi:response.200.description=OK
// +openapi:response.200.contentType=application/json
// +openapi:response.200.schema=OIDCDiscoveryResponse
func handleDiscovery(provider *oidc.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=3600")
		_ = json.NewEncoder(w).Encode(provider.DiscoveryDocument())
	}
}

// handleJWKS 返回 JSON Web Key Set，包含用于验证 JWT 签名的 ECDSA 公钥。
// +openapi:endpoint
// +openapi:method=GET
// +openapi:path=/.well-known/jwks.json
// +openapi:summary=JSON Web Key Set
// +openapi:description=返回用于验证 JWT 签名的 ECDSA 公钥集
// +openapi:tag=OIDC
// +openapi:operationId=getJWKS
// +openapi:response.200.description=OK
// +openapi:response.200.contentType=application/json
func handleJWKS(provider *oidc.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=3600")
		_ = json.NewEncoder(w).Encode(provider.JWKSet())
	}
}

// handleAuthorize 处理 OAuth2 授权请求。验证 client_id、redirect_uri 和 PKCE 参数后，
// 将请求存储为待处理状态，并重定向用户到登录页面。
// +openapi:endpoint
// +openapi:method=GET
// +openapi:path=/oidc/authorize
// +openapi:summary=授权端点
// +openapi:description=发起 OAuth2 Authorization Code Flow，验证参数后重定向到登录页面
// +openapi:tag=OIDC
// +openapi:operationId=oidcAuthorize
// +openapi:response.302.description=重定向到登录页面
func handleAuthorize(provider *oidc.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		responseType := q.Get("response_type")
		clientID := q.Get("client_id")
		redirectURI := q.Get("redirect_uri")
		scope := q.Get("scope")
		state := q.Get("state")
		nonce := q.Get("nonce")
		codeChallenge := q.Get("code_challenge")
		codeChallengeMethod := q.Get("code_challenge_method")

		if responseType != "code" {
			http.Redirect(w, r, "/error?status=400", http.StatusFound)
			return
		}

		client, ok := provider.GetClient(clientID)
		if !ok {
			http.Redirect(w, r, "/error?status=400", http.StatusFound)
			return
		}

		if redirectURI == "" {
			if len(client.RedirectURIs) > 0 {
				redirectURI = client.RedirectURIs[0]
			} else {
				http.Redirect(w, r, "/error?status=400", http.StatusFound)
				return
			}
		}

		if !provider.ValidateRedirectURI(client, redirectURI) {
			http.Redirect(w, r, "/error?status=403", http.StatusFound)
			return
		}

		// Public clients must use PKCE
		if client.Public && codeChallenge == "" {
			http.Redirect(w, r, "/error?status=400", http.StatusFound)
			return
		}

		if codeChallenge != "" && codeChallengeMethod != "S256" {
			http.Redirect(w, r, "/error?status=400", http.StatusFound)
			return
		}

		req := &oidc.AuthorizeRequest{
			ResponseType:        responseType,
			ClientID:            clientID,
			RedirectURI:         redirectURI,
			Scope:               scope,
			State:               state,
			Nonce:               nonce,
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: codeChallengeMethod,
		}

		requestID, err := provider.StorePendingAuthorize(req)
		if err != nil {
			http.Redirect(w, r, "/error?status=500", http.StatusFound)
			return
		}

		// Redirect to login page with request_id
		loginURL := provider.LoginURL() + "?request_id=" + requestID
		http.Redirect(w, r, loginURL, http.StatusFound)
	}
}

// handleLogin 处理用户登录请求。验证用户名和密码后：
// - 如果提供了 requestId，完成 OIDC 授权流程，生成授权码并返回回调 URL；
// - 如果未提供 requestId，执行直接登录，返回会话信息。
// +openapi:endpoint
// +openapi:method=POST
// +openapi:path=/oidc/login
// +openapi:summary=用户登录
// +openapi:description=验证用户名和密码，完成授权流程或直接登录
// +openapi:tag=OIDC
// +openapi:operationId=oidcLogin
// +openapi:requestBody.contentType=application/json
// +openapi:requestBody.schema=OIDCLoginRequest
// +openapi:response.200.description=登录成功
// +openapi:response.200.contentType=application/json
// +openapi:response.200.schema=OIDCLoginResponse
// +openapi:response.401.description=认证失败
// +openapi:response.401.contentType=application/json
// +openapi:response.401.schema=OIDCErrorResponse
func handleLogin(provider *oidc.Provider) http.HandlerFunc {
	type loginRequest struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		RequestID string `json:"requestId"`
	}
	type loginResponse struct {
		RedirectURI string `json:"redirectUri"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			oidcError(w, "invalid_request", "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			oidcError(w, "invalid_request", "username and password are required", http.StatusBadRequest)
			return
		}

		session, _, err := provider.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			logger.Infof("login failed for user %q: %v", req.Username, err)
			oidcError(w, "invalid_grant", "invalid credentials", http.StatusUnauthorized)
			return
		}

		// If requestId provided, complete the authorization flow
		if req.RequestID != "" {
			authReq, err := provider.GetPendingAuthorize(req.RequestID)
			if err != nil {
				oidcError(w, "invalid_request", "invalid or expired request_id", http.StatusBadRequest)
				return
			}

			redirectURL, err := provider.Authorize(r.Context(), authReq, session.UserID, session.AuthTime)
			if err != nil {
				oidcError(w, "server_error", "failed to generate authorization code", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(loginResponse{RedirectURI: redirectURL})
			return
		}

		// Direct login without OIDC flow — just return session info
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sessionId": session.SessionID,
			"userId":    session.UserID,
		})
	}
}

// handleToken 处理令牌请求，支持两种授权类型：
// - authorization_code：用授权码换取访问令牌、ID 令牌和刷新令牌；
// - refresh_token：用刷新令牌获取新的令牌对（令牌轮换）。
// +openapi:endpoint
// +openapi:method=POST
// +openapi:path=/oidc/token
// +openapi:summary=令牌端点
// +openapi:description=用授权码或刷新令牌换取访问令牌
// +openapi:tag=OIDC
// +openapi:operationId=oidcToken
// +openapi:requestBody.contentType=application/x-www-form-urlencoded
// +openapi:requestBody.schema=OIDCTokenRequest
// +openapi:response.200.description=令牌签发成功
// +openapi:response.200.contentType=application/json
// +openapi:response.200.schema=OIDCTokenResponse
// +openapi:response.400.description=请求无效
// +openapi:response.400.contentType=application/json
// +openapi:response.400.schema=OIDCErrorResponse
func handleToken(provider *oidc.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			oidcError(w, "invalid_request", "invalid form data", http.StatusBadRequest)
			return
		}

		grantType := r.FormValue("grant_type")
		switch grantType {
		case "authorization_code":
			handleCodeExchange(w, r, provider)
		case "refresh_token":
			handleRefreshToken(w, r, provider)
		default:
			oidcError(w, "unsupported_grant_type", "only authorization_code and refresh_token are supported", http.StatusBadRequest)
		}
	}
}

// handleCodeExchange 处理授权码换取令牌。验证客户端身份（机密客户端需要 client_secret），
// 验证 PKCE，然后签发 access_token、id_token 和 refresh_token。
func handleCodeExchange(w http.ResponseWriter, r *http.Request, provider *oidc.Provider) {
	req := &oidc.CodeExchangeRequest{
		Code:         r.FormValue("code"),
		RedirectURI:  r.FormValue("redirect_uri"),
		ClientID:     r.FormValue("client_id"),
		CodeVerifier: r.FormValue("code_verifier"),
	}

	if req.Code == "" || req.ClientID == "" {
		oidcError(w, "invalid_request", "code and client_id are required", http.StatusBadRequest)
		return
	}

	// Authenticate confidential clients
	client, ok := provider.GetClient(req.ClientID)
	if !ok {
		oidcError(w, "invalid_client", "unknown client_id", http.StatusUnauthorized)
		return
	}
	if !client.Public {
		clientSecret := r.FormValue("client_secret")
		if clientSecret == "" || subtle.ConstantTimeCompare([]byte(clientSecret), []byte(client.Secret)) != 1 {
			oidcError(w, "invalid_client", "invalid client credentials", http.StatusUnauthorized)
			return
		}
	}

	tokenPair, err := provider.ExchangeCode(r.Context(), req)
	if err != nil {
		logger.Infof("code exchange failed: %v", err)
		oidcError(w, "invalid_grant", "authorization code is invalid or expired", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_ = json.NewEncoder(w).Encode(tokenPair)
}

// handleRefreshToken 处理刷新令牌请求。原子消费旧刷新令牌（单次使用），签发新的令牌对。
func handleRefreshToken(w http.ResponseWriter, r *http.Request, provider *oidc.Provider) {
	req := &oidc.RefreshRequest{
		RefreshToken: r.FormValue("refresh_token"),
		ClientID:     r.FormValue("client_id"),
		Scope:        r.FormValue("scope"),
	}

	if req.RefreshToken == "" || req.ClientID == "" {
		oidcError(w, "invalid_request", "refresh_token and client_id are required", http.StatusBadRequest)
		return
	}

	// Authenticate confidential clients
	client, ok := provider.GetClient(req.ClientID)
	if !ok {
		oidcError(w, "invalid_client", "unknown client_id", http.StatusUnauthorized)
		return
	}
	if !client.Public {
		clientSecret := r.FormValue("client_secret")
		if clientSecret == "" || subtle.ConstantTimeCompare([]byte(clientSecret), []byte(client.Secret)) != 1 {
			oidcError(w, "invalid_client", "invalid client credentials", http.StatusUnauthorized)
			return
		}
	}

	tokenPair, err := provider.RefreshTokens(r.Context(), req)
	if err != nil {
		logger.Infof("token refresh failed: %v", err)
		oidcError(w, "invalid_grant", "refresh token is invalid or expired", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_ = json.NewEncoder(w).Encode(tokenPair)
}

// handleUserInfo 通过 Bearer Token 获取当前认证用户的信息。根据令牌中的 scope 返回相应的 claims。
// +openapi:endpoint
// +openapi:method=GET
// +openapi:path=/oidc/userinfo
// +openapi:summary=用户信息端点
// +openapi:description=通过 Bearer Token 获取当前认证用户的信息
// +openapi:tag=OIDC
// +openapi:operationId=getOIDCUserInfo
// +openapi:response.200.description=OK
// +openapi:response.200.contentType=application/json
// +openapi:response.200.schema=OIDCUserInfoResponse
// +openapi:response.401.description=令牌无效
// +openapi:response.401.contentType=application/json
// +openapi:response.401.schema=OIDCErrorResponse
func handleUserInfo(provider *oidc.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			oidcError(w, "invalid_token", "bearer token required", http.StatusUnauthorized)
			return
		}

		accessToken := strings.TrimPrefix(authHeader, "Bearer ")
		userInfo, err := provider.UserInfoForToken(r.Context(), accessToken)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			oidcError(w, "invalid_token", "invalid or expired token", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userInfo)
	}
}

// handleUserInfoPost 同 handleUserInfo，支持 POST 方法。
// +openapi:endpoint
// +openapi:method=POST
// +openapi:path=/oidc/userinfo
// +openapi:summary=用户信息端点（POST）
// +openapi:description=通过 Bearer Token 获取当前认证用户的信息（POST 方法）
// +openapi:tag=OIDC
// +openapi:operationId=postOIDCUserInfo
// +openapi:response.200.description=OK
// +openapi:response.200.contentType=application/json
// +openapi:response.200.schema=OIDCUserInfoResponse
// +openapi:response.401.description=令牌无效
// +openapi:response.401.contentType=application/json
// +openapi:response.401.schema=OIDCErrorResponse
func handleUserInfoPost() {} //nolint:unused // OpenAPI annotation target only

// oidcError 写入标准 OAuth2 错误响应（RFC 6749 Section 5.2）。
func oidcError(w http.ResponseWriter, errCode, description string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": description,
	})
}
