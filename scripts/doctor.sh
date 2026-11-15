#!/usr/bin/env bash
set -uo pipefail
fail=0
check() { if "$@"; then printf 'OK: %s\n' "$*"; else printf 'FAIL: %s\n' "$*" >&2; fail=1; fi; }
[[ ${XDG_SESSION_TYPE:-} == wayland ]] || { echo 'FAIL: expected logged-in Wayland session'; fail=1; }
[[ ${XDG_CURRENT_DESKTOP:-} == *KDE* ]] || { echo 'FAIL: expected KDE Plasma'; fail=1; }
for tool in ffmpeg pactl ydotool ydotoold openssl; do check command -v "$tool"; done
if command -v ffmpeg >/dev/null; then
  encoders=$(ffmpeg -hide_banner -encoders 2>/dev/null)
  [[ $encoders == *libopus* ]] || { echo 'FAIL: FFmpeg needs libopus encoder'; fail=1; }
fi
check pactl info
check pactl get-default-sink
socket=${YDOTOOL_SOCKET:-/run/afishk-input-$(id -u)/socket}
[[ -S $socket && -w $socket ]] || { echo "FAIL: missing/wrong permissions for $socket"; fail=1; }
check systemctl --user is-active pipewire.service
printf '%s\n' 'This check sends no keys. Use README validation for KWin capture, target rejection and real VPN audio.'
exit "$fail"
