# Pants Build System Setup

This project is configured to use [Pants](https://www.pantsbuild.org/) build system for Go.

## Installation

Install Pants using one of these methods:

### Option 1: Using pip (Recommended)
```bash
pip install pantsbuild.pants
```

### Option 2: Using Homebrew (macOS)
```bash
brew install pantsbuild/tap/pants
```

### Option 3: Download Wrapper Script
```bash
# Download the pants wrapper script
curl -L -o pants https://static.pants.dev/bin/2.25.0/pants
chmod +x pants

# Run pants
./pants --version
```

## Configuration

Pants is already configured for this project:
- **pants.toml**: Main configuration file
  - Go backend enabled
  - golangci-lint configured for linting
  - Minimum Go version: 1.25

- **BUILD files**: Present in all source directories
  - cmd/BUILD: Main binary target
  - internal/*/BUILD: Package targets

## Common Commands

### Build
```bash
# Build the main binary
pants package cmd:dd

# Build all targets
pants package ::
```

### Test
```bash
# Run all tests
pants test ::

# Test specific package
pants test internal/commands:
```

### Lint
```bash
# Lint all Go code
pants lint ::

# Lint specific file
pants lint internal/commands/dora.go
```

### Format
```bash
# Format all Go code
pants fmt ::
```

### Check BUILD files
```bash
# Verify BUILD files are up to date
pants tailor --check ::

# Update BUILD files
pants tailor ::
```

## Advantages Over go build

**Caching:**
- Pants caches build artifacts
- Only rebuilds changed code
- Significantly faster incremental builds

**Dependency Management:**
- Fine-grained dependency tracking
- Parallel builds across packages
- Better monorepo support

**Multi-Language:**
- Ready for Python/Java/Scala if needed
- Unified build system
- Cross-language dependencies

**Linting & Formatting:**
- Integrated golangci-lint
- Automatic code formatting
- Consistent code quality

## Build Comparison

**Traditional:**
```bash
go build -o bin/dd-darwin-arm64 cmd/main.go
go build -o bin/dd-linux-amd64 cmd/main.go
# ...repeat for each platform
```

**With Pants:**
```bash
pants package cmd:dd
# Cached, parallel, cross-platform builds
```

## CI/CD Integration

Example GitHub Actions workflow:

```yaml
- name: Set up Pants
  uses: pantsbuild/actions/init-pants@main
  with:
    pants-version: 2.25.0

- name: Build
  run: pants package ::

- name: Test
  run: pants test ::

- name: Lint
  run: pants lint ::
```

## References

- [Pants Documentation](https://www.pantsbuild.org/)
- [Pants Go Support](https://www.pantsbuild.org/docs/go)
- [Pants Go Configuration](https://www.pantsbuild.org/stable/reference/subsystems/golang)
- [Example Go Project](https://github.com/pantsbuild/example-golang)

## Current Status

✅ Pants configured and ready to use
✅ BUILD files generated
✅ Linting enabled (golangci-lint v1.61.0)
✅ Go 1.25 minimum version set

**Note:** Pants is optional. The project continues to work with standard `go build` commands.
