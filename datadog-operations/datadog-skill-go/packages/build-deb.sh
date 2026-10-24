#!/bin/bash
# Build Debian (.deb) package for datadog-cli
# Usage: ./packages/build-deb.sh [version] [architecture]

set -e

VERSION=${1:-"0.1.0"}
ARCH=${2:-"amd64"}  # amd64 or arm64
PACKAGE_NAME="datadog-cli"
BUILD_DIR="packages/build/deb"

echo "========================================"
echo "Building Debian package"
echo "========================================"
echo "Package: ${PACKAGE_NAME}"
echo "Version: ${VERSION}"
echo "Architecture: ${ARCH}"
echo ""

# Clean previous builds
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}/${PACKAGE_NAME}_${VERSION}_${ARCH}"

# Create package structure
PACKAGE_ROOT="${BUILD_DIR}/${PACKAGE_NAME}_${VERSION}_${ARCH}"
mkdir -p "${PACKAGE_ROOT}/DEBIAN"
mkdir -p "${PACKAGE_ROOT}/usr/bin"
mkdir -p "${PACKAGE_ROOT}/etc/bash_completion.d"
mkdir -p "${PACKAGE_ROOT}/usr/share/zsh/vendor-completions"

# Copy control files
cp packages/debian/DEBIAN/control "${PACKAGE_ROOT}/DEBIAN/"
cp packages/debian/DEBIAN/postinst "${PACKAGE_ROOT}/DEBIAN/"
chmod 755 "${PACKAGE_ROOT}/DEBIAN/postinst"

# Update architecture in control file
sed -i.bak "s/Architecture: amd64/Architecture: ${ARCH}/" "${PACKAGE_ROOT}/DEBIAN/control"
sed -i.bak "s/Version: 0.1.0/Version: ${VERSION}/" "${PACKAGE_ROOT}/DEBIAN/control"
rm "${PACKAGE_ROOT}/DEBIAN/control.bak"

# Determine binary name based on architecture
if [ "${ARCH}" = "amd64" ]; then
    BINARY_NAME="dd-linux-amd64"
elif [ "${ARCH}" = "arm64" ]; then
    BINARY_NAME="dd-linux-arm64"
else
    echo "Error: Unsupported architecture ${ARCH}"
    exit 1
fi

# Copy binary (from release or build)
if [ -f "bin/${BINARY_NAME}" ]; then
    echo "Using binary from bin/${BINARY_NAME}"
    cp "bin/${BINARY_NAME}" "${PACKAGE_ROOT}/usr/bin/dd"
elif [ -f "${BINARY_NAME}" ]; then
    echo "Using binary from ./${BINARY_NAME}"
    cp "${BINARY_NAME}" "${PACKAGE_ROOT}/usr/bin/dd"
else
    echo "Error: Binary ${BINARY_NAME} not found"
    echo "Please download or build the binary first:"
    echo "  curl -L https://github.com/yourusername/datadog-cli-go/releases/download/v${VERSION}/${BINARY_NAME} -o ${BINARY_NAME}"
    exit 1
fi

chmod 755 "${PACKAGE_ROOT}/usr/bin/dd"

# Copy shell completions
if [ -f "completions/dd.bash" ]; then
    cp completions/dd.bash "${PACKAGE_ROOT}/etc/bash_completion.d/dd"
    chmod 644 "${PACKAGE_ROOT}/etc/bash_completion.d/dd"
else
    echo "Warning: completions/dd.bash not found"
fi

if [ -f "completions/dd.zsh" ]; then
    cp completions/dd.zsh "${PACKAGE_ROOT}/usr/share/zsh/vendor-completions/_dd"
    chmod 644 "${PACKAGE_ROOT}/usr/share/zsh/vendor-completions/_dd"
else
    echo "Warning: completions/dd.zsh not found"
fi

# Build the package
echo ""
echo "Building package..."
dpkg-deb --build "${PACKAGE_ROOT}"

# Move to packages directory
OUTPUT_FILE="${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
mv "${BUILD_DIR}/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb" "packages/${OUTPUT_FILE}"

echo ""
echo "========================================"
echo "Package built successfully!"
echo "========================================"
echo "Output: packages/${OUTPUT_FILE}"
echo ""
echo "To install locally:"
echo "  sudo dpkg -i packages/${OUTPUT_FILE}"
echo ""
echo "To install dependencies if needed:"
echo "  sudo apt-get install -f"
echo ""
echo "To verify:"
echo "  dpkg -c packages/${OUTPUT_FILE}  # List contents"
echo "  dpkg -I packages/${OUTPUT_FILE}  # Show info"
echo ""
echo "To test installation:"
echo "  sudo dpkg -i packages/${OUTPUT_FILE}"
echo "  dd --version"
echo "  dd --help"
echo ""
echo "To uninstall:"
echo "  sudo dpkg -r datadog-cli"
echo ""

# Generate SHA256 checksum
SHA256=$(sha256sum "packages/${OUTPUT_FILE}" | awk '{print $1}')
echo "SHA256: ${SHA256}"
echo "${SHA256}  ${OUTPUT_FILE}" > "packages/${OUTPUT_FILE}.sha256"
echo "Checksum saved to: packages/${OUTPUT_FILE}.sha256"
echo ""
