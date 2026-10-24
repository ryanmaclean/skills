#!/bin/bash
# Datadog Skill Setup Script for macOS
# Run this once to set up everything needed for the skill

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           Datadog Skill Setup for macOS                      ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ═══════════════════════════════════════════════════════════════
# Step 1: Check/Install jq
# ═══════════════════════════════════════════════════════════════

echo -e "${YELLOW}Step 1: Checking jq installation...${NC}"

if command -v jq &> /dev/null; then
    echo -e "  ${GREEN}✓${NC} jq is already installed ($(jq --version))"
else
    echo -e "  ${YELLOW}!${NC} jq not found. Installing..."
    
    # Try different methods
    if command -v brew &> /dev/null; then
        echo "  Installing via Homebrew..."
        brew install jq
    elif command -v port &> /dev/null; then
        echo "  Installing via MacPorts..."
        sudo port install jq
    else
        echo "  Installing via direct download..."
        mkdir -p ~/bin
        
        # Detect architecture
        ARCH=$(uname -m)
        if [ "$ARCH" = "arm64" ]; then
            JQ_URL="https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-macos-arm64"
        else
            JQ_URL="https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-macos-amd64"
        fi
        
        curl -L -o ~/bin/jq "$JQ_URL"
        chmod +x ~/bin/jq
        
        # Add to PATH if not already
        if ! echo "$PATH" | grep -q "$HOME/bin"; then
            echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
            export PATH="$HOME/bin:$PATH"
        fi
    fi
    
    if command -v jq &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} jq installed successfully"
    else
        echo -e "  ${RED}✗${NC} Failed to install jq"
        exit 1
    fi
fi

# ═══════════════════════════════════════════════════════════════
# Step 2: Set up Python virtual environment
# ═══════════════════════════════════════════════════════════════

echo ""
echo -e "${YELLOW}Step 2: Setting up Python environment...${NC}"

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

if [ -n "$DD_API_KEY" ] && [ -n "$DD_APP_KEY" ]; then
    echo -e "  ${GREEN}✓${NC} DD_API_KEY is set"
    echo -e "  ${GREEN}✓${NC} DD_APP_KEY is set"
    echo -e "  ${GREEN}✓${NC} DD_SITE: ${DD_SITE:-datadoghq.com}"
else
    echo -e "  ${YELLOW}!${NC} Datadog credentials not set"
    echo ""
    echo "  Add these to your ~/.zshrc:"
    echo ""
    echo -e "  ${BLUE}export DD_API_KEY=\"your_api_key\"${NC}"
    echo -e "  ${BLUE}export DD_APP_KEY=\"your_application_key\"${NC}"
    echo -e "  ${BLUE}export DD_SITE=\"datadoghq.com\"${NC}"
    echo ""
    echo "  Get keys from: Datadog → Organization Settings → API Keys / Application Keys"
    echo ""
    echo -e "  ${YELLOW}Important:${NC} Enable 'Actions API Access' on your app key for workflow features:"
    echo "  Datadog → Organization Settings → Application Keys → Click key → Enable 'Actions API Access'"
    echo ""
    
    read -p "  Would you like to set them now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo ""
        read -p "  Enter DD_API_KEY: " DD_API_KEY_INPUT
        read -p "  Enter DD_APP_KEY: " DD_APP_KEY_INPUT
        read -p "  Enter DD_SITE (default: datadoghq.com): " DD_SITE_INPUT
        DD_SITE_INPUT=${DD_SITE_INPUT:-datadoghq.com}
        
        echo "" >> ~/.zshrc
        echo "# Datadog API credentials (added by dd-skill-test setup)" >> ~/.zshrc
        echo "export DD_API_KEY=\"$DD_API_KEY_INPUT\"" >> ~/.zshrc
        echo "export DD_APP_KEY=\"$DD_APP_KEY_INPUT\"" >> ~/.zshrc
        echo "export DD_SITE=\"$DD_SITE_INPUT\"" >> ~/.zshrc
        
        export DD_API_KEY="$DD_API_KEY_INPUT"
        export DD_APP_KEY="$DD_APP_KEY_INPUT"
        export DD_SITE="$DD_SITE_INPUT"
        
        echo -e "  ${GREEN}✓${NC} Credentials added to ~/.zshrc"
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
        echo "  Please check your API and APP keys"
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
echo -e "     ${BLUE}source ~/.zshrc${NC}"
echo ""
echo "  2. Activate Python environment (if using Python scripts):"
echo -e "     ${BLUE}source .venv/bin/activate${NC}"
echo ""
echo "  3. Run the test suite:"
echo -e "     ${BLUE}./test-skills.sh${NC}"
echo ""
echo "  4. Open in Claude Code:"
echo -e "     ${BLUE}claude${NC}"
echo ""
echo "  5. Try the skill:"
echo -e "     ${BLUE}\"Check my SLO status\"${NC}"
echo -e "     ${BLUE}\"List monitors in alert\"${NC}"
echo -e "     ${BLUE}\"Search for error logs\"${NC}"
echo ""

