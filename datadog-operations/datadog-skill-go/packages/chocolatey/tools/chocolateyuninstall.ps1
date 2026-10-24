$ErrorActionPreference = 'Stop'

$packageName = 'datadog-cli'
$installDir = Join-Path $env:ProgramFiles $packageName

# Remove binary
if (Test-Path $installDir) {
    Remove-Item $installDir -Recurse -Force
    Write-Host "Removed installation directory: $installDir" -ForegroundColor Green
}

# Remove from PATH
# Note: Chocolatey handles this automatically via Install-ChocolateyPath

Write-Host ""
Write-Host "Datadog CLI has been uninstalled successfully." -ForegroundColor Green
Write-Host ""
