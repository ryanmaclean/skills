#!/bin/bash
# Generate SHA256 checksums for Homebrew formula
# Run this after creating a GitHub release with binaries

set -e

VERSION=${1:-"0.1.0"}
GITHUB_USER=${2:-"yourusername"}
REPO="datadog-cli-go"

echo "Generating SHA256 checksums for version v${VERSION}"
echo "GitHub: ${GITHUB_USER}/${REPO}"
echo ""

# Download binaries from GitHub release
echo "Downloading binaries..."

# macOS Intel (amd64)
AMD64_URL="https://github.com/${GITHUB_USER}/${REPO}/releases/download/v${VERSION}/dd-darwin-amd64"
echo "Downloading ${AMD64_URL}"
curl -sL "${AMD64_URL}" -o dd-darwin-amd64

# macOS Apple Silicon (arm64)
ARM64_URL="https://github.com/${GITHUB_USER}/${REPO}/releases/download/v${VERSION}/dd-darwin-arm64"
echo "Downloading ${ARM64_URL}"
curl -sL "${ARM64_URL}" -o dd-darwin-arm64

echo ""
echo "Calculating checksums..."
echo ""

# Calculate SHA256
AMD64_SHA256=$(shasum -a 256 dd-darwin-amd64 | awk '{print $1}')
ARM64_SHA256=$(shasum -a 256 dd-darwin-arm64 | awk '{print $1}')

# Display results
echo "================================"
echo "SHA256 Checksums for v${VERSION}"
echo "================================"
echo ""
echo "AMD64 (Intel):"
echo "  sha256 \"${AMD64_SHA256}\""
echo ""
echo "ARM64 (Apple Silicon):"
echo "  sha256 \"${ARM64_SHA256}\""
echo ""

# Show how to update the formula
echo "================================"
echo "Update Formula/datadog-cli.rb:"
echo "================================"
echo ""
echo "1. Set version:"
echo "   version \"${VERSION}\""
echo ""
echo "2. Update ARM64 SHA256:"
echo "   if Hardware::CPU.arm?"
echo "     url \"https://github.com/${GITHUB_USER}/${REPO}/releases/download/v${VERSION}/dd-darwin-arm64\""
echo "     sha256 \"${ARM64_SHA256}\""
echo ""
echo "3. Update AMD64 SHA256:"
echo "   else"
echo "     url \"https://github.com/${GITHUB_USER}/${REPO}/releases/download/v${VERSION}/dd-darwin-amd64\""
echo "     sha256 \"${AMD64_SHA256}\""
echo ""

# Clean up
rm -f dd-darwin-amd64 dd-darwin-arm64

echo "✓ Done! Update the formula with the checksums above."
