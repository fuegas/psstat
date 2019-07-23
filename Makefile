# Makefile for psstat
BINARY := psstat
OS := linux

COMMIT := $(shell git rev-parse --short HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
VERSION := $(shell git describe --exact-match --tags 2>/dev/null)

# Add defines for commit, branch and version
LDFLAGS := $(LDFLAGS) -X main.commit=$(COMMIT) -X main.branch=$(BRANCH)
ifdef VERSION
	LDFLAGS += -X main.version=$(VERSION)
endif

.PHONY: all
all:
	$(MAKE) binary

.PHONY: binary
binary:
	@printf "Building... "
	@date
	@GOOS=$(OS) go build -v -i -o $(BINARY) -ldflags "$(LDFLAGS)" ./cmd/$(BINARY).go && echo "\033[32;1mBuild success ヽ(°□°)ﾉ\033[0m" || echo "\033[31;1mBuild failed (╯°□°）╯︵ ┻━┻\033[0m"

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	go vet ./...

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
