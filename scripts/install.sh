#!/usr/bin/env bash
# Install a built binary and an opt-in user unit; never start game control here.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
[[ $(id -u) != 0 ]] || { echo 'Run as your logged-in desktop user.' >&2; exit 1; }
binary=${1:-./bin/afishk}
[[ -x "$binary" ]] || { echo "Build first (make build), or pass a release binary." >&2; exit 1; }
base=${XDG_CONFIG_HOME:-$HOME/.config}/afishk
bin=$HOME/.local/bin
units=${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user
apps=${XDG_DATA_HOME:-$HOME/.local/share}/applications
# Desktop Exec and systemd paths below deliberately support ordinary home paths.
[[ "$HOME" =~ ^/[a-zA-Z0-9_./-]+$ && "$base" =~ ^/[a-zA-Z0-9_./-]+$ ]] || { echo 'Nonstandard home/config paths need manual unit installation (see packaging templates).' >&2; exit 1; }
install -d -m 0700 "$base"
install -d "$bin" "$units" "$apps"
install -m 0755 "$binary" "$bin/afishk"
if [[ ! -e "$base/config.json" ]]; then
  sed "s|/home/YOUR_USER|$HOME|g" config.example.json > "$base/config.json"
  chmod 0600 "$base/config.json"
fi
if [[ ! -e "$base/token" ]]; then
  (umask 077; openssl rand -hex 32 > "$base/token")
fi
sed -e "s|@HOME@|$HOME|g" -e "s|@CONFIG@|$base|g" -e "s|@UID@|$(id -u)|g" packaging/afishk.service.in > "$units/afishk.service"
sed "s|@HOME@|$HOME|g" packaging/org.afishk.Companion.desktop.in > "$apps/org.afishk.Companion.desktop"
if command -v update-desktop-database >/dev/null; then update-desktop-database "$apps"; fi
systemctl --user daemon-reload
printf 'Installed. Edit %s/config.json, set token_file to %s/token and exact game identity.\n' "$base" "$base"
printf '%s\n' 'Then follow README desktop validation and start: systemctl --user start afishk'
