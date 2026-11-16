#!/usr/bin/env bash
# Exercise the shipped scripts with isolated destinations and mocked OS commands.
# Optional argument: an extracted native-architecture release directory.
set -euo pipefail
repo=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
source_root=${1:-$repo}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/tree/scripts" "$work/tree/inc" "$work/mock"
cp -R "$source_root/scripts/linux" "$work/tree/scripts/"
cp -R "$source_root/inc/linux" "$work/tree/inc/"
if [[ $# == 1 ]]; then
  cp "$source_root/afishk" "$work/tree/afishk"
else
  cat >"$work/tree/afishk" <<'BIN'
#!/usr/bin/env bash
[[ $1 == version ]] || exit 1
echo 'afishk test'
BIN
fi
chmod +x "$work/tree/afishk"
export BINDIR="$work/bin" CONFIGDIR="$work/config" SYSTEMD_USER_DIR="$work/systemd" APPLICATIONSDIR="$work/applications"
export BINARY='' INSTALL_TEST_LOG="$work/commands" INSTALL_TEST_UID=1000
export XDG_CURRENT_DESKTOP=KDE XDG_SESSION_TYPE=wayland
: >"$INSTALL_TEST_LOG"
cat >"$work/mock/id" <<'MOCK'
#!/usr/bin/env bash
case $1 in -u) echo "$INSTALL_TEST_UID" ;; -g) echo 1000 ;; *) exit 1 ;; esac
MOCK
cat >"$work/mock/uname" <<'MOCK'
#!/usr/bin/env bash
echo Linux
MOCK
for command in sudo systemctl update-desktop-database dnf ffmpeg; do
  cat >"$work/mock/$command" <<'MOCK'
#!/usr/bin/env bash
printf '%s %s\n' "${0##*/}" "$*" >>"$INSTALL_TEST_LOG"
MOCK
done
for command in make go; do
  cat >"$work/mock/$command" <<'MOCK'
#!/usr/bin/env bash
echo 'Release installation invoked a build tool' >&2
exit 99
MOCK
done
chmod +x "$work/mock/"*
export PATH="$work/mock:$PATH"
installer="$work/tree/scripts/linux/install-dnf-plasma-wayland.sh"
# Help and invalid input must have no side effects.
"$installer" --help >/dev/null
if "$installer" --unknown >/dev/null 2>&1; then exit 1; fi
[[ ! -e $BINDIR && ! -s $INSTALL_TEST_LOG ]]
# Preflight root rejection and bad binary rejection before any sudo invocation.
if INSTALL_TEST_UID=0 "$installer" >/dev/null 2>&1; then exit 1; fi
if BINARY=/does/not/exist "$installer" >/dev/null 2>&1; then exit 1; fi
[[ ! -e $BINDIR && ! -s $INSTALL_TEST_LOG ]]
# Full flow, from outside the extracted directory, preserves an existing daemon.
(cd "$work" && "$installer" --existing-ydotool >/dev/null)
grep -q '^sudo dnf install -y ydotool ' "$INSTALL_TEST_LOG"
if grep -Eq '(^| )(make|golang)( |$)|systemctl .* (start|restart|enable)' "$INSTALL_TEST_LOG"; then exit 1; fi
cmp "$work/tree/afishk" "$BINDIR/afishk"
[[ $(stat -c %a "$BINDIR/afishk") == 755 ]]
[[ $(stat -c %a "$CONFIGDIR") == 700 ]]
[[ $(stat -c %a "$SYSTEMD_USER_DIR/afishk.service") == 644 ]]
grep -Fq "ExecStart=$BINDIR/afishk --config $CONFIGDIR/config.json" "$SYSTEMD_USER_DIR/afishk.service"
grep -Fq 'YDOTOOL_SOCKET=/run/afishk-input-1000/socket' "$SYSTEMD_USER_DIR/afishk.service"
grep -Fq 'X-KDE-DBUS-Restricted-Interfaces=org.kde.KWin.ScreenShot2' "$APPLICATIONSDIR/org.afishk.Companion.desktop"
printf 'keep config\n' >"$CONFIGDIR/config.json"
printf 'keep token\n' >"$CONFIGDIR/token"
chmod 600 "$CONFIGDIR/config.json" "$CONFIGDIR/token"
: >"$INSTALL_TEST_LOG"
# --skip-setup must not invoke sudo/dnf, and upgrades must replace the inode.
old_inode=$(stat -c %i "$BINDIR/afishk")
"$installer" --skip-setup >/dev/null
[[ $(stat -c %i "$BINDIR/afishk") != "$old_inode" ]]
[[ $(cat "$CONFIGDIR/config.json") == 'keep config' && $(cat "$CONFIGDIR/token") == 'keep token' ]]
[[ $(stat -c %a "$CONFIGDIR/token") == 600 ]]
if grep -Eq 'sudo|dnf|systemctl .* (start|restart|enable)' "$INSTALL_TEST_LOG"; then exit 1; fi
# Direct/source install accepts BINARY overrides and retains the same defaults.
BINARY="$work/tree/afishk" "$work/tree/scripts/linux/install.sh" >/dev/null
printf 'Installer tests passed (%s).\n' "$source_root"
