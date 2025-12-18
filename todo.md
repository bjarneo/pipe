# Pipe - Code Review TODO

## Critical (Fix Immediately)

- [x] **Command Injection Vulnerability** - `internal/ssh/ssh.go:52` **FIXED**
  - Added comprehensive input validation in `config.Validate()`
  - Regex patterns validate hostname, username, image, tag, container name
  - Dangerous shell characters are rejected
  - Path traversal is blocked for env-file, dockerfile, volumes

- [x] **Race Condition in ExecuteCommand** - `internal/ssh/ssh.go:71-93` **FIXED**
  - Added `sync.WaitGroup` to wait for goroutines to complete
  - Added `sync.Mutex` to protect concurrent writes to string builders

- [x] **Nil Pointer Dereference** - `main.go:30-35` **FIXED**
  - Changed to use `fmt.Fprintf(os.Stderr, ...)` and `os.Exit(1)` instead of `log.Fatal()`

- [ ] **No Tests Exist**
  - Zero test files in entire codebase
  - No safety net for changes or refactoring

- [ ] **No Interfaces Defined**
  - Everything is concrete types
  - Impossible to mock, test, or swap implementations

## High Priority

- [x] **Path Traversal** - `internal/deploy/deploy.go:139` **FIXED**
  - Added path traversal detection in `config.Validate()`
  - EnvFile must be relative path, cannot contain `..`

- [ ] **SSH Key Path Expansion Fails Silently** - `internal/config/config.go:113-119`
  - If `os.UserHomeDir()` fails, path remains as `~/...` causing cryptic errors later

- [x] **Build Args Injection** - `internal/docker/docker.go:43-44` **FIXED**
  - Build arg keys validated with regex `^[a-zA-Z_][a-zA-Z0-9_]*$`
  - Build arg values checked for dangerous shell characters

- [x] **Insufficient Input Validation** - `internal/config/config.go:128-133` **FIXED**
  - Added comprehensive validation for all fields
  - Host, user, image, tag, container name, platform, ports, network, volumes, CPUs, memory

- [x] **Volume Mount Injection** - `internal/docker/docker.go:300-302` **FIXED**
  - Volume format validated (must contain `:`)
  - Checked for dangerous characters and path traversal

- [x] **Scanner Errors Ignored** - `internal/ssh/ssh.go:71-93` **FIXED**
  - Added scanner.Err() checks after scanning loops
  - Increased buffer size to 1MB for long lines

- [ ] **Tight Coupling Between Packages**
  - Direct package dependencies, no interfaces
  - Cannot test packages in isolation

- [x] **Missing context.Context** **FIXED**
  - Added `ExecuteCommandContext()` and `CheckContext()` with context support
  - Old functions now wrap context-aware versions for backward compatibility
  - Proper context cancellation handling

## Medium Priority

### Error Handling
- [ ] Uses `%v` instead of `%w` for error wrapping - loses error chain
- [ ] Silent error handling in multiple locations
- [ ] Inconsistent error messages (capitalization, format)
- [ ] Logger methods return errors but callers often ignore them

### Code Complexity
- [x] `Transfer()` function is 62 lines with 7 concerns - `internal/docker/docker.go:54-116` **FIXED**
  - Extracted: `imageRef()`, `getImageSize()`, `getLocalLayers()`, `getRemoteLayers()`, `calculateLayerDiff()`
- [x] `deltaTransfer()` has complex shell script in string - `internal/docker/docker.go:151-205` **FIXED**
  - Extracted: `getCachedLayerPrefixes()`, `buildDeltaTransferScript()`, `parseTransferSize()`
  - Added `compressionRatio` constant
- [x] `Rollback()` function does too much - `internal/deploy/deploy.go:69-135` **FIXED**
  - Extracted: `getCurrentContainerImage()`, `getImageHistory()`, `findPreviousImage()`, `verifyContainerRunning()`, `cleanupBackupContainer()`, `buildRollbackCommands()`
- [ ] `ExecuteCommand()` handles too many concerns - `internal/ssh/ssh.go:44-108`

### Code Duplication
- [ ] SSH command construction repeated 20+ times
- [ ] Remote command pattern `fmt.Sprintf("%s \"cmd\"", ssh.GetCommand(cfg))` everywhere
- [ ] Docker image reference `fmt.Sprintf("%s:%s", cfg.Image, cfg.Tag)` duplicated

