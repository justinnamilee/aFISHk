#!/usr/bin/env bash
# Nobara/Fedora + Plasma preparation. Keep privileged input separate from HTTP.
set -euo pipefail
usage() {
  cat <<'HELP'
Usage: scripts/setup-plasma.sh [--existing-ydotool]
Installs runtime packages with DNF, then enables a dedicated ydotool daemon.
Run as your normal desktop user; sudo is used for system changes only.
--existing-ydotool  Preserve your existing input daemon; set YDOTOOL_SOCKET
                   in the afishk user service if its socket differs.
This helper does not configure DNS, firewall rules, or start afishk.
HELP
}
existing=false
for arg in "$@"; do
  case "$arg" in
    --existing-ydotool) existing=true ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 1 ;;
  esac
done
[[ $(id -u) != 0 ]] || { echo 'Run as the desktop user.' >&2; exit 1; }
command -v dnf >/dev/null || { echo 'Supported backend: Nobara/Fedora (DNF) + KDE Plasma. Other backends need manual preparation.' >&2; exit 1; }
command -v sudo >/dev/null || { echo 'sudo is required for system preparation.' >&2; exit 1; }
[[ ${XDG_CURRENT_DESKTOP:-} == *KDE* && ${XDG_SESSION_TYPE:-} == wayland ]] || {
  echo 'Run this helper from your logged-in KDE Plasma Wayland session.' >&2; exit 1;
}
packages=(make ydotool pipewire pipewire-pulseaudio pulseaudio-utils)
if ! command -v ffmpeg >/dev/null; then packages+=(ffmpeg-free); fi
echo 'Installing runtime packages from your enabled repositories (existing FFmpeg is preserved).'
sudo dnf install -y "${packages[@]}"
if $existing; then
  echo 'Preserved existing input daemon. Next: make build (source only), make install, then afishk setup.'
  exit 0
fi
uid=$(id -u)
gid=$(id -g)
[[ -x /usr/bin/ydotoold ]] || { echo 'Install ydotool first.' >&2; exit 1; }
daemon_help=$(/usr/bin/ydotoold --help 2>&1) || true
for option in --socket-path --socket-own --socket-perm; do
  [[ $daemon_help == *"$option"* ]] || { echo "ydotoold does not support required option $option" >&2; exit 1; }
done
unit=$(mktemp)
trap 'rm -f "$unit"' EXIT
cat > "$unit" <<UNIT
[Unit]
Description=aFISHk ydotool daemon for UID $uid
After=systemd-modules-load.service
[Service]
Type=simple
ExecStartPre=/usr/sbin/modprobe uinput
ExecStart=/usr/bin/ydotoold --socket-path=/run/afishk-input-$uid/socket --socket-own=$uid:$gid --socket-perm=0600
RuntimeDirectory=afishk-input-$uid
RuntimeDirectoryMode=0711
Restart=on-failure
UMask=0077
[Install]
WantedBy=multi-user.target
UNIT
sudo install -m 0644 "$unit" "/etc/systemd/system/afishk-input-$uid.service"
sudo systemctl daemon-reload
sudo systemctl enable --now "afishk-input-$uid.service"
printf 'Set YDOTOOL_SOCKET=/run/afishk-input-%s/socket (user service installer sets it).\n' "$uid"
printf '%s\n' 'Next: make build (source only), make install, then afishk setup.' 'Run scripts/doctor.sh before starting afishk. FFmpeg must support libopus and pulse.'
