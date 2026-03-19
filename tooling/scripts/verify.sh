#!/usr/bin/env sh
set -eu

echo "[check] Running tests..."
uv run pytest -q
go test ./services/api-gateway/...

echo "[check] Verifying required config examples..."
[ -f "tooling/config/recognition.yaml.example" ] || { echo "Missing tooling/config/recognition.yaml.example" >&2; exit 1; }
[ -f "tooling/config/service.yaml.example" ] || { echo "Missing tooling/config/service.yaml.example" >&2; exit 1; }
[ -f "tooling/config/gateway.yaml.example" ] || { echo "Missing tooling/config/gateway.yaml.example" >&2; exit 1; }

echo "[check] OK"
