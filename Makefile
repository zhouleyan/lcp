PKG_PREFIX := lcp.io/lcp
APP_NAME := lcp-server
DATE_INFO_TAG ?= $(shell date -u +'%Y%m%d-%H%M%S')
BUILD_INFO_TAG ?= $(shell echo $$(git describe --long --all | tr '/' '-')$$( \
	      git diff-index --quiet HEAD -- || echo '-dirty-'$$(git diff-index -u HEAD | openssl sha1 | cut -d' ' -f2 | cut -c 1-8)))
RACE ?= -race
EXTRA_GO_BUILD_TAGS ?=
GO_BUILD_INFO = -X '$(PKG_PREFIX)/lib/buildinfo.Version=$(APP_NAME)-$(DATE_INFO_TAG)-$(BUILD_INFO_TAG)'

lcp-server:
	CGO_ENABLED=0 go build $(RACE) -ldflags "$(GO_BUILD_INFO)" -tags "$(EXTRA_GO_BUILD_TAGS)" -o bin/$(APP_NAME)$(RACE) $(PKG_PREFIX)/app/$(APP_NAME)
