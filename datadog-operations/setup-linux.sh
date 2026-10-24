#!/bin/bash
# Datadog Skill Setup Script for Linux
# Supports: Ubuntu/Debian, RHEL/CentOS/Fedora, Arch

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           Datadog Skill Setup for Linux                      ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Detect package manager
detect_pkg_manager() {
    if command -v apt-get &> /dev/null; then
        echo "apt"
    elif command -v dnf &> /dev/null; then
        echo "dnf"
    elif command -v yum &> /dev/null; then
        echo "yum"
    elif command -v pacman &> /dev/null; then
        echo "pacman"
    elif command -v zypper &> /dev/null; then
        echo "zypper"
    else
        echo "unknown"
    fi
}

PKG_MANAGER=$(detect_pkg_manager)
echo -e "Detected package manager: ${BLUE}$PKG_MANAGER${NC}"
echo ""

# ═══════════════════════════════════════════════════════════════
# Step 1: Check/Install jq and bc
# ═══════════════════════════════════════════════════════════════

echo -e "${YELLOW}Step 1: Checking dependencies (jq, bc, curl)...${NC}"

install_packages() {
    local packages="$1"
    case $PKG_MANAGER in
        apt)
            sudo apt-get update -qq
            sudo apt-get install -y $packages
            ;;
        dnf)
            sudo dnf install -y $packages
            ;;
        yum)
            sudo yum install -y $packages
            ;;
        pacman)
            sudo pacman -S --noconfirm $packages
            ;;
        zypper)
            sudo zypper install -y $packages
            ;;
        *)
            echo -e "${RED}Unknown package manager. Please install manually: $packages${NC}"
            return 1
            ;;
    esac
}

# Check jq
if command -v jq &> /dev/null; then
    echo -e "  ${GREEN}✓${NC} jq is installed ($(jq --version))"
else
    echo -e "  ${YELLOW}!${NC} Installing jq..."
    install_packages "jq" || {
        echo "  Trying direct download..."
        mkdir -p ~/bin
        curl -L -o ~/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-amd64
        chmod +x ~/bin/jq
        export PATH="$HOME/bin:$PATH"
    }
    echo -e "  ${GREEN}✓${NC} jq installed"
fi

# Check bc
if command -v bc &> /dev/null; then
    echo -e "  ${GREEN}✓${NC} bc is installed"
else
    echo -e "  ${YELLOW}!${NC} Installing bc..."
    install_packages "bc"
    echo -e "  ${GREEN}✓${NC} bc installed"
fi

# Check curl
if command -v curl &> /dev/null; then
    echo -e "  ${GREEN}✓${NC} curl is installed"
else
    echo -e "  ${YELLOW}!${NC} Installing curl..."
    install_packages "curl"
    echo -e "  ${GREEN}✓${NC} curl installed"
fi

# ═══════════════════════════════════════════════════════════════
# Step 2: Set up Python virtual environment
# ═══════════════════════════════════════════════════════════════

echo ""
echo -e "${YELLOW}Step 2: Setting up Python environment...${NC}"

# Check python3
if ! command -v python3 &> /dev/null; then
    echo -e "  ${YELLOW}!${NC} Installing python3..."
    install_packages "python3 python3-venv python3-pip"
fi

if [ -d ".venv" ]; then
    echo -e "  ${GREEN}✓${NC} Virtual environment already exists"
else
    echo "  Creating virtual environment..."
    python3 -m venv .venv
    echo -e "  ${GREEN}✓${NC} Created .venv"
fi

echo "  Installing Python dependencies..."
source .venv/bin/activate
pip install -q -r python/requirements.txt
echo -e "  ${GREEN}✓${NC} Python dependencies installed"

# ═══════════════════════════════════════════════════════════════
# Step 3: Check Datadog credentials
# ═══════════════════════════════════════════════════════════════

echo ""
echo -e "${YELLOW}Step 3: Checking Datadog credentials...${NC}"

# Detect shell config file
if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
    SHELL_RC="$HOME/.zshrc"
