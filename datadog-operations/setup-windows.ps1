# Datadog Skill Setup Script for Windows
# Run in PowerShell as Administrator (for Chocolatey) or regular user (for winget/manual)
# Usage: .\setup-windows.ps1

$ErrorActionPreference = "Stop"

Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Blue
Write-Host "║           Datadog Skill Setup for Windows                    ║" -ForegroundColor Blue
Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Blue
Write-Host ""

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

# ═══════════════════════════════════════════════════════════════
# Step 1: Check/Install jq
# ═══════════════════════════════════════════════════════════════

Write-Host "Step 1: Checking jq installation..." -ForegroundColor Yellow

$jqPath = Get-Command jq -ErrorAction SilentlyContinue

if ($jqPath) {
    Write-Host "  ✓ jq is installed" -ForegroundColor Green
} else {
    Write-Host "  ! jq not found. Installing..." -ForegroundColor Yellow
    
    # Try winget first (Windows 10/11)
    $winget = Get-Command winget -ErrorAction SilentlyContinue
    if ($winget) {
        Write-Host "  Installing via winget..."
        winget install jqlang.jq --silent
    }
    # Try Chocolatey
    elseif (Get-Command choco -ErrorAction SilentlyContinue) {
        Write-Host "  Installing via Chocolatey..."
        choco install jq -y
    }
    # Manual download
    else {
        Write-Host "  Downloading jq manually..."
        $jqDir = "$env:USERPROFILE\bin"
        New-Item -ItemType Directory -Force -Path $jqDir | Out-Null
        
        $jqUrl = "https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-windows-amd64.exe"
        Invoke-WebRequest -Uri $jqUrl -OutFile "$jqDir\jq.exe"
        
        # Add to PATH for current session
        $env:PATH = "$jqDir;$env:PATH"
        
        Write-Host "  Downloaded to $jqDir\jq.exe" -ForegroundColor Cyan
        Write-Host "  Add $jqDir to your PATH environment variable permanently" -ForegroundColor Yellow
    }
    
    Write-Host "  ✓ jq installed" -ForegroundColor Green
}

# ═══════════════════════════════════════════════════════════════
# Step 2: Check Python
# ═══════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "Step 2: Checking Python installation..." -ForegroundColor Yellow

$python = Get-Command python -ErrorAction SilentlyContinue
$python3 = Get-Command python3 -ErrorAction SilentlyContinue

if ($python -or $python3) {
    $pyCmd = if ($python3) { "python3" } else { "python" }
    $pyVersion = & $pyCmd --version 2>&1
    Write-Host "  ✓ Python found: $pyVersion" -ForegroundColor Green
    
    # Create virtual environment
    if (-not (Test-Path ".venv")) {
        Write-Host "  Creating virtual environment..."
        & $pyCmd -m venv .venv
        Write-Host "  ✓ Created .venv" -ForegroundColor Green
    } else {
        Write-Host "  ✓ Virtual environment already exists" -ForegroundColor Green
    }
    
    # Activate and install dependencies
    Write-Host "  Installing Python dependencies..."
    & ".venv\Scripts\Activate.ps1"
    & pip install -q -r python\requirements.txt
    Write-Host "  ✓ Python dependencies installed" -ForegroundColor Green
} else {
    Write-Host "  ✗ Python not found" -ForegroundColor Red
    Write-Host "  Install Python from: https://www.python.org/downloads/" -ForegroundColor Yellow
    Write-Host "  Or via winget: winget install Python.Python.3.12" -ForegroundColor Cyan
}

# ═══════════════════════════════════════════════════════════════
# Step 3: Check Datadog credentials
# ═══════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "Step 3: Checking Datadog credentials..." -ForegroundColor Yellow

