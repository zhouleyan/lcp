# ---- Stage 1: Build frontend ----
FROM node:22-alpine AS ui-builder

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /build/ui
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY ui/ ./
RUN pnpm build

# ---- Stage 2: Build Go binary ----
FROM golang:1.26-alpine AS go-builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate OpenAPI spec for go:embed (docs/openapi.json)
RUN go run lcp.io/lcp/cmd/openapi-gen -apis-dir pkg/apis -output docs/openapi.json -format json

# Copy frontend build output for go:embed (ui/dist)
COPY --from=ui-builder /build/ui/dist ./ui/dist

ARG VERSION=""
RUN CGO_ENABLED=0 go build \
    -ldflags "-X 'lcp.io/lcp/lib/buildinfo.Version=${VERSION}'" \
    -o /out/lcp-server \
    lcp.io/lcp/app/lcp-server

# ---- Stage 3: Runtime ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=go-builder /out/lcp-server .
COPY app/lcp-server/config.yaml ./config.yaml

EXPOSE 8428

ENTRYPOINT ["./lcp-server"]
CMD ["-config", "/app/config.yaml"]
