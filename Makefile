PKG_PREFIX := lcp.io/lcp
APP_NAME := lcp-server
DATE_INFO_TAG ?= $(shell date -u +'%Y%m%d-%H%M%S')
BUILD_INFO_TAG ?= $(shell echo $$(git describe --long --all | tr '/' '-')$$( \
	      git diff-index --quiet HEAD -- || echo '-dirty-'$$(git diff-index -u HEAD | openssl sha1 | cut -d' ' -f2 | cut -c 1-8)))
RACE ?= -race
EXTRA_GO_BUILD_TAGS ?=
GO_BUILD_INFO = -X '$(PKG_PREFIX)/lib/buildinfo.Version=$(APP_NAME)-$(DATE_INFO_TAG)-$(BUILD_INFO_TAG)'

.PHONY: lcp-server lcp-server-prod sqlc-generate openapi-gen test lint fmt vet clean ui-install ui-dev ui-build ui-lint

lcp-server:
	CGO_ENABLED=1 go build $(RACE) -ldflags "$(GO_BUILD_INFO)" -tags "$(EXTRA_GO_BUILD_TAGS)" -o bin/$(APP_NAME)$(RACE) $(PKG_PREFIX)/app/$(APP_NAME)

lcp-server-prod:
	CGO_ENABLED=0 go build -ldflags "$(GO_BUILD_INFO)" -tags "$(EXTRA_GO_BUILD_TAGS)" -o bin/$(APP_NAME) $(PKG_PREFIX)/app/$(APP_NAME)

sqlc-generate:
	cd pkg/db && sqlc generate

openapi-gen:
	go run $(PKG_PREFIX)/cmd/openapi-gen -apis-dir pkg/apis -output docs/openapi.json -format json
	go run $(PKG_PREFIX)/cmd/openapi-gen -apis-dir pkg/apis -output docs/openapi.yaml -format yaml

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w -s .

clean:
	rm -rf bin/

ui-install:
	cd ui && pnpm install

ui-dev:
	cd ui && pnpm dev

ui-build:
	cd ui && pnpm build

ui-lint:
	cd ui && pnpm lint
