[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "[check] Running tests..."
uv run pytest -q
go test ./services/api-gateway/...

Write-Host "[check] Verifying required config examples..."
if (-not (Test-Path "tooling/config/recognition.yaml.example")) {
    throw "Missing tooling/config/recognition.yaml.example"
}
if (-not (Test-Path "tooling/config/service.yaml.example")) {
    throw "Missing tooling/config/service.yaml.example"
}
if (-not (Test-Path "tooling/config/gateway.yaml.example")) {
    throw "Missing tooling/config/gateway.yaml.example"
}

Write-Host "[check] OK"
