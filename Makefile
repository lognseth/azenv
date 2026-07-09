BINARY := azenv
GO_CACHE_DIR := $(CURDIR)/.cache/go-build
GO_MOD_CACHE_DIR := $(CURDIR)/.cache/go-mod
GO_ENV := GOCACHE=$(GO_CACHE_DIR) GOMODCACHE=$(GO_MOD_CACHE_DIR)

.PHONY: build test install install-system clean

build:
	$(GO_ENV) go build -o $(BINARY)

test:
	$(GO_ENV) go test ./...

install:
	$(GO_ENV) ./install.sh

install-system:
	$(GO_ENV) ./install.sh --system

clean:
	rm -f $(BINARY)
	rm -rf dist coverage.out .cache
