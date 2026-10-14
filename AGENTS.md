# Repository Guidelines

## Project Structure & Module Organization

`ipinfoq` is a Go CLI for downloading and querying IPinfo Lite data. All Go files live at the repository root in `package main`:

- `main.go`: entry point, environment configuration, and Cobra commands.
- `dataset.go`: downloads, filtering, and CIDR merging.
- `gzip.go`: generic gzip JSON readers and writers.
- `main_test.go` and `dataset_test.go`: CLI and dataset tests with inline fixtures.

`.github/workflows/ci.yml` defines CI and tagged releases; `.bruno/` contains a manual download request. `README.md` documents CLI usage.

## Build, Test, and Development Commands

Use Go 1.26.1 or later, matching `go.mod`.

- `go build -o ipinfoq .`: build the local binary.
- `go run . --help`: inspect commands without installing.
- `go test ./...`: run all tests.
- `gofmt -w *.go`: format Go sources.
- `mise exec -- golangci-lint run`: run the configured linter using `mise.toml`.

CI also checks spelling with `typos`.

## Coding Style & Naming Conventions

Use Go formatting with tabs, UTF-8, LF endings, and a final newline. YAML uses two-space indentation. Follow `.editorconfig` and `.golangci.yml`. Use PascalCase for exported Go identifiers and camelCase for unexported ones; keep initialisms such as `IP`, `ASN`, and `URL` uppercase.

Keep the CLI in `package main`; write new Cobra command definitions manually in `cmd.go`. Use official Go tooling for dependency changes. Separate logical steps with blank lines; keep operations, error checks, and cleanup together. Separate control-flow statements from preceding operations. Comment non-obvious behavior and interface contracts, including inputs and expected errors.

## Testing Guidelines

Use Go's `testing` package with `testify/assert` and `testify/require`. Prefer table-driven subtests in `*_test.go`, with functions named `TestBehavior`. Separate setup, execution, and assertions. Use `t.TempDir()`, `httptest`, and `t.Context()` to isolate tests. If mocks are needed, generate them with `moq` as `mock<interface>_test.go`, using a lowercase interface name without separators.

Run `go test -v -covermode count -coverprofile profile.cov ./...` to reproduce CI coverage reporting. No minimum coverage percentage is configured.

## Commit & Pull Request Guidelines

Use short, imperative commit subjects, matching history: `Add version command` or `Add --merge-cidr flag`. Keep commits focused. PRs should explain the behavior change, link relevant issues, and report validation results. Update README examples when flags or output change.

## Security & Configuration

Pass tokens through `--token`, for example `go run . download -t "$IPINFOQ_TOKEN"`. Keep credentials and downloaded datasets out of commits. Data is cached under `os.UserCacheDir()/ipinfoq/ipinfo_lite.gz`.

Downloads retain both IP versions; `query --ipv` filters results. The API's `Expires` timestamp is stored in the gzip header's comment. An unexpired timestamp makes `download` succeed silently without an HTTP request; expired, missing, or invalid expiration metadata triggers a new download.
