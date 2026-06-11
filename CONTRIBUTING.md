# Contributing to MOFU

Thank you for your interest in contributing to MOFU!

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch
4. Make your changes
5. Run tests
6. Submit a pull request

## Development

### Prerequisites

- Go 1.21 or later
- Git

### Building

```bash
go build ./...
```

### Testing

```bash
go test ./...
```

### Benchmarks

```bash
go test -bench=. -benchmem ./...
```

## Code Style

- Follow standard Go conventions
- Add GoDoc comments to all exported types
- Keep functions focused and small
- Write tests for new functionality

## Pull Requests

- Keep PRs focused on one change
- Include tests for new features
- Update documentation if needed
- Run `go vet` and `go test` before submitting

## Issues

- Use GitHub Issues for bugs and feature requests
- Include reproduction steps for bugs
- Be respectful and constructive

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
