#!/usr/bin/env bash
# Root daemon with one user-owned 0600 socket, rather than /dev/uinput access
# or membership in the input group for the web service.
set -euo pipefail
[[ $(id -u) != 0 ]] || { echo 'Run as the desktop user.' >&2; exit 1; }
uid=$(id -u)
gid=$(id -g)
[[ -x /usr/bin/ydotoold ]] || { echo 'Install ydotool first.' >&2; exit 1; }
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
