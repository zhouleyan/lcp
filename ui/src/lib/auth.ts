const CLIENT_ID = "lcp-ui"
const REDIRECT_URI = `${window.location.origin}/auth/callback`
const TOKEN_KEY = "lcp_access_token"
const REFRESH_TOKEN_KEY = "lcp_refresh_token"

function generateRandomString(length: number): string {
  const array = new Uint8Array(length)
  crypto.getRandomValues(array)
  return Array.from(array, (b) => b.toString(16).padStart(2, "0")).join("")
}

async function sha256(plain: string): Promise<ArrayBuffer> {
  const encoder = new TextEncoder()
  return crypto.subtle.digest("SHA-256", encoder.encode(plain))
}

function base64UrlEncode(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let str = ""
  for (const b of bytes) {
    str += String.fromCharCode(b)
  }
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")
}

export function getAccessToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function getRefreshToken(): string | null {
  return sessionStorage.getItem(REFRESH_TOKEN_KEY)
}

export function setTokens(accessToken: string, refreshToken: string) {
  sessionStorage.setItem(TOKEN_KEY, accessToken)
  sessionStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
}

export function clearTokens() {
  sessionStorage.removeItem(TOKEN_KEY)
  sessionStorage.removeItem(REFRESH_TOKEN_KEY)
  sessionStorage.removeItem("pkce_code_verifier")
}

export function isAuthenticated(): boolean {
  return !!getAccessToken()
}

export async function startAuthFlow() {
  const codeVerifier = generateRandomString(64)
  sessionStorage.setItem("pkce_code_verifier", codeVerifier)

  const challengeBuffer = await sha256(codeVerifier)
  const codeChallenge = base64UrlEncode(challengeBuffer)

  const state = generateRandomString(16)

  const params = new URLSearchParams({
    response_type: "code",
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    scope: "openid profile email phone",
    state,
    code_challenge: codeChallenge,
    code_challenge_method: "S256",
  })

  window.location.href = `/oidc/authorize?${params.toString()}`
}

export async function loginWithCredentials(
  username: string,
  password: string,
  requestId: string,
): Promise<string> {
  const res = await fetch("/oidc/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password, requestId }),
  })

  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error_description || "Login failed")
  }

  const data = await res.json()
  return data.redirectUri
}

export async function exchangeCodeForTokens(code: string) {
  const codeVerifier = sessionStorage.getItem("pkce_code_verifier")
  if (!codeVerifier) {
    throw new Error("Missing PKCE code verifier")
  }

  const params = new URLSearchParams({
    grant_type: "authorization_code",
    code,
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    code_verifier: codeVerifier,
  })

  const res = await fetch("/oidc/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params.toString(),
  })

  if (!res.ok) {
    const err = await res.json()
    throw new Error(err.error_description || "Token exchange failed")
  }

  const data = await res.json()
  setTokens(data.access_token, data.refresh_token)
  sessionStorage.removeItem("pkce_code_verifier")
}

export async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return false

  const params = new URLSearchParams({
    grant_type: "refresh_token",
    refresh_token: refreshToken,
    client_id: CLIENT_ID,
  })

  try {
    const res = await fetch("/oidc/token", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: params.toString(),
    })

    if (!res.ok) {
      clearTokens()
      return false
    }

    const data = await res.json()
    setTokens(data.access_token, data.refresh_token)
    return true
  } catch {
    clearTokens()
    return false
  }
}

export function logout() {
  clearTokens()
  window.location.href = "/login"
}