elif [ -n "$BASH_VERSION" ]; then
    SHELL_RC="$HOME/.bashrc"
else
    SHELL_RC="$HOME/.profile"
fi

echo -e "  Shell config: ${BLUE}$SHELL_RC${NC}"

if [ -n "$DD_API_KEY" ] && [ -n "$DD_APP_KEY" ]; then
    echo -e "  ${GREEN}✓${NC} DD_API_KEY is set"
    echo -e "  ${GREEN}✓${NC} DD_APP_KEY is set"
    echo -e "  ${GREEN}✓${NC} DD_SITE: ${DD_SITE:-datadoghq.com}"
else
    echo -e "  ${YELLOW}!${NC} Datadog credentials not set"
    echo ""
    echo "  Add these to your $SHELL_RC:"
    echo ""
    echo -e "  ${BLUE}export DD_API_KEY=\"your_api_key\"${NC}"
    echo -e "  ${BLUE}export DD_APP_KEY=\"your_application_key\"${NC}"
    echo -e "  ${BLUE}export DD_SITE=\"datadoghq.com\"${NC}"
    echo ""
    echo "  Get keys from: Datadog → Organization Settings → API Keys / Application Keys"
    echo ""
    echo -e "  ${YELLOW}Important:${NC} Enable 'Actions API Access' on your app key for workflow features."
    echo ""
    
    read -p "  Would you like to set them now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo ""
        read -p "  Enter DD_API_KEY: " DD_API_KEY_INPUT
        read -p "  Enter DD_APP_KEY: " DD_APP_KEY_INPUT
        read -p "  Enter DD_SITE (default: datadoghq.com): " DD_SITE_INPUT
        DD_SITE_INPUT=${DD_SITE_INPUT:-datadoghq.com}
        
        echo "" >> "$SHELL_RC"
        echo "# Datadog API credentials (added by dd-skill-test setup)" >> "$SHELL_RC"
        echo "export DD_API_KEY=\"$DD_API_KEY_INPUT\"" >> "$SHELL_RC"
        echo "export DD_APP_KEY=\"$DD_APP_KEY_INPUT\"" >> "$SHELL_RC"
        echo "export DD_SITE=\"$DD_SITE_INPUT\"" >> "$SHELL_RC"
        
        export DD_API_KEY="$DD_API_KEY_INPUT"
        export DD_APP_KEY="$DD_APP_KEY_INPUT"
        export DD_SITE="$DD_SITE_INPUT"
        
        echo -e "  ${GREEN}✓${NC} Credentials added to $SHELL_RC"
    fi
fi

# ═══════════════════════════════════════════════════════════════
# Step 4: Validate setup
# ═══════════════════════════════════════════════════════════════

echo ""
echo -e "${YELLOW}Step 4: Validating setup...${NC}"

if [ -n "$DD_API_KEY" ] && [ -n "$DD_APP_KEY" ]; then
    DD_SITE=${DD_SITE:-datadoghq.com}
    VALIDATE=$(curl -s "https://api.${DD_SITE}/api/v1/validate" \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" 2>/dev/null)
    
    if echo "$VALIDATE" | jq -e '.valid == true' > /dev/null 2>&1; then
        echo -e "  ${GREEN}✓${NC} Datadog API credentials are valid"
    else
        echo -e "  ${RED}✗${NC} Datadog API credentials are invalid"
    fi
else
    echo -e "  ${YELLOW}!${NC} Skipping validation (credentials not set)"
fi

# ═══════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}Setup Complete!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo "Next steps:"
echo ""
echo "  1. Reload your shell:"
echo -e "     ${BLUE}source $SHELL_RC${NC}"
echo ""
echo "  2. Activate Python environment:"
echo -e "     ${BLUE}source .venv/bin/activate${NC}"
echo ""
echo "  3. Run the test suite:"
echo -e "     ${BLUE}./test-skills.sh${NC}"
echo ""
echo "  4. Open in Claude Code:"
echo -e "     ${BLUE}claude${NC}"
echo ""

