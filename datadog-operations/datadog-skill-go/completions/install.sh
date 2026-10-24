#!/bin/bash
# Installation script for dd shell completions
# Supports bash and zsh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_success() {
    echo -e "${GREEN}✓${NC} $1"
}

echo_error() {
    echo -e "${RED}✗${NC} $1"
}

echo_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Detect shell
detect_shell() {
    if [ -n "$BASH_VERSION" ]; then
        echo "bash"
    elif [ -n "$ZSH_VERSION" ]; then
        echo "zsh"
    else
        # Check SHELL environment variable
        case "$SHELL" in
            */bash) echo "bash" ;;
            */zsh) echo "zsh" ;;
            *) echo "unknown" ;;
        esac
    fi
}

# Install bash completion
install_bash() {
    echo_info "Installing bash completion..."

    # Try different locations (in order of preference)
    if [ -d "/usr/local/etc/bash_completion.d" ]; then
        # Homebrew on macOS
        sudo cp "${SCRIPT_DIR}/dd.bash" "/usr/local/etc/bash_completion.d/dd"
        echo_success "Installed to /usr/local/etc/bash_completion.d/dd"
    elif [ -d "/etc/bash_completion.d" ]; then
        # Linux standard location
        sudo cp "${SCRIPT_DIR}/dd.bash" "/etc/bash_completion.d/dd"
        echo_success "Installed to /etc/bash_completion.d/dd"
    elif [ -d "$HOME/.bash_completion.d" ]; then
        # User-local directory
        cp "${SCRIPT_DIR}/dd.bash" "$HOME/.bash_completion.d/dd"
        echo_success "Installed to $HOME/.bash_completion.d/dd"
    else
        # Create user-local directory and use it
        mkdir -p "$HOME/.bash_completion.d"
        cp "${SCRIPT_DIR}/dd.bash" "$HOME/.bash_completion.d/dd"

        # Add to .bashrc if not already there
        if ! grep -q ".bash_completion.d" "$HOME/.bashrc" 2>/dev/null; then
            echo "" >> "$HOME/.bashrc"
            echo "# Load bash completions" >> "$HOME/.bashrc"
            echo "for f in ~/.bash_completion.d/*; do source \$f; done" >> "$HOME/.bashrc"
            echo_info "Added completion loading to .bashrc"
        fi

        echo_success "Installed to $HOME/.bash_completion.d/dd"
    fi

    echo_info "Run 'source ~/.bashrc' or start a new shell to activate"
}

# Install zsh completion
install_zsh() {
    echo_info "Installing zsh completion..."

    # Try different locations (in order of preference)
    if [ -d "/usr/local/share/zsh/site-functions" ]; then
        # Homebrew on macOS
        sudo cp "${SCRIPT_DIR}/dd.zsh" "/usr/local/share/zsh/site-functions/_dd"
        echo_success "Installed to /usr/local/share/zsh/site-functions/_dd"
    elif [ -d "/usr/share/zsh/site-functions" ]; then
        # Linux standard location
        sudo cp "${SCRIPT_DIR}/dd.zsh" "/usr/share/zsh/site-functions/_dd"
        echo_success "Installed to /usr/share/zsh/site-functions/_dd"
    else
        # User-local directory
        mkdir -p "$HOME/.zsh/completions"
        cp "${SCRIPT_DIR}/dd.zsh" "$HOME/.zsh/completions/_dd"

        # Add to fpath in .zshrc if not already there
        if ! grep -q ".zsh/completions" "$HOME/.zshrc" 2>/dev/null; then
            echo "" >> "$HOME/.zshrc"
            echo "# Add custom completions to fpath" >> "$HOME/.zshrc"
            echo "fpath=(~/.zsh/completions \$fpath)" >> "$HOME/.zshrc"
            echo "autoload -Uz compinit && compinit" >> "$HOME/.zshrc"
            echo_info "Added completion loading to .zshrc"
        fi

        echo_success "Installed to $HOME/.zsh/completions/_dd"
    fi

    echo_info "Run 'source ~/.zshrc' or start a new shell to activate"
    echo_info "You may need to run 'rm -f ~/.zcompdump && compinit' to rebuild completion cache"
}

# Main installation logic
main() {
    echo "🚀 Datadog CLI Shell Completion Installer"
    echo ""

    # Check if completion files exist
    if [ ! -f "${SCRIPT_DIR}/dd.bash" ]; then
        echo_error "dd.bash not found in ${SCRIPT_DIR}"
        exit 1
    fi

    if [ ! -f "${SCRIPT_DIR}/dd.zsh" ]; then
        echo_error "dd.zsh not found in ${SCRIPT_DIR}"
        exit 1
    fi

    # Detect shell or allow user to specify
    if [ "$1" = "bash" ] || [ "$1" = "zsh" ]; then
        SHELL_TYPE="$1"
        echo_info "Installing for shell: $SHELL_TYPE (specified)"
    else
        SHELL_TYPE=$(detect_shell)
        echo_info "Detected shell: $SHELL_TYPE"
    fi

    case "$SHELL_TYPE" in
        bash)
            install_bash
            ;;
        zsh)
            install_zsh
            ;;
        unknown)
            echo_error "Could not detect shell type"
            echo ""
            echo "Usage: $0 [bash|zsh]"
            echo ""
            echo "Examples:"
            echo "  $0            # Auto-detect shell"
            echo "  $0 bash       # Install bash completion"
            echo "  $0 zsh        # Install zsh completion"
            exit 1
            ;;
    esac

    echo ""
    echo_success "Installation complete!"
    echo ""
    echo "Test it out:"
    echo "  dd <TAB><TAB>           # See all commands"
    echo "  dd apm --<TAB><TAB>     # See apm flags"
    echo "  dd logs --from <TAB>    # See time range suggestions"
}

# Run main function
main "$@"
