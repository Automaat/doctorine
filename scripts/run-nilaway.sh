#!/usr/bin/env bash
set -euo pipefail

# renovate: datasource=go depName=go.uber.org/nilaway
NILAWAY_VERSION="v0.0.0-20260617211854-01ab7e30fbe0"

if [[ $# -lt 1 ]]; then
	echo "Usage: $0 <module-dir> [include-pkgs]" >&2
	exit 2
fi

MODULE_DIR=$1
INCLUDE_PKGS=${2:-}

export GOTOOLCHAIN=local

go install "go.uber.org/nilaway/cmd/nilaway@${NILAWAY_VERSION}"

NILAWAY_BIN="$(go env GOBIN)"
if [[ -z "$NILAWAY_BIN" ]]; then
	GOPATH_FIRST="${GOPATH-}"
	GOPATH_FIRST="${GOPATH_FIRST%%:*}"
	if [[ -z "$GOPATH_FIRST" ]]; then
		GOPATH_FIRST="$(go env GOPATH | cut -d: -f1)"
	fi
	NILAWAY_BIN="${GOPATH_FIRST}/bin"
fi
NILAWAY_BIN="${NILAWAY_BIN}/nilaway"

if [[ ! -x "$NILAWAY_BIN" ]]; then
	NILAWAY_BIN="$(command -v nilaway || true)"
	if [[ -z "$NILAWAY_BIN" ]]; then
		echo "Could not locate the nilaway binary after install" >&2
		exit 1
	fi
fi

cd "$MODULE_DIR" || exit 1

if [[ -n "$INCLUDE_PKGS" ]]; then
	"$NILAWAY_BIN" -include-pkgs="$INCLUDE_PKGS" ./...
else
	"$NILAWAY_BIN" ./...
fi
