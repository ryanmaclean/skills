# Datadog Skill - PowerShell Wrapper for Windows
# Usage: powershell -ExecutionPolicy Bypass -File dd.ps1 <command> [args]

$ErrorActionPreference = "Stop"

# Check Python 3
$pythonCmd = $null
if (Get-Command python3 -ErrorAction SilentlyContinue) {
    $pythonCmd = "python3"
} elseif (Get-Command python -ErrorAction SilentlyContinue) {
    # Verify it's Python 3
    $version = & python --version 2>&1
    if ($version -match "Python 3\.") {
        $pythonCmd = "python"
    }
}

if (-not $pythonCmd) {
    Write-Error @"
Python 3 not found.

Install Python 3:
  Download from: https://www.python.org/downloads/
  Make sure to check 'Add Python to PATH' during installation
"@
    exit 1
}

# Get script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$PythonCLI = Join-Path $ScriptDir "python\dd.py"

# Check if Python CLI exists
if (-not (Test-Path $PythonCLI)) {
    Write-Error "Python CLI not found at $PythonCLI"
    exit 1
}

# Check dependencies on first run
$depsChecked = Join-Path $env:USERPROFILE ".dd-skill-test-deps-checked"
if (-not (Test-Path $depsChecked)) {
    try {
        & $pythonCmd -c "import requests" 2>$null
    } catch {
        Write-Warning @"
Python dependencies not installed.

Run: pip3 install -r $ScriptDir\requirements.txt
Or:  pip install -r $ScriptDir\requirements.txt
"@
    }
    New-Item -ItemType File -Path $depsChecked -Force | Out-Null
}

# Execute Python CLI with all arguments
& $pythonCmd $PythonCLI @args
exit $LASTEXITCODE
