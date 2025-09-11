#!/usr/bin/env bash
set -euo pipefail
cd frontend
if [ ! -d node_modules ]; then
  echo "Installing frontend dependencies..."
  npm install --no-audit --no-fund
fi
echo "Running frontend unit/integration tests..."
npm test -- --runInBand