if ($env:DD_API_KEY -and $env:DD_APP_KEY) {
    Write-Host "  ✓ DD_API_KEY is set" -ForegroundColor Green
    Write-Host "  ✓ DD_APP_KEY is set" -ForegroundColor Green
    $ddSite = if ($env:DD_SITE) { $env:DD_SITE } else { "datadoghq.com" }
    Write-Host "  ✓ DD_SITE: $ddSite" -ForegroundColor Green
} else {
    Write-Host "  ! Datadog credentials not set" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Set environment variables in PowerShell:" -ForegroundColor Cyan
    Write-Host '  $env:DD_API_KEY = "your_api_key"'
    Write-Host '  $env:DD_APP_KEY = "your_application_key"'
    Write-Host '  $env:DD_SITE = "datadoghq.com"'
    Write-Host ""
    Write-Host "  Or set permanently via System Properties > Environment Variables" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Get keys from: Datadog > Organization Settings > API Keys / Application Keys" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Important: Enable 'Actions API Access' on your app key for workflow features." -ForegroundColor Yellow
    Write-Host ""
    
    $response = Read-Host "  Would you like to set them now for this session? (y/n)"
    if ($response -eq 'y' -or $response -eq 'Y') {
        $env:DD_API_KEY = Read-Host "  Enter DD_API_KEY"
        $env:DD_APP_KEY = Read-Host "  Enter DD_APP_KEY"
        $ddSiteInput = Read-Host "  Enter DD_SITE (default: datadoghq.com)"
        if (-not $ddSiteInput) { $ddSiteInput = "datadoghq.com" }
        $env:DD_SITE = $ddSiteInput
        
        Write-Host "  ✓ Credentials set for this session" -ForegroundColor Green
        Write-Host ""
        Write-Host "  To make permanent, run these in an elevated PowerShell:" -ForegroundColor Yellow
        Write-Host "  [Environment]::SetEnvironmentVariable('DD_API_KEY', '$($env:DD_API_KEY)', 'User')"
        Write-Host "  [Environment]::SetEnvironmentVariable('DD_APP_KEY', '$($env:DD_APP_KEY)', 'User')"
        Write-Host "  [Environment]::SetEnvironmentVariable('DD_SITE', '$($env:DD_SITE)', 'User')"
    }
}

# ═══════════════════════════════════════════════════════════════
# Step 4: Validate setup
# ═══════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "Step 4: Validating setup..." -ForegroundColor Yellow

if ($env:DD_API_KEY -and $env:DD_APP_KEY) {
    $ddSite = if ($env:DD_SITE) { $env:DD_SITE } else { "datadoghq.com" }
    
    try {
        $headers = @{
            "DD-API-KEY" = $env:DD_API_KEY
            "DD-APPLICATION-KEY" = $env:DD_APP_KEY
        }
        $response = Invoke-RestMethod -Uri "https://api.$ddSite/api/v1/validate" -Headers $headers
        
        if ($response.valid -eq $true) {
            Write-Host "  ✓ Datadog API credentials are valid" -ForegroundColor Green
        } else {
            Write-Host "  ✗ Datadog API credentials are invalid" -ForegroundColor Red
        }
    } catch {
        Write-Host "  ✗ Failed to validate credentials: $_" -ForegroundColor Red
    }
} else {
    Write-Host "  ! Skipping validation (credentials not set)" -ForegroundColor Yellow
}

# ═══════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Blue
Write-Host "Setup Complete!" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Blue
Write-Host ""
Write-Host "IMPORTANT: On Windows, you have two options to run the bash scripts:" -ForegroundColor Yellow
Write-Host ""
Write-Host "Option 1: Use WSL (Windows Subsystem for Linux) - Recommended" -ForegroundColor Cyan
Write-Host "  wsl ./test-skills.sh"
Write-Host ""
Write-Host "Option 2: Use Git Bash" -ForegroundColor Cyan
Write-Host "  # Open Git Bash and run:"
Write-Host "  ./test-skills.sh"
Write-Host ""
Write-Host "Option 3: Use the Python scripts directly" -ForegroundColor Cyan
Write-Host "  .venv\Scripts\Activate.ps1"
Write-Host "  python python\query_apm.py --service my-service --duration 1h"
Write-Host ""
Write-Host "For Claude Code:" -ForegroundColor Yellow
Write-Host "  claude" -ForegroundColor Cyan
Write-Host ""

