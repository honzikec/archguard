# Contributing to ArchGuard

First off, thanks for taking the time to contribute! Architecture matters, and we appreciate your help in making ArchGuard better.

## Development Setup

1. Make sure you have Go 1.21 or later installed.
2. Clone the repository.
3. Run `go mod tidy` to download dependencies.

## Building and Testing

To build the CLI locally:

```bash
go build -o archguard ./cmd/archguard/main.go
```

To run the test suite:

```bash
go test -v ./...
```

Tests include fixtures found in the `fixtures/` directory to validate policy rule matching on various structures (layered architecture, monorepos, etc.).

## Submitting Pull Requests

1. Fork the repository and create your feature branch from `main`.
2. If you've added new rule types or core logic, please add corresponding tests and optionally update the fixtures.
3. Ensure the entire test suite passes (`go test ./...`).
4. Ensure your code is formatted with standard Go tools (`go fmt ./...`).
5. Create your Pull Request with a clear description of the problem and your solution.
