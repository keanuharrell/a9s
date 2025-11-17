# Contributing to a9s

Thank you for considering contributing to a9s! We welcome contributions from the community.

## Code of Conduct

Please be respectful and constructive in all interactions.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker (for testing Docker builds)
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
   make deps
   ```

3. Install development tools and git hooks:
   ```bash
   make setup-hooks
   ```

4. Build the project:
   ```bash
   make build
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
   make pre-commit
   ```

   This will:
   - Format your code (`gofmt`, `goimports`)
   - Run linters (`golangci-lint`)
   - Run all tests
   - Check for security issues

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests for a specific package
go test ./internal/aws/... -v
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` and `goimports` for formatting (done automatically by `make fmt`)
- Keep functions small and focused
- Write meaningful commit messages
- Add tests for new features
- Update documentation when needed

### Commit Message Guidelines

We follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

**Examples:**
```
feat(ec2): add support for instance tagging
fix(s3): correct bucket deletion logic
docs(readme): update installation instructions
test(iam): add tests for role assessment
```

### Pull Request Process

1. Update the README.md or documentation if needed
2. Add tests for new features
3. Ensure all tests pass: `make test`
4. Ensure linting passes: `make lint`
5. Update CHANGELOG.md with your changes
6. Submit a pull request

### Pull Request Checklist

- [ ] Code follows the project style guidelines
- [ ] Tests added/updated and passing
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventional commits
- [ ] No merge conflicts
- [ ] All CI checks passing

## Project Structure

```
a9s/
├── cmd/              # CLI commands and root command
├── internal/
│   ├── aws/          # AWS service integrations (EC2, IAM, S3)
│   └── tui/          # Terminal UI components
├── scripts/          # Build and utility scripts
├── man/              # Man pages
├── .github/          # GitHub Actions workflows
├── Makefile          # Build automation
└── .goreleaser.yml   # Release configuration
```

## Adding New Features

### Adding a New AWS Service

1. Create a new file in `internal/aws/`:
   ```bash
   touch internal/aws/newservice.go
   touch internal/aws/newservice_test.go
   ```

2. Implement the service following existing patterns
3. Add CLI command in `cmd/`
4. Add TUI view in `internal/tui/`
5. Add tests
6. Update documentation

### Adding New Commands

1. Create command file in `cmd/`:
   ```go
   // cmd/newcmd.go
   package cmd

   import "github.com/spf13/cobra"

   var newCmd = &cobra.Command{
       Use:   "newcmd",
       Short: "Description",
       RunE: func(cmd *cobra.Command, args []string) error {
           // Implementation
           return nil
       },
   }

   func init() {
       rootCmd.AddCommand(newCmd)
   }
   ```

2. Add tests in `cmd/newcmd_test.go`
3. Update man pages: `make man`

## Testing

### Unit Tests

Write unit tests for all new code:

```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input", "output"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := NewFeature(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Integration Tests

For AWS integration tests, use mocks or require AWS credentials:

```go
func TestEC2Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    // Test with real AWS API
}
```

Run without integration tests:
```bash
go test -short ./...
```

## Release Process

Releases are automated via GitHub Actions when a tag is pushed:

1. Update version and changelog
2. Create and push a tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

3. GitHub Actions will:
   - Build binaries for all platforms
   - Generate checksums
   - Create Docker images
   - Publish to GitHub releases
   - Update Homebrew formula

## Getting Help

- Open an issue for bugs or feature requests
- Start a discussion for questions
- Check existing issues and PRs before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
