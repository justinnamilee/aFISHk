#!/usr/bin/env bash
# One-command release installation, sharing all logic with source installs.
set -euo pipefail
here=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
setup=true
args=()
for arg in "$@"; do
  case "$arg" in
    --skip-setup) setup=false ;;
    --existing-ydotool) args+=(--existing-ydotool) ;;
    --help | -h)
      cat <<'HELP'
Usage: scripts/linux/install-dnf-plasma-wayland.sh [--skip-setup] [--existing-ydotool]
Run as your desktop user. Prepares DNF + KDE Plasma Wayland and installs aFISHk.
--skip-setup       Install/update files only; do not install packages or touch ydotool.
--existing-ydotool Preserve your own input daemon during system preparation.
BINARY and install-directory overrides are accepted as environment variables.
Configuration is preserved. aFISHk is never started or restarted automatically.
HELP
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg; use --help." >&2
      exit 1
      ;;
  esac
done
"$here/install.sh" --check
if $setup; then "$here/setup-dnf-plasma-wayland.sh" "${args[@]}"; fi
exec "$here/install.sh"
