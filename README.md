# plex-preroll

A Go application for managing seasonal preroll videos in Plex Media Server. Automatically selects which preroll videos should be active based on the current date and updates your Plex server's preroll settings.

## Features

- Automatically updates Plex server preroll settings based on date
- Define custom seasons with date ranges (e.g., Halloween, Christmas, Summer)
- Support for overlapping seasons
- Handles seasons that cross year boundaries
- Falls back to default prerolls when no season matches
- Directory expansion - specify a directory and all video files will be included
- Dry-run mode for testing without updating Plex
- Simple YAML configuration

## Installation

### From Source

Requires Go 1.20 or later:

```bash
git clone https://github.com/yourusername/plex-preroll.git
cd plex-preroll
make build
```

This will create a binary at `bin/plex-preroll` with version information included.

### Pre-built Binary

Download the latest release from the releases page.

## Configuration

### Environment Variables

The application looks for a `.env` file in the following locations, using the first one it finds:
1. Current working directory (`./.env`). This is useful for project-specific overrides.
2. Platform-specific user configuration directory:
    - Linux: `$XDG_CONFIG_HOME/plex-preroll/.env` or `~/.config/plex-preroll/.env`
    - macOS: `~/Library/Application Support/plex-preroll/.env`
    - Windows: `%APPDATA%\plex-preroll\.env`
3. Default path: `~/.config/plex-preroll/.env` (if different from the platform-specific user config directory).

Create a `.env` file in one of these locations with:

```bash
# Required
PLEX_TOKEN=your-plex-token-here

# Optional (defaults shown)
PLEX_SERVER_URL=http://localhost:32400
PLEX_PREROLL_CFG=~/.config/plex-preroll/config.yaml
```

To get your Plex token, see the [Plex support article](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/).

**Tip**: For automated runs (like cron jobs), placing the `.env` file in your user configuration directory (e.g., `~/.config/plex-preroll/.env` on Linux) is recommended.

### Preroll Configuration

The application loads its preroll definitions from a YAML configuration file.
If the `PLEX_PREROLL_CFG` environment variable is set, it will use the path specified. Otherwise, it looks for `config.yaml` in the following locations, using the first one it finds:
1. Platform-specific user configuration directory (inside a `plex-preroll` subdirectory, same as `.env` file):
    - Linux: `$XDG_CONFIG_HOME/plex-preroll/config.yaml` or `~/.config/plex-preroll/config.yaml`
    - macOS: `~/Library/Application Support/plex-preroll/config.yaml`
    - Windows: `%APPDATA%\plex-preroll\config.yaml`
2. Default path: `~/.config/plex-preroll/config.yaml` (if different from the platform-specific user config directory).

Example `config.yaml`:

```yaml
seasons:
  halloween:
    from: 10-01
    to: 11-01
    files:
      - /media/prerolls/halloween/spooky1.mp4
      - /media/prerolls/halloween/spooky2.mp4
  
  thanksgiving:
    from: 11-20
    to: 11-30
    files:
      - /media/prerolls/thanksgiving/
  
  christmas:
    from: 11-20
    to: 01-06
    files:
      - /media/prerolls/christmas/santa.mp4
      - /media/prerolls/christmas/snowman.mp4
  
  summer:
    from: 06-01
    to: 08-31
    files:
      - /media/prerolls/summer/

default:
  - /media/prerolls/default/movie_trivia.mp4
  - /media/prerolls/default/coming_soon.mp4
```

### Configuration Format

- **seasons**: Named time periods with:
  - `from`: Start date in MM-DD format
  - `to`: End date in MM-DD format
  - `files`: List of video file paths or directories (directories will be expanded to include all video files)
- **default**: List of video files to use when no season matches

Notes:
- Seasons can overlap (all matching seasons will be included)
- Seasons can cross year boundaries (e.g., Christmas from Nov 20 to Jan 6)
- All file paths must be absolute paths.
- Directories in the `files` list will be expanded. Only files with recognized video extensions (e.g., `.mp4`, `.mkv`, `.mov`, `.avi`, `.ts`, `.webm`, etc.) will be included from these directories. Explicitly listed files are also checked against these extensions.
- If a configured directory cannot be read (e.g., due to permissions), the application will log an error and exit.

## Usage

### Manual Execution

```bash
# Run and update Plex
./bin/plex-preroll

# Dry run - show what would be set without updating Plex
./bin/plex-preroll --dry-run

# Show version
./bin/plex-preroll --version

# With custom config file
PLEX_PREROLL_CFG=/path/to/config.yaml ./bin/plex-preroll
```

### Automated Execution with Cron

The intended use is to run this automatically via cron. Add to your crontab:

```bash
# Run daily at 4 AM
0 4 * * * /path/to/plex-preroll >> /path/to/logs/plex-preroll.log 2>&1

# Run every 6 hours
0 */6 * * * /path/to/plex-preroll >> /path/to/logs/plex-preroll.log 2>&1
```

**Recommended setup**: Place your `.env` file at `~/.config/plex-preroll/.env` and the application will automatically load it, eliminating the need to set environment variables in your crontab.

If you prefer to set environment variables directly in crontab:

```bash
# Crontab with environment variables
PLEX_TOKEN=your-token-here
PLEX_SERVER_URL=http://localhost:32400
0 4 * * * /path/to/plex-preroll >> /path/to/logs/plex-preroll.log 2>&1
```

### Example Output

When run on December 25th with the example configuration:
```
2023-12-25 04:00:01 Active prerolls for today:
  - /media/prerolls/christmas/santa.mp4
  - /media/prerolls/christmas/snowman.mp4
2023-12-25 04:00:01 Successfully updated Plex preroll settings
```

## Development

### Prerequisites

- Go 1.20 or later
- Make (optional, for using Makefile)

### Building

```bash
# Build with version info
make build

# Build without make
go build -o bin/plex-preroll ./cmd/plex-preroll
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test ./internal -run TestGetActivePrerolls
```

### Project Structure

```
├── cmd/plex-preroll/     # Application entry point
├── internal/             # Core business logic
│   ├── config.go         # Configuration and preroll selection
│   ├── config_test.go    # Configuration tests
│   ├── files.go          # File system operations
│   ├── files_test.go     # File system tests
│   ├── plexclient.go     # Plex API client
│   └── testdata/         # Test fixtures
├── .env.example          # Example environment configuration
├── Makefile              # Build automation
└── bin/                  # Compiled binaries
```

## Troubleshooting

### Common Issues

1. **"PLEX_TOKEN environment variable is not set"**
   - Create a `.env` file with your Plex token or set the environment variable

2. **"Failed to update Plex preroll settings"**
   - Verify your Plex server URL is correct
   - Ensure your Plex token has the necessary permissions
   - Check that Plex server is running and accessible

3. **"No video files found in directory" or "Specified file ... is not a recognized video file"**
   - Ensure the directory exists and contains files with common video extensions (e.g., `.mp4`, `.mkv`).
   - Check that explicitly listed files also have recognized video extensions.
   - Verify file and directory permissions.

4. **"Failed to read configured directory..."**
   - This indicates a permission issue or other problem accessing a directory listed in your `config.yaml`. Ensure the application has rights to list files in that directory.

5. **Prerolls not showing in Plex**
   - Verify the video files are accessible to the Plex server
   - Check Plex server logs for any errors
   - Ensure preroll feature is enabled in Plex settings

## License

[Add your license here]

## Contributing

[Add contribution guidelines if applicable]