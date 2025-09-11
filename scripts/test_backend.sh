#!/usr/bin/env bash
set -euo pipefail
cd backend/src
echo "Running backend unit tests..."

# Avoid slow remote toolchain downloads inside Tilt when host Go is older.
export GOTOOLCHAIN=local

# Verify Go toolchain satisfies module's go directive (1.24)
GO_VER_RAW=$(go version | awk '{print $3}')
GO_MAJ_MIN=$(echo "$GO_VER_RAW" | sed -E 's|go([0-9]+)\.([0-9]+).*|\1.\2|')
need_major=1; need_minor=24
cur_major=${GO_MAJ_MIN%%.*}
cur_minor=${GO_MAJ_MIN##*.}
if [ "${cur_major}" -lt "${need_major}" ] || { [ "${cur_major}" -eq "${need_major}" ] && [ "${cur_minor}" -lt "${need_minor}" ]; }; then
	echo "Error: Go ${need_major}.${need_minor}+ required by go.mod (found ${GO_VER_RAW})." >&2
	echo "Fix: Ensure Tilt runs with Go ${need_major}.${need_minor}+ in PATH or allow remote toolchain (unset GOTOOLCHAIN)." >&2
	exit 1
fi

# Ensure modules are tidy to avoid failures when the go directive or deps changed
go mod tidy
go test -count=1 -race -timeout=60s ./...
