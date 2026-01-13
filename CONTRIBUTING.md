# Contributing to Space

Thank you for your interest in contributing to Space!

## Prerequisites

This project uses [Nix](https://nixos.org/) to provide a consistent development environment with all required dependencies (Go 1.25+, golangci-lint, etc.).

1. [Install Nix](https://github.com/DeterminateSystems/nix-installer) - supports Linux, macOS, and WSL2
2. Run `nix develop` (or `nix-shell`) to enter the development shell

## Building

```bash
go build -o space .
```

## Testing

```bash
# Run all tests
go test -race ./...
```

## Code Style

- Format code: `golangci-lint fmt`
- Lint code: `golangci-lint run`

Please ensure your code passes linting before submitting a PR.

## Submitting Changes

Before submitting a pull request, please [open an issue](https://github.com/namespacelabs/space/issues/new) describing the problem you're solving or the feature you'd like to add. This helps us discuss the approach before you invest time in implementation.

1. Open an issue explaining the problem or feature
2. Fork the repository
3. Create a feature branch
4. Make your changes
5. Run tests and linting
6. Submit a pull request linking to the issue
