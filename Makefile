# Makefile for psstat
-include $(wildcard Makefile.make)

BINARY  ?= psstat
OS      ?= linux
GOARCH  ?= amd64

COMMIT  ?= $(shell git rev-parse --short HEAD)
BRANCH  ?= $(shell git rev-parse --abbrev-ref HEAD)
VERSION ?= $(shell git describe 2>/dev/null)

PKG_NAME ?= "$(BINARY)"
PKG_DESCRIPTION ?= "Gather resource usage of processes for Telegraf"
PKG_BIN_DIR ?= "/usr/sbin/"
PKG_LICENSE ?= "MIT"
PKG_VENDOR ?= "unknown"
PKG_MAINTAINER ?= "$(shell git config user.name) <$(shell git config user.email)>"
PKG_URI ?= "https://github.com/fuegas/psstat"

# Add defines for commit, branch and version
LDFLAGS += -X main.commit=$(COMMIT) -X main.branch=$(BRANCH)
ifdef VERSION
	LDFLAGS += -X main.version=$(VERSION)
endif

# Remove DWARF tables
LDFLAGS += -s -w

# Base BUILDARCH on GOARCH
ifeq ($(GOARCH),386)
	BUILDARCH = i386
else
	BUILDARCH = $(GOARCH)
endif

.PHONY: all
all:
	$(MAKE) binary

.PHONY: binary
binary:
	@printf "Building... "
	@date
	@GOOS=$(OS) GO111MODULE=auto go build -v -o $(BINARY) -ldflags "$(LDFLAGS)" ./cmd/$(BINARY).go && echo "\033[32;1mBuild success ヽ(°□°)ﾉ\033[0m" || (echo "\033[31;1mBuild failed (╯°□°）╯︵ ┻━┻\033[0m" && exit 1)

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	go vet ./...

.PHONY: test
test:
	go test -short ./...

.PHONY: clean
clean:
	-rm -f $(BINARY)

.PHONY: clear
clear:
	@clear

.PHONY: watch
watch:
	@clear
	@echo "Watching current directory for changes"
	@fswatch --recursive --event Updated --one-per-batch ./*  | xargs -n1 -I{} make clear binary

.PHONY: deb
deb: binary
	@fpm \
		--input-type dir \
		--output-type deb \
		--deb-no-default-config-files \
		--force \
		--architecture $(BUILDARCH) \
		--description $(PKG_DESCRIPTION) \
		--license $(PKG_LICENSE) \
		--maintainer $(PKG_MAINTAINER) \
		--name "$(PKG_NAME)" \
		--url $(PKG_URI) \
		--vendor $(PKG_VENDOR) \
		--version "$(VERSION)" \
		$(BINARY)="$(PKG_BIN_DIR)"

.PHONY: rpm
rpm: binary
	@fpm \
		--input-type dir \
		--output-type rpm \
		--force \
		--architecture $(BUILDARCH) \
		--description "$(PKG_DESCRIPTION)" \
		--license $(PKG_LICENSE) \
		--maintainer $(PKG_MAINTAINER) \
		--name "$(PKG_NAME)" \
		--url $(PKG_URI) \
		--vendor $(PKG_VENDOR) \
		--version "$(VERSION)" \
		$(BINARY)="$(PKG_BIN_DIR)"
