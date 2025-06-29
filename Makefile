VERSION := $(shell git describe --tags --always --dirty)

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/plex-preroll ./cmd/plex-preroll
