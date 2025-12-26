# Testing

This document covers how to run and write tests for the Pipe project.

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Verbose Output

```bash
go test -v ./...
```

### Run Tests for a Specific Package

```bash
go test ./internal/docker
go test ./internal/config
go test ./internal/ssh
go test ./internal/logger
go test ./internal/stats
go test ./internal/container
go test ./internal/deploy
```

### Run a Specific Test

```bash
go test -v -run TestRunBuilder ./internal/docker
go test -v -run TestValidate ./internal/config
```

### Run Tests Matching a Pattern

```bash
go test -v -run "TestValidate.*" ./internal/config
go test -v -run "TestRunBuilder.*" ./internal/docker
```

## Code Coverage

### View Coverage Summary

```bash
go test -cover ./...
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

Open `coverage.html` in a browser to view the detailed coverage report.

### Coverage by Package

```bash
go test -cover ./internal/config
go test -cover ./internal/docker
```

## Test Structure

Tests are located alongside the source files in each package:

```
internal/
├── config/
│   ├── config.go
│   └── config_test.go
├── container/
│   ├── container.go
│   └── container_test.go
├── deploy/
│   ├── deploy.go
│   └── deploy_test.go
├── docker/
│   ├── docker.go
│   └── docker_test.go
├── logger/
│   ├── logger.go
│   └── logger_test.go
├── ssh/
│   ├── ssh.go
│   └── ssh_test.go
└── stats/
    ├── stats.go
    └── stats_test.go
```

## Test Categories

### Unit Tests

All tests are unit tests that don't require external dependencies (no real SSH connections or Docker daemon needed).

- **config**: Validation rules, injection prevention, configuration merging
- **ssh**: Command building, flag generation
- **docker**: Layer parsing, size parsing, RunBuilder, command generation
- **logger**: Logging methods, quiet mode, nil safety
- **stats**: Byte/duration formatting, JSON output
- **container**: ID truncation, time formatting, status colorization
- **deploy**: Rollback command generation, image history parsing

### Security Tests

The `config` package includes extensive security tests for command injection prevention:

```bash
go test -v -run "TestValidate_CommandInjectionPrevention" ./internal/config
```

## Writing Tests

### Test Naming Convention

```go
func TestFunctionName(t *testing.T) { ... }
func TestFunctionName_SpecificCase(t *testing.T) { ... }
```

### Table-Driven Tests

Most tests use table-driven patterns:

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"valid input", "foo", "bar"},
        {"empty input", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Example(tt.input)
            if result != tt.expected {
                t.Errorf("Example(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}
```

### Testing with Mock Writers

The logger package can be tested without file I/O:

```go
var buf bytes.Buffer
log := logger.NewWithWriter(&buf, false)
log.Info("test message")
// Check buf.String() for output
```

## Continuous Integration

Run tests before committing:

```bash
go build -o pipe . && go test ./...
```

This ensures the project builds and all tests pass.
