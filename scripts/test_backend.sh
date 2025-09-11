#!/usr/bin/env bash
set -euo pipefail
cd backend/src
echo "Running backend unit tests..."
go test -count=1 -race -timeout=60s ./...
