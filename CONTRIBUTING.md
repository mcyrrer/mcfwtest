# Contributing to FWProbe

Thank you for your interest in contributing to FWProbe! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git

### Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/mcfwtest.git
   cd mcfwtest
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/mcyrrer/mcfwtest.git
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```

### Building

```bash
go build -o fwprobe ./cmd/fwprobe
```

### Running Tests

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test ./... -cover
```

Run tests with verbose output:
```bash
go test ./... -v
```

## Code Style

### Go Code

- Follow standard Go conventions and idioms
- Use `gofmt` to format your code
- Run `go vet` to check for common mistakes
- Add comments to all exported types, functions, and methods
- Keep functions focused and small
- Prefer clear code over clever code

### Testing

- Write unit tests for all new functionality
- Aim for at least 80% code coverage
- Use table-driven tests where appropriate
- Test both success and error cases
- Use descriptive test names

Example:
```go
func TestExpandCIDR(t *testing.T) {
    tests := []struct {
        name        string
        cidr        string
        expected    []string
        expectError bool
    }{
        {
            name:     "single host /32",
            cidr:     "192.168.1.1/32",
            expected: []string{"192.168.1.1"},
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ExpandCIDR(tt.cidr)
            // Assertions...
        })
    }
}
```

### Commit Messages

- Use clear and descriptive commit messages
- Start with a verb in the present tense (e.g., "Add", "Fix", "Update")
- Keep the first line under 72 characters
- Add a blank line followed by a more detailed description if needed

Example:
```
Add support for UDP probing

Implement UDP probe functionality with ICMP unreachable detection.
Includes unit tests and documentation updates.
```

## Pull Request Process

1. Create a new branch for your feature or bugfix:
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. Make your changes and commit them with clear messages

3. Ensure all tests pass:
   ```bash
   go test ./...
   ```

4. Push your branch to your fork:
   ```bash
   git push origin feature/my-new-feature
   ```

5. Open a Pull Request on GitHub with:
   - A clear title describing the change
   - A description of what changed and why
   - Any relevant issue numbers (e.g., "Fixes #123")

6. Wait for review and address any feedback

## Areas for Contribution

### High Priority

- UDP probe implementation (Phase 2)
- JSON and JUnit output formatters (Phase 2)
- GitHub Actions CI/CD pipeline (Phase 3)
- Documentation improvements

### Medium Priority

- Additional test coverage
- Performance optimizations
- Error message improvements
- Example configurations

### Low Priority

- Windows support (future)
- Application-layer probes (future)
- Prometheus metrics (future)

## Reporting Bugs

When reporting bugs, please include:

- FWProbe version (`fwprobe version`)
- Operating system and architecture
- Configuration file (if applicable)
- Steps to reproduce
- Expected behavior
- Actual behavior
- Any error messages or logs

## Suggesting Features

We welcome feature suggestions! Please:

- Check existing issues to avoid duplicates
- Provide a clear use case
- Explain how it aligns with FWProbe's goals
- Consider backward compatibility

## Code Review Guidelines

When reviewing code:

- Be respectful and constructive
- Focus on the code, not the person
- Explain your reasoning
- Suggest alternatives when possible
- Approve when the code meets standards

## Questions?

If you have questions about contributing, feel free to:

- Open an issue on GitHub
- Reach out to the maintainers

## License

By contributing to FWProbe, you agree that your contributions will be licensed under the Apache License 2.0.
