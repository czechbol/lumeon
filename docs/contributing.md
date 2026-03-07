# Contributing to lumEON

Contributions are welcome. This guide covers how to set up your environment, the code conventions to follow, and how to get a pull request merged.

## Table of contents

- [Prerequisites](#prerequisites)
- [Development setup](#development-setup)
- [Code style](#code-style)
- [Linting](#linting)
- [Testing](#testing)
- [Commit messages](#commit-messages)
- [Pull request process](#pull-request-process)
- [Release process](#release-process)

---

## Prerequisites

- **Go** — version specified in `go.mod` (currently 1.26)
- **golangci-lint** v2.11.1 — install via the [official instructions](https://golangci-lint.run/welcome/install/)
- **pre-commit** — install via `pip install pre-commit` or your OS package manager
- **goreleaser** (optional, for building release packages) — [goreleaser.com](https://goreleaser.com/install/)

---

## Development setup

```sh
git clone https://github.com/czechbol/lumeon.git
cd lumeon
go mod download
pre-commit install
```

`pre-commit install` sets up the git hooks defined in `.pre-commit-config.yaml`. They run automatically on each commit and check:

- No trailing whitespace
- Files end with a newline
- YAML files are valid and consistently formatted (via `yamlfmt`)
- No accidentally committed large files

You can also run all hooks manually at any time:

```sh
pre-commit run --all-files
```

---

## Code style

lumEON uses the standard Go formatting toolchain. The linter enforces these automatically, but the key rules are:

- **gofmt / goimports** — standard formatting and import ordering
- **golines** — lines are wrapped at 120 characters
- **Import grouping** — stdlib first, then external, then internal (`github.com/czechbol/lumeon/...`), separated by blank lines (enforced by `gci`)

There is no project-specific style guide beyond what the linter checks. When in doubt, follow the patterns in the existing code.

---

## Linting

```sh
golangci-lint run
```

The linter config is in `.golangci.yml`. It uses golangci-lint v2 with a large set of enabled linters covering correctness, style, security, and performance. Key things to be aware of:

- **`//nolint` directives** must include a reason: `//nolint:gosec // explanation here`. Directives without a reason are rejected by `nolintlint`.
- **Comments on exported identifiers** must end in a period (enforced by `godot`).
- **Error sentinel values** should be named `ErrXxx` and error types named `XxxError`.

CI runs the linter on every push and pull request. A PR will not be merged if lint fails.

---

## Testing

```sh
go test ./...
```

Tests use [testify](https://github.com/stretchr/testify) for assertions. Hardware tests use a mock i2c bus (`core/hardware/i2c/mock/`) so they work without real hardware.

When adding new hardware-facing code, add a corresponding mock and cover it with tests. The existing fan and OLED test suites in `core/hardware/` are good examples to follow.

CI runs `go test ./...` on every push and pull request against `main`.

---

## Commit messages

Use conventional commit format:

```
<type>: <short description>

[optional body]
```

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `ci`, `chore`.

Examples:

```
feat: add CPU temperature threshold to fan curve interpolation
fix: prevent display flicker on wake when ticker fires immediately
docs: add developer guide
```

Note: commits prefixed with `docs:` or `test:` are excluded from the generated changelog on releases.

---

## Pull request process

1. Fork the repository and create a feature branch off `main`.
2. Make your changes. Run `go test ./...` and `golangci-lint run` locally before pushing.
3. Open a pull request against `main`. Describe what changed and why.
4. CI will run tests and lint. Both must pass.
5. A maintainer will review and merge.

Keep pull requests focused. If you are fixing a bug and notice an unrelated cleanup opportunity, put it in a separate PR.

---

## Release process

Releases are handled by the maintainer using GoReleaser:

```sh
goreleaser release --clean
```

This builds the `lumeond` binary for `arm` and `arm64`, compresses it with UPX, and produces `.deb`, `.rpm`, `.apk`, and `.pkg.tar.zst` packages. Packages include the binary, the default `lumeon.toml` config, and the systemd service unit. The release is published to GitHub Releases automatically.

For a local snapshot build (no GitHub publish):

```sh
goreleaser build --snapshot --clean
```

Output lands in `dist/`.
