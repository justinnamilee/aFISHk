#!/usr/bin/env bash
# Compatibility entry point for source checkouts.
set -euo pipefail
exec "$(dirname -- "${BASH_SOURCE[0]}")/linux/install.sh" "$@"
