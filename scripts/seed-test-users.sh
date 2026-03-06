#!/usr/bin/env bash
# Seed test users for pagination testing via API.
# Usage: ./scripts/seed-test-users.sh [count] [base_url]
#   count    - number of users to create (default: 55)
#   base_url - server base URL (default: http://localhost:8428)
#
# Logs in as admin (Admin123!) to get a session, then creates users via API.

set -euo pipefail

COUNT=${1:-55}
BASE_URL=${2:-"http://localhost:8428"}
API_BASE="${BASE_URL}/api/iam/v1"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-Admin123!}"

# --- Step 1: Login to get a session token ---
echo "Authenticating as ${ADMIN_USER}..."

LOGIN_RESP=$(curl -s -X POST "${BASE_URL}/oidc/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}")

# Direct login returns sessionId + userId
USER_ID=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('userId',''))" 2>/dev/null || true)

if [ -z "$USER_ID" ]; then
  echo "Login failed: $LOGIN_RESP"
  exit 1
fi

# For direct login without OIDC flow, we need a token.
# Use the OIDC authorization code flow programmatically.

# Step 1: Start authorize
AUTH_URL="${BASE_URL}/oidc/authorize?response_type=code&client_id=lcp-ui&scope=openid+profile+email+phone&state=seed&code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&code_challenge_method=S256"

# Follow redirect to get request_id from login page URL
REDIRECT=$(curl -s -o /dev/null -w "%{redirect_url}" "$AUTH_URL")
REQUEST_ID=$(echo "$REDIRECT" | python3 -c "import sys; from urllib.parse import urlparse, parse_qs; print(parse_qs(urlparse(sys.stdin.read().strip()).query).get('request_id',[''])[0])")

if [ -z "$REQUEST_ID" ]; then
  echo "Failed to get request_id from authorize redirect"
  exit 1
fi

# Step 2: Login with request_id to get authorization code
LOGIN_RESP=$(curl -s -X POST "${BASE_URL}/oidc/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\",\"requestId\":\"${REQUEST_ID}\"}")

REDIRECT_URI=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('redirectUri',''))" 2>/dev/null || true)

if [ -z "$REDIRECT_URI" ]; then
  echo "Login with requestId failed: $LOGIN_RESP"
  exit 1
fi

AUTH_CODE=$(echo "$REDIRECT_URI" | python3 -c "import sys; from urllib.parse import urlparse, parse_qs; print(parse_qs(urlparse(sys.stdin.read().strip()).query).get('code',[''])[0])")

# Step 3: Exchange code for tokens
# The code_verifier that produces challenge E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM is "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
TOKEN_RESP=$(curl -s -X POST "${BASE_URL}/oidc/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=${AUTH_CODE}&client_id=lcp-ui&code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")

ACCESS_TOKEN=$(echo "$TOKEN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || true)

if [ -z "$ACCESS_TOKEN" ]; then
  echo "Token exchange failed: $TOKEN_RESP"
  exit 1
fi

echo "Authenticated. Token obtained."

# --- Step 2: Create test users ---
FIRST_NAMES=(
  "alice" "bob" "charlie" "david" "emma" "frank" "grace" "henry" "iris" "jack"
  "kate" "leo" "mia" "noah" "olivia" "peter" "quinn" "rose" "sam" "tina"
  "uma" "victor" "wendy" "xander" "yuki" "zara" "adam" "bella" "carl" "diana"
  "ethan" "fiona" "george" "hannah" "ivan" "julia" "kevin" "luna" "mark" "nora"
  "oscar" "penny" "ray" "stella" "tom" "ursula" "vince" "willow" "xenia" "yara"
  "zane" "amber" "brian" "chloe" "derek" "elena" "felix" "gina" "hugo" "isla"
)

DISPLAY_NAMES=(
  "Alice Wang" "Bob Li" "Charlie Zhang" "David Chen" "Emma Liu" "Frank Yang" "Grace Huang" "Henry Wu" "Iris Zhou" "Jack Xu"
  "Kate Sun" "Leo Ma" "Mia Zhu" "Noah Hu" "Olivia Guo" "Peter He" "Quinn Lin" "Rose Luo" "Sam Zheng" "Tina Liang"
  "Uma Song" "Victor Xie" "Wendy Tang" "Xander Han" "Yuki Cao" "Zara Xu" "Adam Deng" "Bella Feng" "Carl Jiang" "Diana Yu"
  "Ethan Dong" "Fiona Xiao" "George Ye" "Hannah Pan" "Ivan Cheng" "Julia Su" "Kevin Fan" "Luna Ren" "Mark Wei" "Nora Fang"
  "Oscar Shi" "Penny Yao" "Ray Qian" "Stella Dai" "Tom Qiu" "Ursula Yin" "Vince Zou" "Willow Peng" "Xenia Bai" "Yara Meng"
  "Zane Xiong" "Amber Jin" "Brian Hou" "Chloe Gong" "Derek Shao" "Elena Wan" "Felix Tao" "Gina Lei" "Hugo Long" "Isla Hao"
)

echo "Seeding ${COUNT} test users..."

created=0
skipped=0
for i in $(seq 1 "$COUNT"); do
  idx=$(( (i - 1) % ${#FIRST_NAMES[@]} ))
  suffix=$(printf "%03d" "$i")
  username="${FIRST_NAMES[$idx]}_test_${suffix}"
  email="${FIRST_NAMES[$idx]}${suffix}@test.example.com"
  display_name="${DISPLAY_NAMES[$idx]} ${suffix}"
  phone="138$(printf "%08d" "$i")"
  status="active"
  if (( i % 7 == 0 )); then
    status="inactive"
  fi

  http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/users" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -d "{
      \"metadata\": {},
      \"spec\": {
        \"username\": \"${username}\",
        \"email\": \"${email}\",
        \"displayName\": \"${display_name}\",
        \"phone\": \"${phone}\",
        \"password\": \"Test1234\",
        \"status\": \"${status}\"
      }
    }")

  if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
    created=$((created + 1))
  elif [ "$http_code" = "409" ]; then
    skipped=$((skipped + 1))
  else
    echo "  [${http_code}] Failed: ${username}"
  fi

  # Progress indicator every 10 users
  if (( i % 10 == 0 )); then
    echo "  Progress: ${i}/${COUNT}"
  fi
done

echo "Done. Created: ${created}, Skipped (already exists): ${skipped}"
