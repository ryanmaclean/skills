#!/usr/bin/env bash
#
# Datadog CLI Plugin Installer for Claude Code
#
# This script installs the Datadog CLI plugin into Claude Code's plugin directory.
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  Datadog CLI Plugin Installer for Claude Code${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo ""
}

# Check if Claude Code is installed
check_claude_code() {
    if [ ! -d "$HOME/.claude" ]; then
        print_error "Claude Code not found. Please install Claude Code first."
        exit 1
    fi
    print_success "Claude Code installation found"
}

# Detect plugin source directory
detect_source_dir() {
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local plugin_root="$(cd "$script_dir/.." && pwd)"
    echo "$plugin_root"
}

# Check if Datadog CLI binary exists
check_cli_binary() {
    local repo_root="$(cd "$1/../.." && pwd)"
    local binary_path=""

    # Detect architecture and OS
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
    esac

    case "$os" in
        darwin) os="darwin" ;;
        linux) os="linux" ;;
        *)
            print_error "Unsupported OS: $os"
            exit 1
            ;;
    esac

    binary_path="$repo_root/bin/dd-${os}-${arch}"

    if [ ! -f "$binary_path" ]; then
        print_warning "Datadog CLI binary not found at: $binary_path"
        print_info "Building CLI binary..."

        if [ ! -f "$repo_root/Makefile" ]; then
            print_error "Makefile not found. Cannot build CLI."
            exit 1
        fi

        cd "$repo_root"
        make build

        if [ ! -f "$binary_path" ]; then
            print_error "Failed to build CLI binary"
            exit 1
        fi
    fi

    print_success "Datadog CLI binary found: $binary_path"
    echo "$binary_path"
}

# Install plugin
install_plugin() {
    local source_dir="$1"
    local target_dir="$HOME/.claude/plugins/user/datadog-cli"

    print_info "Installing plugin to: $target_dir"

    # Create target directory
    mkdir -p "$target_dir"

    # Copy plugin files
    print_info "Copying plugin files..."
    cp -r "$source_dir/.claude-plugin" "$target_dir/"
    cp -r "$source_dir/commands" "$target_dir/"
    cp -r "$source_dir/scripts" "$target_dir/"
    cp "$source_dir/README.md" "$target_dir/"

    print_success "Plugin files installed"
}

# Check for credentials
check_credentials() {
    local has_api_key=false
    local has_app_key=false

    if [ ! -z "${DD_API_KEY:-}" ]; then
        has_api_key=true
    fi

    if [ ! -z "${DD_APP_KEY:-}" ]; then
        has_app_key=true
    fi

    if [ "$has_api_key" = true ] && [ "$has_app_key" = true ]; then
        print_success "Datadog credentials found in environment"
        return 0
    fi

    print_warning "Datadog credentials not found in environment"
    return 1
}

# Prompt for credentials
prompt_credentials() {
    echo ""
    print_info "Datadog credentials setup"
    echo ""

    read -p "Enter Datadog API Key: " api_key
    read -p "Enter Datadog App Key: " app_key
    read -p "Enter Datadog Site (default: datadoghq.com): " site
    site=${site:-datadoghq.com}

    echo ""
    print_info "Add these to your shell profile (~/.zshrc or ~/.bashrc):"
    echo ""
    echo "export DD_API_KEY=\"$api_key\""
    echo "export DD_APP_KEY=\"$app_key\""
    echo "export DD_SITE=\"$site\""
    echo "export DD_CLI_PATH=\"$cli_binary\""
    echo ""
}

# Test installation
test_installation() {
    local cli_binary="$1"

    print_info "Testing CLI binary..."

    if [ ! -x "$cli_binary" ]; then
        print_warning "CLI binary not executable, making it executable..."
        chmod +x "$cli_binary"
    fi

    if "$cli_binary" version >/dev/null 2>&1; then
        print_success "CLI binary is working"
        "$cli_binary" version
    else
        print_warning "CLI binary test failed (credentials may not be set)"
    fi
}

# Main installation
main() {
    print_header

    # Check prerequisites
    check_claude_code

    # Detect source directory
    local source_dir=$(detect_source_dir)
    print_info "Plugin source: $source_dir"

    # Check and build CLI binary if needed
    local cli_binary=$(check_cli_binary "$source_dir")

    # Install plugin
    install_plugin "$source_dir"

    # Check credentials
    if ! check_credentials; then
        echo ""
        read -p "Would you like to set up credentials now? (y/N): " setup_creds
        if [[ "$setup_creds" =~ ^[Yy]$ ]]; then
            prompt_credentials
        else
            print_warning "Remember to set DD_API_KEY, DD_APP_KEY, DD_SITE, and DD_CLI_PATH"
        fi
    fi

    # Test installation
    echo ""
    test_installation "$cli_binary"

    # Success message
    echo ""
    print_success "Installation complete!"
    echo ""
    print_info "Next steps:"
    echo "  1. Set environment variables (if not already set)"
    echo "  2. Restart Claude Code"
    echo "  3. Try: 'Check the health of my api service using Datadog'"
    echo ""
    print_info "For more information, see: $HOME/.claude/plugins/user/datadog-cli/README.md"
    echo ""
}

# Run main
main "$@"
