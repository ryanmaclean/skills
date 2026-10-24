# Binary Optimization Guide

This document explains the binary size optimizations applied to the Datadog CLI.

---

## Binary Size Comparison

### Before Optimization
- **Standard build**: `go build cmd/main.go`
- **Size**: ~16-17MB per binary
- **Total (6 platforms)**: ~100MB

### After Optimization
- **Optimized build**: `go build -ldflags="-s -w" -trimpath cmd/main.go`
- **Size**: ~11MB per binary
- **Total (6 platforms)**: ~66MB
- **Reduction**: **31% smaller** (5MB saved per binary)

---

## Optimization Techniques

### 1. Strip Debug Symbols (`-ldflags="-s"`)

Removes DWARF debug information from the binary.

**Effect**:
- Reduces binary size by ~2-3MB
- Makes debugging harder (use only for releases)
- No runtime performance impact

**Trade-off**: Can't use debuggers like delve on the binary.

### 2. Strip Symbol Table (`-ldflags="-w"`)

Removes the Go symbol table and debug information.

**Effect**:
- Reduces binary size by ~2-3MB
- Removes function names from stack traces
- No runtime performance impact

**Trade-off**: Stack traces show addresses instead of function names.

### 3. Trim File Paths (`-trimpath`)

Removes absolute file paths from the binary.

**Effect**:
- Reduces binary size slightly (~100-500KB)
- Makes builds reproducible
- Removes local path information

**Trade-off**: Error messages show relative paths only.

### 4. Disable CGO (`CGO_ENABLED=0`)

Ensures pure Go binary without C dependencies.

**Effect**:
- Fully static binary
- Cross-compilation friendly
- No libc dependencies

**Trade-off**: Can't use C libraries (not needed for this project).

---

## Build Commands

### Development Build (Debug-Friendly)
```bash
go build -o dd cmd/main.go
```
- Size: ~16MB
- Full debug symbols
- Stack traces with function names
- Local file paths

### Production Build (Optimized)
```bash
go build -ldflags="-s -w" -trimpath -o dd cmd/main.go
```
- Size: ~11MB (31% smaller)
- No debug symbols
- Reproducible builds
- Smaller download/deployment

### Release Build (With Version Info)
```bash
VERSION=v1.0.0
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

go build \
  -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.Date=${DATE}" \
  -trimpath \
  -o dd \
  cmd/main.go
```
- Size: ~11MB
- Version information injected
- Production-ready

---

## Advanced Optimization (Optional)

### UPX Compression

UPX can further compress the binary by ~60-70%.

**Install UPX**:
```bash
# macOS
brew install upx

# Linux
apt-get install upx-ucl

# Windows
choco install upx
```

**Compress Binary**:
```bash
# Build optimized
go build -ldflags="-s -w" -trimpath -o dd cmd/main.go

# Compress with UPX
upx --best --lzma dd

# Result: ~3-4MB (70% smaller than optimized!)
```

**Trade-offs**:
- **Slower startup**: ~10-50ms decompression time
- **More memory**: Temporary decompression overhead
- **Antivirus flags**: Some AVs detect UPX as suspicious
- **macOS codesigning**: Breaks code signature

**Recommendation**: Use for Linux/Windows only, skip for macOS.

---

## Size Analysis

### What's in the Binary?

Use Go tooling to analyze:

```bash
# Build with symbol table for analysis
go build -o dd cmd/main.go

# Analyze binary size
go tool nm -size dd | sort -rn | head -20

# Or use go-binsize-treemap
go get -u github.com/nikolaydubina/go-binsize-treemap
go build -o dd cmd/main.go
go tool nm -size dd | go-binsize-treemap > treemap.svg
```

### Size Breakdown (Approximate)

For our ~11MB optimized binary:

| Component | Size | Percentage |
|-----------|------|------------|
| Go runtime | ~2MB | 18% |
| Standard library | ~3MB | 27% |
| Dependencies (dd-trace-go, datadog-go, go-git) | ~4MB | 36% |
| Our code (commands, client, observability) | ~2MB | 18% |

---

## Platform-Specific Optimizations

### macOS
- **Code signing**: Required for Gatekeeper
- **Notarization**: Required for distribution
- **Universal binary**: Can combine amd64 + arm64 with `lipo`

### Linux
- **Static linking**: Already done with CGO_ENABLED=0
- **MUSL**: Could use alpine for even smaller binaries
- **Strip**: Can use system `strip` command for extra reduction

### Windows
- **Resource section**: Could add icon and metadata
- **Manifest**: Could embed for UAC control
- **Authenticode**: Should sign for distribution

---

## Recommendations

### For Development
- Use standard build (go build)
- Keep debug symbols
- Faster iteration

### For CI/CD
- Use optimized build (-ldflags="-s -w" -trimpath)
- Inject version info
- Fast enough, much smaller

### For Distribution
- Use optimized build
- Consider UPX for Linux/Windows
- Skip UPX for macOS (codesigning issues)
- Sign binaries (macOS: codesign, Windows: Authenticode)

---

## Script Usage

We provide `scripts/build-optimized.sh` for convenient builds:

```bash
# Build all platforms (dev version)
./scripts/build-optimized.sh

# Build all platforms (release version)
./scripts/build-optimized.sh v1.0.0

# Verify sizes
ls -lh bin/
```

This script builds optimized binaries for all 6 platforms with version injection.

---

## Performance Impact

Binary size optimization has **zero runtime performance impact**.

The optimizations only affect:
- **Download time**: Smaller = faster download
- **Disk space**: Smaller = less storage
- **Memory**: Slightly less (smaller binary footprint)
- **Startup time**: No measurable difference

**Benchmarks** (macOS M1):
```
Standard build (16MB):     startup ~3ms
Optimized build (11MB):    startup ~3ms
UPX compressed (4MB):      startup ~15ms (decompression overhead)
```

---

## Summary

**Recommended Settings for Release**:
```bash
go build \
  -ldflags="-s -w -X main.Version=${VERSION}" \
  -trimpath \
  -o dd \
  cmd/main.go
```

**Results**:
- **31% size reduction** (16MB → 11MB)
- **No performance loss**
- **Production ready**
- **Reproducible builds**

**Optional UPX** (Linux/Windows only):
- **Further 70% reduction** (11MB → 3-4MB)
- **Small startup cost** (~10-50ms)
- **Not recommended for macOS** (codesigning)

---

**Date**: January 21, 2026
**Go Version**: 1.25.6
**Optimization Level**: Production-ready
