# ---- Stage 1: Build frontend ----
# Skipped when PREBUILT_UI=true (uses local ui/dist from build context)
FROM node:20-alpine AS ui-builder

ARG PREBUILT_UI=false

WORKDIR /app/ui
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN if [ "$PREBUILT_UI" = "false" ]; then corepack enable && pnpm install --frozen-lockfile; fi
COPY ui/ .
RUN if [ "$PREBUILT_UI" = "false" ]; then pnpm build; fi

# ---- Stage 2: Build Go binary ----
FROM golang:1.26-alpine AS go-builder

RUN apk add --no-cache git tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui-builder /app/ui/dist ui/dist

# Generate OpenAPI spec for go:embed
RUN go run lcp.io/lcp/cmd/openapi-gen -apis-dir pkg/apis -output app/lcp-server/apis/openapi.json -format json

ARG VERSION=""
RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w -X 'lcp.io/lcp/lib/buildinfo.Version=${VERSION}'" \
    -o /out/lcp-server \
    lcp.io/lcp/app/lcp-server

# ---- Stage 3: Runtime ----
FROM scratch

COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=go-builder /out/lcp-server /app/lcp-server

EXPOSE 8428

ENTRYPOINT ["/app/lcp-server"]
