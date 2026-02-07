# FWProbe — Firewall Rule Testing Tool

FWProbe is a cross-platform, CLI-based network testing tool written in Go. It validates firewall rules by testing whether specified TCP and UDP ports are open or closed on target endpoints.

## Features

- **TCP Connection Testing**: Validate TCP port accessibility
- **CIDR Range Support**: Test entire network ranges
- **Concurrent Execution**: Run multiple tests in parallel for fast results
- **Flexible Configuration**: YAML-based configuration with sensible defaults
- **Pretty Terminal Output**: Color-coded results with Unicode formatting
- **CI/CD Friendly**: Exit codes and multiple output formats planned

## Installation

### From Source

```bash
go install github.com/mcyrrer/mcfwtest/cmd/fwprobe@latest
```

### Build Locally

```bash
git clone https://github.com/mcyrrer/mcfwtest.git
cd mcfwtest
go build -o fwprobe ./cmd/fwprobe
```

## Quick Start

1. Create a configuration file `fwprobe.yaml`:

```yaml
version: "1"

defaults:
  timeout: 2s
  protocol: tcp

endpoints:
  - name: google-dns
    description: "Google Public DNS"
    host: 8.8.8.8
    port: 53
    expect: open

  - name: localhost-http
    description: "Local HTTP server"
    host: 127.0.0.1
    port: 80
    expect: closed
```

2. Run the tests:

```bash
fwprobe run
```

3. View the results:

```
╭────────────────────────────────────╮
│ FWProbe — Firewall Rule Testing    │
│ Config: fwprobe.yaml                │
│ Tests: 2 | Concurrency: 10          │
╰────────────────────────────────────╯

  google-dns
    ✔ 8.8.8.8:53/tcp ··· open (31ms) PASS

  localhost-http
    ✔ 127.0.0.1:80/tcp ··· closed (0ms) PASS

────────────────────────────────────────────────────────────
  Results: 2 passed · 0 failed · 2 total
  Duration: 31ms
────────────────────────────────────────────────────────────
```

## Configuration Reference

### Configuration File Structure

```yaml
version: "1"

defaults:
  timeout: 2s          # Global default timeout
  protocol: tcp        # Global default protocol (tcp|udp)

endpoints:
  - name: endpoint-name       # Required: Unique identifier
    description: "..."         # Optional: Human-readable description
    host: 192.168.1.1          # Single IP, hostname, or CIDR range
    # OR
    hosts:                     # Multiple hosts
      - 192.168.1.1
      - 192.168.1.2
    port: 80                   # Single port
    # OR
    ports:                     # Multiple ports
      - 80
      - 443
    protocol: tcp              # tcp or udp (overrides default)
    timeout: 3s                # Timeout override (100ms-30s)
    expect: open               # Expected result: open|closed
```

### Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `version` | string | Yes | — | Config schema version. Must be `"1"`. |
| `defaults.timeout` | duration | No | `2s` | Global timeout for connection attempts. |
| `defaults.protocol` | string | No | `tcp` | Global default protocol. |
| `endpoints[].name` | string | Yes | — | Unique identifier for the endpoint test. |
| `endpoints[].description` | string | No | — | Human-readable description. |
| `endpoints[].host` | string | Conditional | — | Single IP, hostname, or CIDR range. |
| `endpoints[].hosts` | []string | Conditional | — | List of IPs/hostnames. |
| `endpoints[].port` | int | Conditional | — | Single port number (1–65535). |
| `endpoints[].ports` | []int | Conditional | — | List of port numbers. |
| `endpoints[].protocol` | string | No | `tcp` | `tcp` or `udp`. |
| `endpoints[].timeout` | duration | No | `2s` | Per-endpoint timeout override. |
| `endpoints[].expect` | string | No | `open` | Expected result: `open` or `closed`. |

### CIDR Range Examples

```yaml
endpoints:
  - name: small-network
    host: 192.168.1.0/30  # Tests 192.168.1.1 and 192.168.1.2
    port: 22
    expect: closed

  - name: larger-network
    host: 10.0.0.0/24     # Tests 10.0.0.1 through 10.0.0.254
    port: 443
    expect: open
```

**Note**: CIDR ranges are capped at `/16` (65,534 hosts) to prevent resource exhaustion.

## CLI Reference

### Commands

```
fwprobe run         Run firewall tests
fwprobe validate    Validate configuration file
fwprobe version     Print version information
fwprobe help        Help about any command
```

### `fwprobe run` Options

```
-c, --config string       Path to config file (default "fwprobe.yaml")
-o, --output string       Output format: pretty, json, junit (default "pretty")
    --fail-fast           Stop on first failure
-j, --concurrency int     Max concurrent tests (default 10)
-f, --filter string       Run only endpoints matching name pattern (glob)
    --no-color            Disable colored output
    --exit-code           Exit with code 1 if any test fails (default true)
-q, --quiet               Only print failures and summary
```

### `fwprobe validate` Options

```
-c, --config string    Path to config file (default "fwprobe.yaml")
```

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All tests passed |
| `1` | One or more tests failed |
| `2` | Configuration error (invalid YAML, validation failure) |
| `3` | Runtime error (permissions, network error) |

## Examples

### Validate Configuration

```bash
fwprobe validate -c my-config.yaml
```

### Run Tests with Custom Config

```bash
fwprobe run -c production.yaml
```

### Run Specific Endpoints

```bash
fwprobe run --filter "mysql-*"
```

### Stop on First Failure

```bash
fwprobe run --fail-fast
```

### Quiet Mode (Only Show Failures)

```bash
fwprobe run --quiet
```

## Development

### Prerequisites

- Go 1.21 or later

### Build

```bash
go build -o fwprobe ./cmd/fwprobe
```

### Run Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test ./... -cover
```

## Current Status

**Phase 1 Complete (v0.1)**:
- ✅ Core TCP probing
- ✅ YAML configuration parsing and validation
- ✅ CIDR range expansion
- ✅ DNS resolution
- ✅ Concurrent test execution
- ✅ Pretty terminal output
- ✅ Comprehensive unit tests

**Planned for Phase 2 (v0.2)**:
- UDP probing with ICMP unreachable detection
- JSON output format
- JUnit XML output format

**Planned for Phase 3 (v0.3)**:
- GitHub Actions CI/CD pipeline
- GoReleaser for cross-compilation
- Pre-built binaries for Linux and macOS

**Planned for Phase 4 (v1.0)**:
- Complete documentation
- Integration tests
- Final polish and release

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for details.

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing
- [testify](https://github.com/stretchr/testify) - Test assertions
