# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that manages Plex Media Server preroll videos based on seasonal dates. It reads a YAML configuration file to determine which preroll videos should be active based on the current date and updates the Plex server's preroll preferences via its REST API.

## Development Commands

```bash
# Build the application with version info
make build

# Build without make (no version info)
go build -o bin/plex-preroll ./cmd/plex-preroll

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal

# Run a specific test
go test ./internal -run TestGetActivePrerolls

# Run the application
go run ./cmd/plex-preroll

# Run with custom config
PLEX_PREROLL_CFG=/path/to/config.yaml go run ./cmd/plex-preroll

# Format code
go fmt ./...

# Run linter (if golangci-lint is installed)
golangci-lint run
```

## Code Architecture

The codebase follows standard Go project layout with clear separation of concerns:

### Entry Point (`cmd/plex-preroll/main.go`)
- Parses CLI flags (`--dry-run`, `--version`)
- Loads `.env` file for Plex credentials
- Determines config file location (env var or default)
- Orchestrates the preroll selection and Plex update flow

### Core Business Logic (`internal/`)
- **`config.go`**: YAML configuration loading and date-based preroll selection
  - Handles MM-DD date parsing with normalization
  - Supports overlapping seasons and year-crossing date ranges
  - Falls back to default prerolls when no season matches
- **`files.go`**: File system operations
  - Expands directories to individual video files
  - Validates that all paths are absolute
  - Skips invalid paths with warnings
- **`plexclient.go`**: Plex API client
  - Updates server preferences via REST API
  - Handles authentication with X-Plex-Token header

### Configuration Flow
1. Load environment variables from `.env` file (PLEX_SERVER_URL, PLEX_TOKEN)
2. Load YAML configuration for preroll definitions
3. Determine active prerolls based on current date
4. Expand any directory paths to individual files
5. Update Plex server with semicolon-separated preroll list

## Testing

Tests follow Go conventions with `*_test.go` files alongside source files:

- **`config_test.go`**: Comprehensive date-based selection testing
  - Table-driven tests for various date scenarios
  - Tests overlapping seasons (multiple active at once)
  - Tests year-crossing seasons (e.g., Nov 20 - Jan 6)
  - Tests default fallback behavior
- **`files_test.go`**: File system operation testing
  - Uses temporary directories for isolation
  - Tests directory expansion and path validation

Test data is organized in `internal/testdata/prerolls_test.yaml`.

## Configuration

### Environment Variables
Loaded from a `.env` file. The search order is:
1. Current working directory (`./.env`)
2. Platform-specific user config dir (e.g., `~/.config/plex-preroll/.env` on Linux, `~/Library/Application Support/plex-preroll/.env` on macOS, `%APPDATA%\plex-preroll\.env` on Windows)
3. Legacy path `~/.config/plex-preroll/.env`

Contents:
- `PLEX_SERVER_URL`: Plex server URL (default: `http://localhost:32400`)
- `PLEX_TOKEN`: Required authentication token (no default)
- `PLEX_PREROLL_CFG`: Override config file location (optional)

### YAML Configuration
Path determined as follows:
1. Value of `PLEX_PREROLL_CFG` environment variable, if set.
2. Otherwise, searches in platform-specific user config dir (e.g., `~/.config/plex-preroll/config.yaml` on Linux).
3. Then, legacy path `~/.config/plex-preroll/config.yaml`.

Structure:
```yaml
seasons:
  <season_name>:
    from: MM-DD  # e.g., "11-20" or "1-6"
    to: MM-DD
    files:
      - /absolute/path/to/video1.mp4
      - /absolute/path/to/directory/  # Will be expanded
default:
  - /absolute/path/to/default.mp4
```

Key behaviors:
- Dates use MM-DD format with automatic zero-padding
- Multiple seasons can be active simultaneously (files are merged and deduplicated).
- Seasons can cross year boundaries.
- Only absolute paths are accepted for files and directories.
- Directories are expanded to find video files:
    - Only files with recognized video extensions (e.g., `.mp4`, `.mkv`, `.mov`, etc.) are included.
    - Explicitly listed files are also checked against these extensions.
- If a configured directory is unreadable, the application will error and exit.