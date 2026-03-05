# Authentication

## OIDC Flow

The OIDC flow based on the backend:
1. Frontend initiates → /oidc/authorize with PKCE params → redirects to /login?request_id=xxx
2. Login page posts username/password + requestId to POST /oidc/login → gets redirectUri with auth code
3. Frontend callback page receives code → exchanges via POST /oidc/token → gets access_token + refresh_token
4. Store tokens, attach to API requests
