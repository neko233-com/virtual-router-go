$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

Write-Host "Running all Go unit tests..." -ForegroundColor Yellow

go test ./...
if ($LASTEXITCODE -ne 0) {
    Write-Error "Tests failed"
    exit 1
}

Write-Host "All tests passed" -ForegroundColor Green
