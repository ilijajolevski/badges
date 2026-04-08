# Contributing to CertifyHub Service

Thank you for your interest in contributing. This document outlines the process
for reporting issues and submitting changes.

## Reporting Issues

Please use the project issue tracker to report bugs or request features. When
filing a bug report, include:

- A clear description of the problem
- Steps to reproduce it
- Expected vs. actual behaviour
- Environment details (OS, Go version, Docker version if applicable)

For security vulnerabilities, do **not** open a public issue. Contact the
maintainers directly (see [Support](#support)).

## Branch and Pull Request Workflow

1. Fork the repository and create a feature branch from `main`:
   ```
   git checkout -b feature/your-feature-name
   ```
2. Keep commits focused and well-described.
3. Submit a merge request (MR) against `main`.
4. Ensure all checks pass before requesting review.

Branch naming conventions:
- `feature/<description>` — new functionality
- `fix/<description>` — bug fixes
- `docs/<description>` — documentation-only changes
- `refactor/<description>` — code restructuring without behaviour change

## Code Style

- Follow standard Go formatting: run `gofmt` or `goimports` before committing.
- Keep functions small and focused.
- Avoid adding dependencies without discussion.

## Testing

All changes must pass existing tests. New functionality should include tests.

```bash
make test          # Run all tests
```

Run a single test:
```bash
go test -v -run TestFunctionName ./internal/badge/
```

## Building

```bash
make deps          # Install Go dependencies
make db-init       # Create the db/ directory (first run only)
make run           # Run locally on port 9000
make build         # Build binary to bin/badge-service
```

## Support

For contribution-related questions, open an issue or contact the project
maintainers listed in the AUTHORS file.