### Security
- [x] Log files created with world-readable permissions (0644) - `internal/logger/logger.go:16` **FIXED**
  - Changed to 0600 permissions
  - Added nil checks and sync before close in Fatal()
- [ ] Sensitive data (SSH key paths, commands) logged without redaction
- [ ] SSH key content written to disk in GitHub Action - `action.yml:70`
- [ ] SSH known hosts auto-trusted without verification - `action.yml:72`

### Architecture
- [ ] SSH package contains generic command execution (wrong package)
- [ ] Docker package is 383 lines handling 5+ concerns
- [ ] Config package has CLI parsing + help text mixed in
- [ ] Deploy package has no clear abstraction or interface
- [ ] Global logger dependency passed to every function

### Resource Management
- [ ] Logger `Close()` errors ignored
- [ ] Temp directories may persist on SSH connection failure
- [ ] No cleanup of backup containers on failed rollback

## Low Priority

- [ ] Magic numbers (compression ratio 40%, max releases 5)
- [ ] Hardcoded values without constants
- [ ] Inconsistent string operations
- [ ] Default scanner buffer may be too small for long Docker output
- [ ] Text truncation without ellipsis in stats display
- [ ] Empty volume/tag strings not filtered
- [ ] Malformed build arguments silently ignored

## Architectural Recommendations

### Recommended Package Structure
```
├── cmd/pipe/main.go      # CLI parsing, dependency injection setup
├── internal/
│   ├── config/           # Pure config struct + validation only
│   ├── executor/         # CommandExecutor interface + implementation
│   ├── docker/
│   │   ├── client/       # DockerClient interface
│   │   ├── transfer/     # TransferStrategy interface
│   │   └── parser/       # Parsing utilities
│   ├── remote/           # RemoteExecutor interface (SSH implementation)
│   ├── deploy/           # DeploymentStrategy interface
│   └── logger/           # Logger interface, accepts io.Writer
```

### Key Interfaces to Define
```go
type Logger interface {
    Info(message string) error
    Error(message string, err error) error
    Close() error
}

type CommandExecutor interface {
    Execute(ctx context.Context, command string) (*CommandResult, error)
}

type DockerClient interface {
    Check(ctx context.Context) error
    Build(ctx context.Context, opts BuildOptions) error
    Transfer(ctx context.Context, opts TransferOptions) error
    Deploy(ctx context.Context, opts DeployOptions) error
}

type RemoteExecutor interface {
    Execute(ctx context.Context, command string) (*Result, error)
    CopyFile(ctx context.Context, local, remote string) error
}
```

## Progress Tracking

### Completed
- [x] Fixed missing `--dockerfile` flag in `action.yml`
- [x] Fixed nil pointer dereference in `main.go:initLogger()`
- [x] Fixed race condition in `ssh.go:ExecuteCommand()` - added sync.WaitGroup and sync.Mutex
- [x] Added comprehensive input validation in `config.Validate()` to prevent command injection
- [x] Fixed log file permissions from 0644 to 0600
- [x] Added nil checks to logger methods
- [x] Fixed path traversal vulnerability for env-file, dockerfile, volumes
- [x] Added build args key/value validation
- [x] Added scanner.Err() checks in ExecuteCommand with 1MB buffer
- [x] Fixed SSH key path expansion silent failure - now shows warning
- [x] Added comprehensive tests for config validation (14 test suites, 100+ test cases)
- [x] Added context.Context support with ExecuteCommandContext() and CheckContext()
- [x] Added Logger interface (`logger.Interface`) for dependency injection
- [x] Added `NewWithWriter()` for custom output destinations
- [x] Added CommandExecutor interface (`ssh.Executor`) for testability
- [x] Added `ssh.DefaultExecutor` implementation
- [x] Refactored `Transfer()` - extracted `imageRef()`, `getImageSize()`, `getLocalLayers()`, `getRemoteLayers()`, `calculateLayerDiff()`
- [x] Refactored `deltaTransfer()` - extracted `getCachedLayerPrefixes()`, `buildDeltaTransferScript()`, `parseTransferSize()`, added `compressionRatio` constant
- [x] Refactored `Rollback()` - extracted `getCurrentContainerImage()`, `getImageHistory()`, `findPreviousImage()`, `verifyContainerRunning()`, `cleanupBackupContainer()`, `buildRollbackCommands()`

### In Progress
- [ ] None

### Next Up
- [ ] Add more test coverage for other packages
- [ ] Consider using Docker SDK instead of shell commands
- [ ] Add retry logic for network operations

### Blocked
- None
