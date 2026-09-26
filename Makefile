VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

GOFILES_MAIN := $(shell find ./cmd/plex-preroll ./internal -type f -name '*.go')
GOFILES_SETUP := $(shell find ./cmd/plex-preroll-setup ./internal -type f -name '*.go')
BUILD_DEPS := go.mod go.sum Makefile

.PHONY: all build build-setup clean

all: build build-setup

build: bin/plex-preroll

build-setup: bin/plex-preroll-setup

bin/plex-preroll: $(GOFILES_MAIN) $(BUILD_DEPS)
	@mkdir -p bin
	go build -ldflags "-X main.version=$(VERSION)" -o $@ ./cmd/plex-preroll

bin/plex-preroll-setup: $(GOFILES_SETUP) $(BUILD_DEPS)
	@mkdir -p bin
	go build -o $@ ./cmd/plex-preroll-setup

clean:
	rm -f bin/plex-preroll bin/plex-preroll-setup
