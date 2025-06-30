VERSION := $(shell git describe --tags --always --dirty)

# Find all .go files in the project, excluding vendor directories.
GOFILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
# Specifically find .go files for the setup tool
GOFILES_SETUP := $(shell find ./cmd/plex-preroll-setup -type f -name '*.go') $(shell find ./internal -type f -name '*.go' -not -path "./vendor/*")


.PHONY: all build build-setup clean

all: build build-setup

build: bin/plex-preroll
	@echo "plex-preroll binary is up to date."

build-setup: bin/plex-preroll-setup
	@echo "plex-preroll-setup binary is up to date."

bin/plex-preroll: $(filter-out $(GOFILES_SETUP),$(GOFILES))
	@echo "Building plex-preroll..."
	@mkdir -p bin
	@go build -ldflags "-X main.version=$(VERSION)" -o $@ ./cmd/plex-preroll

bin/plex-preroll-setup: $(GOFILES_SETUP)
	@echo "Building plex-preroll-setup..."
	@mkdir -p bin
	@go build -o $@ ./cmd/plex-preroll-setup

clean:
	@echo "Cleaning..."
	@rm -f bin/plex-preroll bin/plex-preroll-setup
