#!/usr/bin/env bash
# Install a built binary and opt-in desktop integration; never start control.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

BINDIR=${BINDIR:-$HOME/.local/bin}
CONFIGDIR=${CONFIGDIR:-${XDG_CONFIG_HOME:-$HOME/.config}/afishk}
SYSTEMD_USER_DIR=${SYSTEMD_USER_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user}
APPLICATIONSDIR=${APPLICATIONSDIR:-${XDG_DATA_HOME:-$HOME/.local/share}/applications}
BINARY=${BINARY:-bin/afishk}

[[ $(id -u) != 0 ]] || {
  echo 'Run the installer as your logged-in desktop user.' >&2
  exit 1
}
[[ -x $BINARY ]] || {
  echo 'Run make build first (or set BINARY to a release binary).' >&2
  exit 1
}
for path in "$BINDIR" "$CONFIGDIR" "$SYSTEMD_USER_DIR" "$APPLICATIONSDIR"; do
  [[ $path == /* ]] || {
    echo 'Install directories must be absolute.' >&2
    exit 1
  }
  [[ $path =~ ^/[a-zA-Z0-9_./-]*$ ]] || {
    echo 'Install directories support letters, digits, underscore, dot, slash and hyphen.' >&2
    exit 1
  }
done
install -d -m 0700 "$CONFIGDIR"
install -d "$BINDIR" "$SYSTEMD_USER_DIR" "$APPLICATIONSDIR"
install -m 0755 "$BINARY" "$BINDIR/afishk"
sed -e "s|@BINDIR@|$BINDIR|g" -e "s|@CONFIG@|$CONFIGDIR|g" -e "s|@UID@|$(id -u)|g" packaging/afishk.service.in >"$SYSTEMD_USER_DIR/afishk.service"
sed -e "s|@BINDIR@|$BINDIR|g" -e "s|@CONFIG@|$CONFIGDIR|g" packaging/org.afishk.Companion.desktop.in >"$APPLICATIONSDIR/org.afishk.Companion.desktop"
if command -v update-desktop-database >/dev/null; then update-desktop-database "$APPLICATIONSDIR"; fi
if ! systemctl --user daemon-reload; then echo 'No user bus: run systemctl --user daemon-reload in your desktop session.' >&2; fi
printf 'Installed. To create a new configuration: %s/afishk setup --config %s/config.json\n' "$BINDIR" "$CONFIGDIR"
printf '%s\n' 'Existing configuration is preserved. Validate before starting or restarting afishk.'
