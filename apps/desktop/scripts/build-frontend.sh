#!/usr/bin/env bash
# Build the web app and stage it under frontend/dist for go:embed.
set -euo pipefail
cd "$(dirname "$0")/../../.." # repo root
npm run build -w @cal/web
rm -rf apps/desktop/frontend/dist
mkdir -p apps/desktop/frontend
cp -r apps/web/dist apps/desktop/frontend/dist
