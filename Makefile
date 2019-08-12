# Makefile for psstat
-include $(wildcard Makefile.make)

BINARY  ?= psstat
OS      ?= linux
GOARCH  ?= amd64

COMMIT  ?= $(shell git rev-parse --short HEAD)
BRANCH  ?= $(shell git rev-parse --abbrev-ref HEAD)
VERSION ?= $(shell git describe --abbrev=0 --match=HEAD 2>/dev/null)

# Add defines for commit, branch and version
LDFLAGS += -X main.commit=$(COMMIT) -X main.branch=$(BRANCH)
ifdef VERSION
	LDFLAGS += -X main.version=$(VERSION)
endif

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
	@GOOS=$(OS) go build -v -i -o $(BINARY) -ldflags "$(LDFLAGS)" ./cmd/$(BINARY).go && echo "\033[32;1mBuild success ヽ(°□°)ﾉ\033[0m" || (echo "\033[31;1mBuild failed (╯°□°）╯︵ ┻━┻\033[0m" && exit 1)

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
		--name psstat \
		--description "Gather resource usage of processes for Telegraf" \
		--version "$(VERSION)" \
		--deb-no-default-config-files \
		--architecture $(BUILDARCH) \
		--force \
		$(BINARY)=/usr/sbin/

.PHONY: rpm
rpm: binary
	@fpm \
		--input-type dir \
		--output-type rpm \
		--name psstat \
		--description "Gather resource usage of processes for Telegraf" \
		--version "$(VERSION)" \
		--deb-no-default-config-files \
		--architecture $(BUILDARCH) \
		--force \
		$(BINARY)=/usr/sbin/
