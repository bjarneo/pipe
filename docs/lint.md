# Linting

Pipe uses [golangci-lint](https://golangci-lint.run/) for code quality and consistency.

## Quick Start

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Or use make
make lint
```

## Git Hooks

Install pre-commit hooks to automatically run format, lint, and tests before each commit:

```bash
make install-hooks
```

This installs a pre-commit hook that:
1. Checks code formatting
2. Runs golangci-lint
3. Runs all tests

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make` | Run fmt, lint, test, and build |
| `make fmt` | Format code with gofmt |
| `make lint` | Run golangci-lint |
| `make test` | Run all tests |
| `make build` | Build the binary |
| `make check` | Run fmt, lint, and test |
| `make install-hooks` | Install git pre-commit hook |

## Enabled Linters

| Linter | Description |
|--------|-------------|
| errcheck | Check for unchecked errors |
| gosimple | Simplify code |
| govet | Report suspicious constructs |
| ineffassign | Detect ineffective assignments |
| staticcheck | Static analysis checks |
| unused | Find unused code |
| gofmt | Check formatting |
| goimports | Check import ordering |
| misspell | Find spelling mistakes |
| bodyclose | Check HTTP response body is closed |
| copyloopvar | Detect loop variable issues |
| goconst | Find repeated strings that could be constants |
| gosec | Security checks |
| prealloc | Suggest slice pre-allocation |
| unconvert | Remove unnecessary type conversions |

## Configuration

Configuration is in `.golangci.yml` at the repository root.

### Excluded Checks

Some checks are intentionally excluded:

- **G104** (gosec): Logger methods always return nil
- **G204** (gosec): Subprocess with variables is expected for CLI tools
- **G304** (gosec): File paths from config are expected

### Constants

The linter enforces constants for strings with 3+ occurrences and minimum length of 3 characters.

## Fixing Issues

```bash
# Auto-fix formatting
gofmt -w .

# Run with auto-fix (where possible)
golangci-lint run --fix
```

## IDE Integration

Most Go IDEs support golangci-lint natively or via plugins:

- **VS Code**: Use the Go extension with `go.lintTool` set to `golangci-lint`
- **GoLand**: Built-in support via Settings → Tools → golangci-lint
- **Vim/Neovim**: Use [ale](https://github.com/dense-analysis/ale) or [null-ls](https://github.com/jose-elias-alvarez/null-ls.nvim)
