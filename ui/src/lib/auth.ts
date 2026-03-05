const CLIENT_ID = "lcp-ui"
const TOKEN_KEY = "lcp_access_token"
const REFRESH_TOKEN_KEY = "lcp_refresh_token"

function generateRandomString(length: number): string {
  const array = new Uint8Array(length)
  crypto.getRandomValues(array)
  return Array.from(array, (b) => b.toString(16).padStart(2, "0")).join("")
}

async function sha256(plain: string): Promise<ArrayBuffer> {
  const encoder = new TextEncoder()
  const data = encoder.encode(plain)

  // crypto.subtle is only available in secure contexts (HTTPS or localhost)
  if (crypto.subtle) {
    return crypto.subtle.digest("SHA-256", data)
  }

  // Fallback: pure JS SHA-256 for non-secure contexts (e.g. HTTP with IP access)
  return sha256Fallback(data)
}

function sha256Fallback(data: Uint8Array): ArrayBuffer {
  const K: number[] = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1,
    0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786,
    0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147,
    0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
    0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a,
    0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ]

  const rotr = (n: number, x: number) => (x >>> n) | (x << (32 - n))
  const ch = (x: number, y: number, z: number) => (x & y) ^ (~x & z)
  const maj = (x: number, y: number, z: number) => (x & y) ^ (x & z) ^ (y & z)
  const sigma0 = (x: number) => rotr(2, x) ^ rotr(13, x) ^ rotr(22, x)
  const sigma1 = (x: number) => rotr(6, x) ^ rotr(11, x) ^ rotr(25, x)
  const gamma0 = (x: number) => rotr(7, x) ^ rotr(18, x) ^ (x >>> 3)
  const gamma1 = (x: number) => rotr(17, x) ^ rotr(19, x) ^ (x >>> 10)

  // Pre-processing: padding
  const bitLen = data.length * 8
  const padded = new Uint8Array(
    Math.ceil((data.length + 9) / 64) * 64,
  )
  padded.set(data)
  padded[data.length] = 0x80
  const view = new DataView(padded.buffer)
  view.setUint32(padded.length - 4, bitLen, false)

  let [h0, h1, h2, h3, h4, h5, h6, h7] = [
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
    0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
  ]

  for (let offset = 0; offset < padded.length; offset += 64) {
    const W = new Array<number>(64)
    for (let t = 0; t < 16; t++) W[t] = view.getUint32(offset + t * 4, false)
    for (let t = 16; t < 64; t++)
      W[t] = (gamma1(W[t - 2]) + W[t - 7] + gamma0(W[t - 15]) + W[t - 16]) | 0

    let [a, b, c, d, e, f, g, h] = [h0, h1, h2, h3, h4, h5, h6, h7]
    for (let t = 0; t < 64; t++) {
      const T1 = (h + sigma1(e) + ch(e, f, g) + K[t] + W[t]) | 0
      const T2 = (sigma0(a) + maj(a, b, c)) | 0
      h = g; g = f; f = e; e = (d + T1) | 0
      d = c; c = b; b = a; a = (T1 + T2) | 0
    }
    h0 = (h0 + a) | 0; h1 = (h1 + b) | 0; h2 = (h2 + c) | 0; h3 = (h3 + d) | 0
    h4 = (h4 + e) | 0; h5 = (h5 + f) | 0; h6 = (h6 + g) | 0; h7 = (h7 + h) | 0
  }

  const result = new ArrayBuffer(32)
  const out = new DataView(result)
  ;[h0, h1, h2, h3, h4, h5, h6, h7].forEach((v, i) => out.setUint32(i * 4, v, false))
  return result
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
