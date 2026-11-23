#!/usr/bin/env bash
# Shared source/release installer. Run as the desktop user; never start control.
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
cd -- "$root"
check_only=false
case ${1:-} in
  --check) check_only=true ;;
  --help | -h)
    echo 'Usage: scripts/linux/install.sh [--check]'
    echo 'Install only; prepare the desktop with setup-dnf-plasma-wayland.sh first.'
    echo 'Overrides: BINARY BINDIR CONFIGDIR SYSTEMD_USER_DIR APPLICATIONSDIR.'
    exit 0
    ;;
  '') ;;
  *)
    echo 'Unknown argument; use --help.' >&2
    exit 1
    ;;
esac
[[ $# -le 1 ]] || {
  echo 'Too many arguments.' >&2
  exit 1
}
BINDIR=${BINDIR:-$HOME/.local/bin}
CONFIGDIR=${CONFIGDIR:-${XDG_CONFIG_HOME:-$HOME/.config}/afishk}
SYSTEMD_USER_DIR=${SYSTEMD_USER_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user}
APPLICATIONSDIR=${APPLICATIONSDIR:-${XDG_DATA_HOME:-$HOME/.local/share}/applications}
if [[ -z ${BINARY:-} ]]; then
  BINARY=$root/afishk
  if [[ ! -f $BINARY ]]; then BINARY=$root/bin/afishk; fi
fi
[[ $(uname -s) == Linux ]] || {
  echo 'This installer requires Linux.' >&2
  exit 1
}
[[ $(id -u) != 0 ]] || {
  echo 'Run the installer as your logged-in desktop user.' >&2
  exit 1
}
[[ -x $BINARY ]] || {
  echo 'Missing executable: build from source or extract the complete release archive.' >&2
  exit 1
}
for path in "$BINDIR" "$CONFIGDIR" "$SYSTEMD_USER_DIR" "$APPLICATIONSDIR"; do
  [[ $path =~ ^/[a-zA-Z0-9_./-]+$ ]] || {
    echo 'Install directories must be absolute and use letters, digits, underscore, dot, slash or hyphen.' >&2
    exit 1
  }
done
for template in afishk.service.in org.afishk.Companion.desktop.in; do
  [[ -r $root/inc/linux/$template ]] || {
    echo "Missing inc/linux/$template; extract the complete archive." >&2
    exit 1
  }
done
# Reject a wrong-architecture or damaged binary before setup invokes sudo.
if ! "$BINARY" version; then
  echo 'Cannot run this binary. Choose the release matching your CPU architecture.' >&2
  exit 1
fi
if $check_only; then exit 0; fi
install -d -m 0700 "$CONFIGDIR"
install -d "$BINDIR" "$SYSTEMD_USER_DIR" "$APPLICATIONSDIR"
# Stage in each destination filesystem, then rename: upgrades also work while
# the old executable is running. Existing configuration is never overwritten.
bin_tmp=$(mktemp "$BINDIR/.afishk.XXXXXX")
service_tmp=''
desktop_tmp=''
cleanup() { rm -f -- "$bin_tmp" ${service_tmp:+"$service_tmp"} ${desktop_tmp:+"$desktop_tmp"}; }
trap cleanup EXIT
service_tmp=$(mktemp "$SYSTEMD_USER_DIR/.afishk.XXXXXX")
desktop_tmp=$(mktemp "$APPLICATIONSDIR/.afishk.XXXXXX")
install -m 0755 "$BINARY" "$bin_tmp"
sed -e "s|@BINDIR@|$BINDIR|g" -e "s|@CONFIG@|$CONFIGDIR|g" -e "s|@UID@|$(id -u)|g" "$root/inc/linux/afishk.service.in" >"$service_tmp"
sed -e "s|@BINDIR@|$BINDIR|g" -e "s|@CONFIG@|$CONFIGDIR|g" "$root/inc/linux/org.afishk.Companion.desktop.in" >"$desktop_tmp"
chmod 0644 "$service_tmp" "$desktop_tmp"
mv -f -- "$bin_tmp" "$BINDIR/afishk"
mv -f -- "$service_tmp" "$SYSTEMD_USER_DIR/afishk.service"
mv -f -- "$desktop_tmp" "$APPLICATIONSDIR/org.afishk.Companion.desktop"
if command -v update-desktop-database >/dev/null; then update-desktop-database "$APPLICATIONSDIR"; fi
if ! systemctl --user daemon-reload; then echo 'No user bus: run systemctl --user daemon-reload in your desktop session.' >&2; fi
printf 'Installed. New configuration: %s/afishk setup --web --config %s/config.json\n' "$BINDIR" "$CONFIGDIR"
printf '%s\n' 'Existing configuration is preserved. Validate before explicitly starting or restarting afishk.'
