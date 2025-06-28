# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that manages Plex Media Server preroll videos based on seasonal dates. It reads a YAML configuration file to determine which preroll videos should be active based on the current date.

## Development Commands

```bash
# Build the application
go build -o bin/plex-preroll ./cmd/plex-preroll

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run the application
go run ./cmd/plex-preroll

# Format code
go fmt ./...

# Run linter (if golangci-lint is installed)
golangci-lint run
```

## Code Architecture

The codebase follows standard Go project layout:

- **`cmd/plex-preroll/`**: Application entry point that loads configuration and prints active prerolls
- **`internal/`**: Core business logic
  - `config.go`: Configuration loading and preroll selection logic
  - `config_test.go`: Comprehensive unit tests

Key design patterns:
- Configuration is loaded from YAML files using `gopkg.in/yaml.v3`
- Date handling uses Go's `time` package with custom MM-DD parsing
- Seasons can overlap (e.g., Thanksgiving and Christmas) and cross year boundaries
- Results are sorted for consistent output

## Configuration

The application looks for configuration in:
1. Environment variable `PLEX_PREROLL_CFG` (if set)
2. Default: `~/.plex-preroll/config.yaml`

Configuration structure:
```yaml
seasons:
  <season_name>:
    from: MM-DD
    to: MM-DD
    files:
      - path/to/video1.mp4
      - path/to/video2.mp4
default:
  - path/to/default.mp4
```

## Testing

Tests are located in `internal/config_test.go` and cover:
- Configuration loading from YAML
- Date-based preroll selection
- Overlapping seasons
- Year-crossing seasons (e.g., Christmas: Nov 20 - Jan 6)
- Default fallback behavior

Test data is in `internal/testdata/prerolls_test.yaml`.