# plex-preroll

Set Plex Media Server's preroll videos by season. Run the command manually or on a schedule; it selects videos for today's date and updates your server's Movie pre-roll video setting.

All matching seasons contribute videos. If no season matches, the command uses the default list. It sends a semicolon-separated list, so Plex chooses one video at random for each playback. See [Plex's preroll documentation](https://support.plex.tv/articles/202920803-extras/) for playback requirements.

## Install from source

You need Go 1.26.0 or later. Use a current Go release for its security fixes.

```bash
git clone https://github.com/bsmccain/plex-preroll.git
cd plex-preroll
make all
```

This builds `bin/plex-preroll` and the interactive `bin/plex-preroll-setup` tool. Without Make:

```bash
go build -o bin/plex-preroll ./cmd/plex-preroll
go build -o bin/plex-preroll-setup ./cmd/plex-preroll-setup
```

On Windows, use `.exe` output names, such as `bin/plex-preroll.exe`. The shell examples below use Bash.

## Set up

```bash
./bin/plex-preroll-setup
```

The setup tool asks for your Plex server URL and token, then creates a `.env` file and a YAML configuration. It suggests locations in your user configuration directory and asks before overwriting existing files. Token input is hidden in a terminal. On Unix, the `.env` file has owner-only `0600` permissions, including when replacing an existing file. On Windows, store it in your private user directory and use Windows file permissions to control access.

Get your token by following [Plex's authentication-token instructions](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/). Treat it as a password. Use an HTTPS server URL when sending it across an untrusted network.

Run a dry run before updating Plex:

```bash
./bin/plex-preroll --dry-run
./bin/plex-preroll
```

The command uses the local date and time zone of the machine running it. It must be able to inspect file metadata and list configured directories locally. Plex must be able to read the videos for playback. If Plex runs on another host or in a container, mount the videos at the same absolute paths for both programs. The command does not upload videos or translate paths.

## Configure manually

The application checks for `.env` in this order:

1. The current working directory.
2. The `plex-preroll` subdirectory of your operating system's user configuration directory.
3. `~/.config/plex-preroll/.env`, if different from the second location.

It loads the first file it finds. Existing environment variables take precedence over values from that file. A malformed `.env` file stops the command with an error.

The usual user configuration locations are:

| Platform | Configuration directory |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/plex-preroll`, or `~/.config/plex-preroll` |
| macOS | `~/Library/Application Support/plex-preroll` |
| Windows | `%APPDATA%\plex-preroll` |

Copy [.env.example](.env.example) to `.env` in one of those locations and replace the token placeholder:

```dotenv
PLEX_SERVER_URL=http://localhost:32400
PLEX_TOKEN=your-plex-token-here
# Optional override for the YAML file:
# PLEX_PREROLL_CFG=/absolute/path/to/config.yaml
```

On Unix, restrict a manually created credential file with `chmod 600 /path/to/.env`. Keep real tokens and local configuration out of version control.

`PLEX_TOKEN` is required, including for dry runs. `PLEX_SERVER_URL` defaults to `http://localhost:32400`.

Set `PLEX_PREROLL_CFG` to select a YAML file. A leading `~/` in this value expands to your home directory. Otherwise, the command searches for `config.yaml` in your user configuration directory, then `~/.config/plex-preroll/config.yaml`. It does not search the current directory for YAML unless you set `PLEX_PREROLL_CFG`.

### Seasons and videos

Use [config/prerolls.yaml](config/prerolls.yaml) as a starting point. Replace the sample paths with your own:

```yaml
seasons:
  - name: halloween
    from: "10-01"
    to: "10-31"
    files:
      - /media/prerolls/halloween/

  - name: thanksgiving
    from: "11-20"
    to: "11-30"
    files:
      - /media/prerolls/thanksgiving/turkeys.mp4

  - name: christmas
    from: "11-20"
    to: "01-06"
    files:
      - /media/prerolls/christmas/santa.mp4
      - /media/prerolls/christmas/snowman.mp4

default:
  - /media/prerolls/default/intro.mp4
```

`seasons` is a list. Each entry has a `name`, a `from` date, a `to` date, and a `files` list. The setup tool writes this same format. Unknown fields, extra YAML documents, and missing season file lists are rejected. Use an explicit empty list when you intend to clear prerolls.

- Dates use `MM-DD`. Single-digit months and days are also accepted. Dates must be valid calendar dates; February 29 is allowed.
- Both endpoints are inclusive. A season from `"12-25"` to `"12-25"` applies only on December 25.
- A start date after the end date crosses New Year, such as November 20 through January 6.
- Overlapping seasons combine their videos. The default list is used only when no season matches.
- Video paths must be absolute. Use forward slashes or single-quoted paths in Windows YAML, such as `'C:\Media\Prerolls\intro.mp4'`.
- Directories include video files directly inside them. The command does not recurse into subdirectories.
- Supported extensions are `.mp4`, `.mkv`, `.avi`, `.mov`, `.wmv`, `.ts`, `.m2ts`, `.flv`, `.asf`, `.3gp`, and `.webm`, case-insensitively.
- Commas and semicolons are not allowed in video paths because Plex treats them as separators.
- The final list is sorted and deduplicated.

A configured path that cannot be inspected, an unreadable directory, an unsupported explicit file, or a directory with no usable videos stops the update. Existing Plex settings remain unchanged. This also prevents a missing media mount from clearing your prerolls.

An empty selected list explicitly clears Plex's preroll setting. For example, `default: []` clears it on dates when no season matches. A matching season with `files: []` contributes no videos and does not trigger the default list.

## Run on a schedule

The command runs once and exits. On Linux or macOS, place `.env` and `config.yaml` in your user configuration directory, then add a cron entry with the full binary path:

```cron
# Update daily at 4 AM. Create the log directory first.
0 4 * * * /absolute/path/to/bin/plex-preroll >> /absolute/path/to/logs/plex-preroll.log 2>&1
```

Run the job as the user who owns the configuration and can inspect media paths and list configured directories. Windows users can schedule `plex-preroll.exe` with Task Scheduler under that user account.

Other commands:

```bash
./bin/plex-preroll --version
PLEX_PREROLL_CFG=/absolute/path/to/config.yaml ./bin/plex-preroll --dry-run
```

## Troubleshooting

If the token is missing, check the `.env` search locations and the environment of the user running the command.

If loading YAML fails, check that `seasons` is a list with `- name:` entries, and that dates are valid. Quote dates as shown above.

If file validation fails, check the selected paths, permissions, and media mounts. Run `--dry-run` as the same user as your scheduled job. A directory containing only subdirectories or unsupported files is not a valid video source.

If updating Plex fails, check the server URL, token, and network connection. Requests have a 30-second timeout and do not follow redirects. When using a proxy, configure the final server URL directly.

If Plex accepts the update but does not play a preroll, check that Plex can read those exact file paths. Enable Cinema Trailers in the Plex app and the library's advanced settings. Client support varies; see [Plex's playback requirements](https://support.plex.tv/articles/202920803-extras/).

## Development

```bash
make all
go test -race -cover ./...
go vet ./...
```

CI tests both commands and the shared packages on Linux, macOS, and Windows. Linux also tests the minimum supported Go version. A separate job checks for reachable known vulnerabilities with `govulncheck` on current stable Go.

See [CONTRIBUTING.md](CONTRIBUTING.md) for change and test guidance, and [SECURITY.md](SECURITY.md) to report vulnerabilities privately.

## Credits

Inspired by [Plex-Automatic-Preroll](https://github.com/Nacer27/Plex-Automatic-Preroll), a Python script that rotates Plex prerolls by month. plex-preroll is an independent Go implementation and shares no code with it.

## License

[MIT](LICENSE), copyright Brian McCain.

