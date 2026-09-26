# Contributing

Open an issue for a bug or proposed change, or send a focused pull request. For bugs, include the operating system, Go version, application version or commit, steps to reproduce, and expected behavior. Remove tokens and private paths from configuration and logs. Report vulnerabilities using [SECURITY.md](SECURITY.md).

## Development

Install Go 1.26.0 or later, clone the repository, and run:

```bash
make all
go test -race -cover ./...
go vet ./...
```

Format changed Go files with `gofmt`. Add regression tests for behavior changes, including a command-level test when a bug involves setup or configuration loading. Use temporary files and HTTP test servers so tests do not depend on a Plex server, media library, or credentials.

Keep the README, sample configuration, and setup tool consistent with the shared configuration types. Tests run on Linux, macOS, and Windows; avoid hard-coded Unix paths when a temporary directory will work.

Describe what your pull request changes and how you checked it. Contributions are covered by the repository's [MIT license](LICENSE).
