[CmdletBinding()]
param(
    [string]$Config = "tooling/config/gateway.yaml"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

go run ./services/api-gateway/cmd/gateway --config $Config
