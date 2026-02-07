# CLAUDE.md - AI Assistant Guidelines for FWProbe

## Project Overview

FWProbe is a cross-platform CLI firewall rule testing tool written in Go. It validates whether specified TCP and UDP ports are open or closed on target endpoints using YAML-based configuration.

- **Repository:** github.com/mcyrrer/mcfwtest
- **Language:** Go 1.25.6
- **CLI Framework:** Cobra
- **License:** MIT

## Project Structure

```
cmd/fwprobe/main.go       # CLI entry point, defines all commands and flags
internal/config/           # YAML config parsing, defaults, strict validation
internal/network/          # CIDR expansion and DNS resolution
internal/probe/            # TCP and UDP probe implementations
internal/output/           # Output formatters (pretty, JSON, JUnit XML)
internal/runner/           # Test orchestration and concurrent execution
testdata/                  # YAML fixtures for config validation tests
```

## Build & Development Commands

```bash
# Build the binary
go build -o fwprobe ./cmd/fwprobe

# Run all tests
go test ./...

# Run tests with race detection and coverage (as CI does)
go test -v -race -coverprofile=coverage.out ./...

# Run go vet
go vet ./...

# Run linter (golangci-lint must be installed)
golangci-lint run

# Tidy dependencies
go mod tidy
```

## Testing

- Test framework: `testify` (assertions and require)
- Test files: `internal/config/config_test.go`, `internal/probe/tcp_test.go`, `internal/network/cidr_test.go`
- Test data: YAML fixtures in `testdata/`
- Coverage target: 80%+
- CI runs with `-race` flag on ubuntu-latest and macos-latest
- Use table-driven tests with descriptive names

## Code Style & Linting

Configured in `.golangci.yml` with these linters enabled:
- errcheck, gosimple, govet, ineffassign, staticcheck, unused
- gofmt, goimports, misspell, revive, bodyclose, gocritic, gosec

Key rules:
- Format with `gofmt`
- All exported types and functions must have comments
- errcheck and gosec are excluded in test files
- Strict mode on YAML decoding rejects unknown fields

## Architecture Notes

- Config uses strict YAML parsing (`KnownFields(true)`) to catch typos
- Duration fields use a custom `Duration` type wrapping `time.Duration` with YAML unmarshaling
- Endpoints support both singular (`host`/`port`) and plural (`hosts`/`ports`) fields but not both simultaneously
- CIDR ranges are capped at /16 (65,534 hosts)
- Runner uses a worker pool with configurable concurrency (default 10)
- UDP probing requires root/admin privileges for ICMP detection

## CLI Commands

| Command | Purpose |
|---------|---------|
| `fwprobe run` | Execute firewall tests from config file |
| `fwprobe validate` | Validate a config file and print summary |
| `fwprobe createconfig` | Generate a template config to stdout or file |
| `fwprobe version` | Print version, commit, and build date |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All tests passed |
| 1 | One or more tests failed |
| 2 | Configuration error |
| 3 | Runtime error |

## Configuration Schema (version "1")

```yaml
version: "1"
defaults:
  timeout: 2s            # 100ms-30s
  protocol: tcp          # tcp or udp
endpoints:
  - name: unique-name    # Required, must be unique
    description: "..."   # Optional
    host: 1.2.3.4        # Single host/IP/CIDR (mutually exclusive with hosts)
    hosts: [...]         # Multiple hosts (mutually exclusive with host)
    port: 80             # Single port 1-65535 (mutually exclusive with ports)
    ports: [80, 443]     # Multiple ports (mutually exclusive with port)
    protocol: tcp        # Overrides default
    timeout: 3s          # Overrides default
    expect: open         # open or closed
```

## CI/CD

- GitHub Actions: `.github/workflows/ci.yaml` (test, vet, build matrix)
- Release: `.github/workflows/release.yaml` with GoReleaser on version tags
- Cross-platform builds: Linux/macOS, amd64/arm64

## Commit Message Convention

- Start with a verb in present tense: Add, Fix, Update, Remove
- First line under 72 characters
- Blank line then detailed description if needed
