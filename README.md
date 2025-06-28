# plex-preroll

A Go application for managing seasonal preroll videos in Plex Media Server. Automatically selects which preroll videos should be active based on the current date and your configured seasons.

## Features

- Define custom seasons with date ranges (e.g., Halloween, Christmas, Summer)
- Support for overlapping seasons
- Handles seasons that cross year boundaries
- Falls back to default prerolls when no season matches
- Simple YAML configuration

## Installation

### From Source

Requires Go 1.24.4 or later:

```bash
git clone https://github.com/bsmccain/plex-preroll.git
cd plex-preroll
go build -o bin/plex-preroll ./cmd/plex-preroll
```

### Pre-built Binary

If available, you can use the pre-built binary in the `bin/` directory.

## Configuration

Create a configuration file at `~/.plex-preroll/config.yaml` or set the `PLEX_PREROLL_CFG` environment variable to point to your config file.

### Example Configuration

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
      - /media/prerolls/thanksgiving/turkey.mp4
  
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
      - /media/prerolls/summer/beach.mp4

default:
  - /media/prerolls/default/movie_trivia.mp4
  - /media/prerolls/default/coming_soon.mp4
```

### Configuration Format

- **seasons**: Named time periods with:
  - `from`: Start date in MM-DD format
  - `to`: End date in MM-DD format
  - `files`: List of video file paths to use as prerolls
- **default**: List of video files to use when no season matches

Note: Seasons can overlap (both will be included) and can cross year boundaries.

## Usage

Run the application to see which prerolls are currently active:

```bash
# Using the compiled binary
./bin/plex-preroll

# Or using go run
go run ./cmd/plex-preroll

# With a custom config file
PLEX_PREROLL_CFG=/path/to/config.yaml ./bin/plex-preroll
```

The application will output the paths of all preroll videos that should be active based on the current date.

### Example Output

If run on December 25th with the example configuration:
```
/media/prerolls/christmas/santa.mp4
/media/prerolls/christmas/snowman.mp4
```

## Integration with Plex

To use the selected prerolls in Plex Media Server:

1. Run this application to get the current preroll list
2. Update Plex's preroll settings with the output
3. Consider setting up a cron job or scheduled task to update automatically

Note: This application only determines which prerolls should be active. You'll need to integrate it with Plex's API or configuration to actually update the preroll settings.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...
```

### Project Structure

```
├── cmd/plex-preroll/     # Application entry point
├── internal/             # Core business logic
│   ├── config.go         # Configuration and preroll selection
│   └── config_test.go    # Unit tests
├── config/               # Example configuration
└── bin/                  # Compiled binaries
```

## License

[Add your license here]

## Contributing

[Add contribution guidelines if applicable]