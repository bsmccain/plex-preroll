# Repository guidance

This Go application updates Plex Media Server's preroll setting from seasonal YAML configuration. The minimum Go version is 1.26.0. Keep README examples, the setup tool, and the shared configuration types in agreement.

## Commands

```bash
make all
go test -race -cover ./...
go vet ./...
go fmt ./...
```

`make build` builds the main command with Git version information. `make build-setup` builds the setup command. Both use shared code in `internal`. Do not run a real Plex update while testing; use `--dry-run` or an HTTP test server.

## Code

- `cmd/plex-preroll` loads environment and YAML configuration, selects today's videos, validates files, and updates Plex unless `--dry-run` is set.
- `cmd/plex-preroll-setup` creates configuration interactively using the shared YAML types. Keep tokens hidden during terminal input and restrict credential-file permissions on Unix.
- `internal/config.go` loads configuration and selects matching inclusive date ranges. Equal dates represent one day; descending ranges cross New Year.
- `internal/files.go` reads configured directories without recursion and checks video file metadata without opening file contents. It then sorts and deduplicates video paths. Invalid or unavailable selected paths must fail before a Plex update.
- `internal/plexclient.go` sends the token in `X-Plex-Token`. Requests have a 30-second timeout and must not follow redirects.

The Plex value is a semicolon-separated list. Plex randomly chooses one video from this list. Reject comma and semicolon characters in video paths because they change Plex's interpretation.

## Configuration

`seasons` is a YAML list with `name`, `from`, `to`, and `files` fields. `default` is a list of paths used when no season matches. See [config/prerolls.yaml](config/prerolls.yaml).

Dates use `MM-DD`, with single-digit month and day input accepted and normalized. Validate actual calendar dates and permit February 29. Combine all matching seasons. A selected empty list means intentionally clearing prerolls; a directory containing no videos is an error.

Search `.env` in the working directory, the platform's user configuration directory under `plex-preroll`, then `~/.config/plex-preroll`. Existing environment variables take precedence. Malformed or unreadable discovered environment files must fail visibly.

`PLEX_TOKEN` is required. `PLEX_SERVER_URL` defaults to `http://localhost:32400`. `PLEX_PREROLL_CFG` overrides the YAML path and supports a leading `~/`; otherwise search for `config.yaml` in the user configuration directory and the home `.config` fallback. Video file paths remain absolute and must work for both this command and Plex.

## Tests and changes

Keep regression tests for credential handling, redirect rejection, date boundaries, missing media, and the complete setup-to-update flow. Command tests should execute the built commands with isolated configuration and mock Plex servers. Account for Windows path and permission differences.

Never commit tokens, real `.env` files, or private media paths. Use generic examples and follow [CONTRIBUTING.md](CONTRIBUTING.md). Report security issues using [SECURITY.md](SECURITY.md).
