# Contributing to a9s

Thank you for considering contributing to a9s! We welcome contributions from the community.

## Code of Conduct

Please be respectful and constructive in all interactions.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Make
- Git

### Setup Development Environment

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/yourusername/a9s.git
   cd a9s
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the project:
   ```bash
   make build
   ```

4. Run the application:
   ```bash
   ./a9s
   ```

## Development Workflow

### Before Making Changes

1. Create a new branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes

3. Run the automated checks:
   ```bash
   make lint
   make test
   ```

### Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/services/ec2/... -v
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions small and focused
- Write meaningful commit messages
- Add tests for new features

## Commit Message Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/) with automatic semantic versioning:

| Type | Description | Version Bump |
|------|-------------|--------------|
| `feat:` | New feature | Minor (0.X.0) |
| `fix:` | Bug fix | Patch (0.0.X) |
| `docs:` | Documentation | None |
| `refactor:` | Code refactoring | Patch |
| `test:` | Adding tests | None |
| `chore:` | Maintenance | None |
| `BREAKING CHANGE:` | Breaking change | Major (X.0.0) |

**Examples:**
```
feat(lambda): add function invocation support
fix(ec2): correct instance state display
docs: update keyboard shortcuts
refactor(tui): simplify view rendering
```

## Pull Request Process

1. Ensure all tests pass: `make test`
2. Ensure linting passes: `make lint`
3. Update documentation if needed
4. Submit a pull request

### Pull Request Checklist

- [ ] Code follows the project style guidelines
- [ ] Tests added/updated and passing
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventional commits
- [ ] All CI checks passing

## Project Structure

```
a9s/
├── cmd/                    # CLI entry point
│   └── root.go             # Main command setup
├── internal/
│   ├── core/               # Core interfaces and types
│   ├── config/             # Configuration loading
│   ├── container/          # Dependency injection
│   ├── registry/           # Service registry
│   ├── hooks/              # Event hooks system
│   ├── services/           # AWS service implementations
│   │   ├── base/           # Base view components
│   │   ├── ec2/            # EC2 service & view
│   │   ├── iam/            # IAM service & view
│   │   ├── s3/             # S3 service & view
│   │   └── lambda/         # Lambda service & view
│   ├── tui/                # Terminal UI
│   │   ├── app.go          # Main TUI application
│   │   ├── theme/          # UI theming
│   │   └── components/     # Reusable UI components
│   └── aws/                # AWS client factory
├── .github/workflows/      # CI/CD pipelines
├── .goreleaser.yml         # Release configuration
└── .releaserc.json         # Semantic release config
```

## Adding New Features

### Adding a New AWS Service

1. Create service implementation in `internal/services/newservice/`:
   ```
   internal/services/newservice/
   ├── service.go    # AWS API interactions
   └── view.go       # TUI view
   ```

2. Implement the `core.AWSService` interface
3. Implement the `core.View` interface
4. Register the service in `cmd/root.go`
5. Add tests
6. Update documentation

### Architecture Principles

- **Interface-driven**: All services implement `core.AWSService`
- **Dependency injection**: Services receive dependencies via constructor
- **Event-based**: Use hooks for cross-cutting concerns
- **Separation of concerns**: Service logic separate from TUI rendering

## Release Process

Releases are **fully automatic** via semantic-release:

1. Push commits to `main` branch
2. semantic-release analyzes commit messages
3. If releasable changes exist:
   - Version is bumped automatically
   - CHANGELOG.md is updated
   - GitHub release is created
   - Binaries are built and uploaded
   - Homebrew formula is updated

**No manual tagging required!**

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
