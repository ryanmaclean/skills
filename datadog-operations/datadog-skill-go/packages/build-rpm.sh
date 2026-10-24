#!/bin/bash
# Build RPM package for datadog-cli
# Usage: ./packages/build-rpm.sh [version] [architecture]

set -e

VERSION=${1:-"0.1.0"}
ARCH=${2:-"x86_64"}  # x86_64 or aarch64
PACKAGE_NAME="datadog-cli"

echo "========================================"
echo "Building RPM package"
echo "========================================"
echo "Package: ${PACKAGE_NAME}"
echo "Version: ${VERSION}"
echo "Architecture: ${ARCH}"
echo ""

# Check if rpmbuild is available
if ! command -v rpmbuild &> /dev/null; then
    echo "Error: rpmbuild not found"
    echo "Please install rpm-build:"
    echo "  # RedHat/CentOS/Fedora:"
    echo "  sudo yum install rpm-build"
    echo "  # Ubuntu/Debian:"
    echo "  sudo apt-get install rpm"
    echo "  # macOS:"
    echo "  brew install rpm"
    exit 1
fi

# Create RPM build directories
mkdir -p ~/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Copy spec file
cp packages/rpm/datadog-cli.spec ~/rpmbuild/SPECS/

# Update version in spec file
sed -i.bak "s/Version:        0.1.0/Version:        ${VERSION}/" ~/rpmbuild/SPECS/datadog-cli.spec
rm ~/rpmbuild/SPECS/datadog-cli.spec.bak

# Determine binary name based on architecture
if [ "${ARCH}" = "x86_64" ]; then
    BINARY_NAME="dd-linux-amd64"
elif [ "${ARCH}" = "aarch64" ]; then
    BINARY_NAME="dd-linux-arm64"
else
    echo "Error: Unsupported architecture ${ARCH}"
    exit 1
fi

# Download or copy binary
if [ -f "bin/${BINARY_NAME}" ]; then
    echo "Using binary from bin/${BINARY_NAME}"
    cp "bin/${BINARY_NAME}" ~/rpmbuild/SOURCES/dd-linux-${ARCH}
elif [ -f "${BINARY_NAME}" ]; then
    echo "Using binary from ./${BINARY_NAME}"
    cp "${BINARY_NAME}" ~/rpmbuild/SOURCES/dd-linux-${ARCH}
else
    echo "Downloading binary from GitHub..."
    curl -L "https://github.com/yourusername/datadog-cli-go/releases/download/v${VERSION}/${BINARY_NAME}" \
         -o ~/rpmbuild/SOURCES/dd-linux-${ARCH}
fi

# Copy shell completions to SOURCES
mkdir -p ~/rpmbuild/SOURCES/completions
if [ -f "completions/dd.bash" ]; then
    cp completions/dd.bash ~/rpmbuild/SOURCES/completions/
else
    echo "Warning: completions/dd.bash not found"
fi

if [ -f "completions/dd.zsh" ]; then
    cp completions/dd.zsh ~/rpmbuild/SOURCES/completions/
else
    echo "Warning: completions/dd.zsh not found"
fi

# Build the RPM
echo ""
echo "Building RPM package..."
rpmbuild -bb ~/rpmbuild/SPECS/datadog-cli.spec

# Find and copy the built RPM
RPM_FILE=$(find ~/rpmbuild/RPMS -name "${PACKAGE_NAME}-${VERSION}-*.rpm" | head -1)

if [ -z "${RPM_FILE}" ]; then
    echo "Error: RPM file not found in ~/rpmbuild/RPMS"
    exit 1
fi

# Copy to packages directory
mkdir -p packages
cp "${RPM_FILE}" packages/
OUTPUT_FILE=$(basename "${RPM_FILE}")

echo ""
echo "========================================"
echo "Package built successfully!"
echo "========================================"
echo "Output: packages/${OUTPUT_FILE}"
echo ""
echo "To install locally:"
echo "  sudo rpm -ivh packages/${OUTPUT_FILE}"
echo ""
echo "To upgrade:"
echo "  sudo rpm -Uvh packages/${OUTPUT_FILE}"
echo ""
echo "To verify:"
echo "  rpm -qpl packages/${OUTPUT_FILE}  # List contents"
echo "  rpm -qpi packages/${OUTPUT_FILE}  # Show info"
echo ""
echo "To test installation:"
echo "  sudo rpm -ivh packages/${OUTPUT_FILE}"
echo "  dd --version"
echo "  dd --help"
echo ""
echo "To uninstall:"
echo "  sudo rpm -e datadog-cli"
echo ""

# Generate SHA256 checksum
SHA256=$(sha256sum "packages/${OUTPUT_FILE}" | awk '{print $1}')
echo "SHA256: ${SHA256}"
echo "${SHA256}  ${OUTPUT_FILE}" > "packages/${OUTPUT_FILE}.sha256"
echo "Checksum saved to: packages/${OUTPUT_FILE}.sha256"
echo ""
