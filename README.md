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

The application looks for a `.env` file in these locations (in order):
1. Current working directory
2. `~/.config/plex-preroll/.env` (recommended for automated runs)

Create a `.env` file with:

```bash
# Required
PLEX_TOKEN=your-plex-token-here

# Optional (defaults shown)
PLEX_SERVER_URL=http://localhost:32400
PLEX_PREROLL_CFG=~/.config/plex-preroll/config.yaml
```

To get your Plex token, see the [Plex support article](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/).

**Tip**: For cron jobs, place the `.env` file at `~/.config/plex-preroll/.env` so the application can find it automatically without needing to set environment variables in your crontab.

### Preroll Configuration

Create a configuration file at `~/.config/plex-preroll/config.yaml` (or specify a custom location with `PLEX_PREROLL_CFG`):

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
- All file paths must be absolute paths
- Directories in the files list will be expanded to include all video files

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

3. **"No video files found in directory"**
   - Ensure the directory exists and contains video files
   - Check file permissions

4. **Prerolls not showing in Plex**
   - Verify the video files are accessible to the Plex server
   - Check Plex server logs for any errors
   - Ensure preroll feature is enabled in Plex settings

## License

[Add your license here]

## Contributing

[Add contribution guidelines if applicable]