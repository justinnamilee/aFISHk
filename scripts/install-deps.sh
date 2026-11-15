#!/usr/bin/env bash
# Nobara/Fedora desktop packages only. Does not enable repos or change devices.
set -euo pipefail
command -v dnf >/dev/null || { echo 'This helper targets Nobara/Fedora (dnf required).' >&2; exit 1; }
if [[ $(id -u) == 0 ]]; then
  echo 'Run this as your desktop user; sudo is used only for package installation.' >&2; exit 1
fi
packages=(ydotool pipewire pipewire-pulseaudio pulseaudio-utils openssl)
if ! command -v ffmpeg >/dev/null; then packages+=(ffmpeg-free); fi
sudo dnf install -y "${packages[@]}"
printf '%s\n' 'Packages installed. Run scripts/doctor.sh in your unlocked KDE session.'
