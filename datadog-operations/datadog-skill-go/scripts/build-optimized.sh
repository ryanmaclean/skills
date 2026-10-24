#!/bin/bash
# Build optimized binaries for all platforms

set -e

VERSION=${1:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.Date=${DATE}"

echo "Building optimized binaries..."
echo "Version: ${VERSION}"
echo "Commit: ${COMMIT}"
echo "Date: ${DATE}"
echo ""

mkdir -p bin

# macOS Intel
echo "Building darwin/amd64..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="${LDFLAGS}" \
  -trimpath \
  -o bin/dd-darwin-amd64 \
  cmd/main.go

# macOS Apple Silicon
echo "Building darwin/arm64..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build \
  -ldflags="${LDFLAGS}" \
  -trimpath \
  -o bin/dd-darwin-arm64 \
  cmd/main.go

# Linux AMD64
echo "Building linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="${LDFLAGS}" \
  -trimpath \
  -o bin/dd-linux-amd64 \
  cmd/main.go

# Linux ARM64
echo "Building linux/arm64..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
  -ldflags="${LDFLAGS}" \
  -trimpath \
  -o bin/dd-linux-arm64 \
  cmd/main.go

# Windows AMD64
echo "Building windows/amd64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags="${LDFLAGS}" \
  -trimpath \
  -o bin/dd-windows-amd64.exe \
  cmd/main.go

# Windows ARM64
echo "Building windows/arm64..."
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build \
  -ldflags="${LDFLAGS}" \
  -trimpath \
  -o bin/dd-windows-arm64.exe \
  cmd/main.go

echo ""
echo "Build complete! Binary sizes:"
ls -lh bin/
