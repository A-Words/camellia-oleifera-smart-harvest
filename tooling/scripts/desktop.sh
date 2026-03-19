#!/usr/bin/env sh
set -eu

cd clients/operator-console
bun run tauri:dev
