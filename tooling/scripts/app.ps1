[CmdletBinding()]
param(
    [string]$Host = "127.0.0.1",
    [int]$Port = 8000
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Push-Location "services/recognition-api"
try {
    uv run uvicorn main:app --reload --host $Host --port $Port
}
finally {
    Pop-Location
}
