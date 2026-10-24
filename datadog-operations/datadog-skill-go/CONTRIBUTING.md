# Contributing to Datadog CLI

Thank you for your interest in contributing to the Datadog CLI project.

## Table of Contents

- [Development Setup](#development-setup)
- [Building the Project](#building-the-project)
- [Running Tests](#running-tests)
- [Code Quality](#code-quality)
- [CI/CD Pipeline](#cicd-pipeline)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Docker (optional, for container builds)

### Clone the Repository

```bash
git clone https://github.com/datadog/skill.git
cd skill
```

### Install Dependencies

```bash
go mod download
```

## Building the Project

### Build for Current Platform

```bash
go build -o dd ./cmd
```

### Build with Version Information

```bash
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build \
  -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
  -o dd \
  ./cmd
```

### Cross-Platform Build

Build for Linux:
```bash
GOOS=linux GOARCH=amd64 go build -o dd-linux-amd64 ./cmd
```

Build for macOS:
```bash
GOOS=darwin GOARCH=arm64 go build -o dd-darwin-arm64 ./cmd
```

Build for Windows:
```bash
GOOS=windows GOARCH=amd64 go build -o dd-windows-amd64.exe ./cmd
```

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Tests with Race Detector

```bash
go test -race ./...
```

### Run Specific Package Tests

```bash
go test ./internal/commands
go test ./internal/client
```

## Code Quality

### Format Code

```bash
gofmt -s -w .
```

### Check Formatting

```bash
gofmt -s -l .
```

### Run go vet

```bash
go vet ./...
```

### Run golangci-lint

Install golangci-lint:
```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

Run linter:
```bash
golangci-lint run
```

### Security Scanning

```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run security scan
gosec ./...
```

## CI/CD Pipeline

The project uses GitHub Actions for continuous integration and deployment.

### Workflows

1. **CI Workflow** (`.github/workflows/ci.yml`)
   - Runs on push to main/develop and pull requests
   - Tests on multiple OS (Ubuntu, macOS, Windows)
   - Tests on multiple Go versions (1.21, 1.22, 1.23)
   - Runs linters, formatters, and security scans
   - Uploads coverage reports

2. **Build Workflow** (`.github/workflows/build.yml`)
   - Builds binaries for 6 platforms
   - Generates checksums
   - Uploads artifacts

3. **Coverage Workflow** (`.github/workflows/coverage.yml`)
   - Generates detailed coverage reports
   - Enforces 70% coverage threshold
   - Comments coverage on PRs

4. **Docker Workflow** (`.github/workflows/docker.yml`)
   - Builds multi-arch Docker images
   - Pushes to GitHub Container Registry
   - Runs vulnerability scans

5. **Release Workflow** (`.github/workflows/release.yml`)
   - Triggers on version tags
   - Creates GitHub releases with binaries
   - Generates changelog

### Running CI Checks Locally

Before pushing, ensure all CI checks will pass:

```bash
# Format code
gofmt -s -w .

# Run tests
go test -race ./...

# Run linters
golangci-lint run

# Run security scan
gosec ./...

# Build
go build -o dd ./cmd
```

## Submitting Changes

### Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests and linters
5. Commit with descriptive messages
6. Push to your fork
7. Submit a pull request

### Commit Message Guidelines

Follow conventional commit format:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test additions/changes
- `refactor:` Code refactoring
- `chore:` Build/tooling changes

Examples:
```
feat: Add network monitoring command
fix: Correct timestamp parsing in logs command
docs: Update installation instructions
test: Add unit tests for APM command
```

### Pull Request Process

1. Ensure all CI checks pass
2. Update documentation if needed
3. Add tests for new functionality
4. Ensure coverage meets threshold (70%)
5. Request review from maintainers
6. Address review feedback
7. Squash commits if requested

### Code Review Checklist

- [ ] Code follows Go conventions
- [ ] Tests added for new functionality
- [ ] Documentation updated
- [ ] No security vulnerabilities
- [ ] All CI checks pass
- [ ] Coverage threshold met

## Release Process

### Creating a Release

Releases are automated via GitHub Actions when version tags are pushed.

1. Update version in `cmd/main.go`
2. Commit changes: `git commit -am "chore: Bump version to v1.2.3"`
3. Create and push tag:
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```

4. GitHub Actions will:
   - Build binaries for all platforms
   - Generate checksums
   - Create GitHub release
   - Upload release artifacts
   - Generate changelog

### Version Format

Follow semantic versioning (semver):
- Major: Breaking changes (v2.0.0)
- Minor: New features (v1.1.0)
- Patch: Bug fixes (v1.0.1)

Pre-release versions:
- Alpha: v1.0.0-alpha.1
- Beta: v1.0.0-beta.1
- RC: v1.0.0-rc.1

## Docker Development

### Build Docker Image

```bash
docker build -t datadog-cli:dev .
```

### Run Docker Container

```bash
docker run --rm datadog-cli:dev version
docker run --rm datadog-cli:dev help
```

### Multi-arch Build

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t datadog-cli:latest \
  .
```

## Project Structure

```
.
├── cmd/                    # Main application entry point
│   ├── main.go            # CLI entry point
│   └── demo/              # Demo application
├── internal/              # Internal packages
│   ├── client/            # Datadog API client
│   ├── commands/          # CLI command implementations
│   ├── context/           # Context detection
│   └── observability/     # Observability instrumentation
├── .github/workflows/     # CI/CD workflows
├── Dockerfile             # Container image definition
└── README.md              # Project documentation
```

## Getting Help

- Open an issue for bug reports
- Start a discussion for feature requests
- Check existing issues and discussions

## Code of Conduct

Be respectful and inclusive in all interactions. We follow the Go community standards.

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.
