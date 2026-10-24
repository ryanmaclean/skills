# CI/CD Guide for Datadog CLI

Complete guide to the continuous integration and deployment infrastructure for the Datadog CLI project.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Workflow Details](#workflow-details)
- [Security Considerations](#security-considerations)
- [Local Development](#local-development)
- [Release Process](#release-process)
- [Troubleshooting](#troubleshooting)

## Overview

The Datadog CLI uses GitHub Actions for all CI/CD operations. The pipeline includes:

- Automated testing on multiple platforms and Go versions
- Code quality checks (linting, formatting, security)
- Cross-platform binary builds
- Docker image creation and publishing
- Automated releases with changelog generation
- Code coverage reporting and enforcement

## Quick Start

### For Contributors

Before submitting a PR, run:

```bash
make ci
```

This runs all CI checks locally:
- Code formatting check
- `go vet`
- golangci-lint
- Tests with race detector
- Coverage report

### For Maintainers

Creating a release:

```bash
# Update version in cmd/main.go if needed
git tag v1.2.3
git push origin v1.2.3
```

The release workflow automatically:
- Builds binaries for all platforms
- Creates GitHub release
- Uploads artifacts with checksums
- Generates changelog

## Workflow Details

### 1. CI Workflow

**File:** `.github/workflows/ci.yml`

**Triggers:**
- Push to `main` or `develop`
- Pull requests to `main` or `develop`

**Jobs:**

#### Test Matrix
Runs on every push and PR:
- **Operating Systems:** Ubuntu, macOS, Windows
- **Go Versions:** 1.21, 1.22, 1.23
- **Total Combinations:** 9 (3 OS × 3 Go versions)

**Steps:**
1. Checkout code
2. Set up Go with caching
3. Verify dependencies (`go mod verify`)
4. Run `go vet`
5. Check code formatting (`gofmt`)
6. Run tests with race detector
7. Upload coverage (Ubuntu + Go 1.23 only)

#### Lint
Runs golangci-lint with comprehensive checks:
- Error checking
- Code simplification
- Security issues
- Best practices
- Code style

#### Build
Builds binary and verifies it works:
```bash
go build -v -o dd ./cmd
./dd version
```

#### Security
Scans code for security vulnerabilities:
- Runs Gosec security scanner
- Uploads results to GitHub Security tab
- Generates SARIF report

**Required for PR Merge:** Yes

### 2. Build Workflow

**File:** `.github/workflows/build.yml`

**Triggers:**
- Push to `main` or `develop`
- Pull requests
- Manual dispatch

**Platforms:**
- linux/amd64
- linux/arm64
- darwin/amd64 (Intel Mac)
- darwin/arm64 (Apple Silicon)
- windows/amd64
- windows/arm64

**Build Process:**

1. Extract version info from git
2. Build static binary with CGO disabled
3. Inject version, commit, build date via ldflags
4. Create platform-specific archive:
   - Unix: `tar.gz`
   - Windows: `zip`
5. Generate SHA256 checksum
6. Upload as artifact (7-day retention)

**Artifacts:**
- `dd-{os}-{arch}.tar.gz` (or `.zip`)
- `dd-{os}-{arch}.tar.gz.sha256`

### 3. Release Workflow

**File:** `.github/workflows/release.yml`

**Triggers:**
- Version tags: `v*.*.*` (e.g., `v1.2.3`)
- Pre-release tags: `v*.*.*-*` (e.g., `v1.0.0-beta.1`)

**Process:**

1. **Build Job:**
   - Builds for all 6 platforms
   - Version from git tag
   - Creates release archives with version in name
   - Generates checksums

2. **Release Job:**
   - Downloads all artifacts
   - Generates changelog from commits
   - Creates release notes with installation instructions
   - Creates GitHub Release
   - Uploads all binaries and checksums
   - Marks pre-releases appropriately

**Release Notes Include:**
- Version number
- Platform-specific installation commands
- Changelog from commits
- SHA256 checksums

**Example:**
```bash
# Create release
git tag v1.2.3
git push origin v1.2.3

# Create pre-release
git tag v1.3.0-beta.1
git push origin v1.3.0-beta.1
```

### 4. Coverage Workflow

**File:** `.github/workflows/coverage.yml`

**Triggers:**
- Push to `main` or `develop`
- Pull requests

**Process:**

1. Run tests with coverage
2. Generate HTML report
3. Calculate coverage percentage
4. Enforce 70% minimum threshold
5. Upload to Codecov
6. Comment on PR with results
7. Create GitHub Actions summary

**Coverage Threshold:** 70% minimum

**Failure:** PR check fails if coverage below 70%

**Artifacts:**
- HTML coverage report (30-day retention)

### 5. Docker Workflow

**File:** `.github/workflows/docker.yml`

**Triggers:**
- Push to `main` or `develop`
- Version tags
- Pull requests to `main`
- Manual dispatch

**Process:**

1. Set up QEMU for multi-arch builds
2. Set up Docker Buildx
3. Log in to GitHub Container Registry
4. Extract metadata for tags
5. Build multi-arch image (linux/amd64, linux/arm64)
6. Push to ghcr.io (except for PRs)
7. Run Trivy vulnerability scanner
8. Upload security results

**Image Tags:**
- `latest` - Latest on default branch
- `main` - Latest on main branch
- `develop` - Latest on develop branch
- `v1.2.3` - Semantic version
- `v1.2` - Major.minor
- `v1` - Major version
- `{branch}-{sha}` - Branch with commit

**Registry:** `ghcr.io/{owner}/{repo}`

**Usage:**
```bash
docker pull ghcr.io/{owner}/{repo}:latest
docker run --rm ghcr.io/{owner}/{repo}:latest version
```

### 6. Validate Workflow

**File:** `.github/workflows/validate.yml`

**Triggers:**
- Push to `main` or `develop`
- Pull requests

**Checks:**

1. **Workflow Files:** YAML syntax validation
2. **Go Files:** Verify go.mod/go.sum consistency
3. **Dockerfile:** Lint with Hadolint
4. **Markdown:** Lint all .md files
5. **YAML Files:** Validate all YAML syntax

## Security Considerations

### Command Injection Prevention

All workflows follow GitHub Actions security practices:

**Safe Inputs (can use directly):**
```yaml
run: echo "${{ github.ref_name }}"
run: echo "${{ github.sha }}"
run: echo "${{ github.repository }}"
```

**Unsafe Inputs (must use env vars):**
```yaml
# WRONG - vulnerable to command injection
run: echo "${{ github.event.issue.title }}"

# RIGHT - safe pattern
env:
  TITLE: ${{ github.event.issue.title }}
run: echo "${TITLE}"
```

**Risky Inputs:**
- `github.event.issue.title/body`
- `github.event.pull_request.title/body`
- `github.event.comment.body`
- `github.event.head_commit.message`
- User-provided content

### Permissions

Each workflow declares minimal required permissions:

```yaml
permissions:
  contents: read        # Read repository
  contents: write       # Create releases
  packages: write       # Push Docker images
  pull-requests: write  # Comment on PRs
```

### Dependency Security

- **Dependabot:** Automated dependency updates
- **Gosec:** Security scanning for Go code
- **Trivy:** Docker image vulnerability scanning
- **SARIF:** Upload results to GitHub Security

## Local Development

### Prerequisites

- Go 1.21+
- Docker (for container builds)
- golangci-lint
- make

### Running CI Checks Locally

```bash
# All checks (recommended before pushing)
make ci

# Individual checks
make fmt-check          # Check formatting
make vet                # Run go vet
make lint               # Run golangci-lint
make test               # Run tests
make test-race          # Test with race detector
make test-coverage      # Generate coverage
make build              # Build binary
```

### Build Commands

```bash
# Current platform
make build

# All platforms
make build-all

# With version info
VERSION=v1.2.3 make build

# Docker image
make docker
make docker-run
```

### Running Tests

```bash
# All tests
go test ./...

# With coverage
make test-coverage
open coverage.html

# With race detector
make test-race

# Specific package
go test ./internal/commands
```

## Release Process

### Semantic Versioning

Follow [semver](https://semver.org/):

- **Major (v2.0.0):** Breaking changes
- **Minor (v1.1.0):** New features, backward compatible
- **Patch (v1.0.1):** Bug fixes, backward compatible

**Pre-releases:**
- Alpha: `v1.0.0-alpha.1`
- Beta: `v1.0.0-beta.1`
- RC: `v1.0.0-rc.1`

### Creating a Release

1. **Prepare:**
   ```bash
   # Run full CI locally
   make ci

   # Update version if needed (optional)
   # Edit cmd/main.go to update default version
   ```

2. **Create Tag:**
   ```bash
   # Standard release
   git tag v1.2.3
   git push origin v1.2.3

   # Pre-release
   git tag v1.3.0-beta.1
   git push origin v1.3.0-beta.1
   ```

3. **Automation:**
   - GitHub Actions builds all binaries
   - Creates GitHub Release
   - Generates changelog
   - Uploads artifacts

4. **Verify:**
   - Check GitHub Releases page
   - Download and test binaries
   - Verify checksums

### Release Checklist

- [ ] All CI checks passing
- [ ] Version number updated (if needed)
- [ ] CHANGELOG reviewed
- [ ] Breaking changes documented
- [ ] Migration guide created (if needed)
- [ ] Tag created and pushed
- [ ] Release notes reviewed
- [ ] Binaries tested
- [ ] Docker image verified
- [ ] Documentation updated

## Troubleshooting

### Test Failures

**Symptoms:** Tests fail in CI but pass locally

**Solutions:**
1. Check OS-specific code paths
2. Verify Go version matches CI
3. Check for race conditions: `make test-race`
4. Review test logs in GitHub Actions

### Build Failures

**Symptoms:** Build fails for specific platform

**Solutions:**
1. Check for platform-specific code
2. Verify build tags
3. Test locally with:
   ```bash
   GOOS=linux GOARCH=arm64 go build ./cmd
   ```

### Coverage Failures

**Symptoms:** Coverage below 70% threshold

**Solutions:**
1. Generate coverage report: `make test-coverage`
2. Open `coverage.html` to see uncovered code
3. Add tests for uncovered packages
4. Focus on critical paths first

### Docker Build Failures

**Symptoms:** Docker image build fails

**Solutions:**
1. Test locally: `make docker`
2. Check Dockerfile syntax
3. Verify base image availability
4. Review build logs for errors

### Release Not Created

**Symptoms:** Tag pushed but release not created

**Solutions:**
1. Verify tag format: `v1.2.3` (not `1.2.3`)
2. Check workflow run in Actions tab
3. Review release workflow logs
4. Ensure GITHUB_TOKEN has permissions

### Workflow Syntax Errors

**Symptoms:** Workflow fails to run

**Solutions:**
1. Validate YAML syntax:
   ```bash
   python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"
   ```
2. Use GitHub's workflow editor
3. Check for indentation errors
4. Verify action versions exist

## Best Practices

### For Contributors

1. Run `make ci` before pushing
2. Write tests for new features
3. Keep coverage above 70%
4. Follow Go conventions
5. Update documentation

### For Reviewers

1. Check test coverage
2. Verify security implications
3. Review performance impact
4. Test locally if needed
5. Ensure documentation updated

### For Maintainers

1. Keep workflows updated
2. Monitor dependency updates
3. Review security alerts
4. Update Go versions regularly
5. Maintain changelog

## Metrics and Monitoring

### GitHub Actions

- **Build Time:** Target < 10 minutes
- **Test Success Rate:** > 95%
- **Coverage:** > 70%

### Docker Images

- **Image Size:** Target < 50MB
- **Build Time:** Target < 5 minutes
- **Vulnerability Count:** 0 critical/high

## Support

- **Issues:** Open GitHub issue
- **Discussions:** GitHub Discussions
- **Documentation:** See CONTRIBUTING.md
- **Security:** See SECURITY.md

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/actions)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Semantic Versioning](https://semver.org/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [Security Best Practices](https://github.blog/security/)
