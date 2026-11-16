#!/usr/bin/env bash
# Inspect cross-built PE files and the source/license companion on Linux.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
version=${1:?version required}
stage=${2:?staging directory required}
for name in afishk.exe ffmpeg.exe; do
  x86_64-w64-mingw32-objdump -f "$stage/$name" | grep -q 'pei-x86-64'
done
x86_64-w64-mingw32-objdump -p "$stage/afishk.exe" | grep -q 'Windows GUI'
# The encoder must not silently acquire a non-system DLL dependency.
imports=$(x86_64-w64-mingw32-objdump -p "$stage/ffmpeg.exe" | sed -n 's/.*DLL Name: //p' | tr '[:upper:]' '[:lower:]')
while IFS= read -r dll; do
  case "$dll" in
    kernel32.dll | msvcrt.dll | ws2_32.dll | secur32.dll | advapi32.dll | user32.dll | bcrypt.dll | ntdll.dll | ole32.dll | shell32.dll | ucrtbase.dll | api-ms-win-crt-*.dll) ;;
    *)
      echo "Unexpected FFmpeg DLL dependency: $dll" >&2
      exit 1
      ;;
  esac
done <<<"$imports"
for license in FFmpeg-LGPL-2.1.txt Opus-BSD.txt MinGW-copyright.txt GCC-runtime-copyright.txt GPL-3.txt NSIS-copyright.txt NOTICE.txt; do [[ -s $stage/licenses/$license ]]; done
[[ -s dist/afishk_${version}_windows_amd64_setup.exe ]]
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
tar -xzf "dist/afishk_${version}_windows_media_sources.tar.gz" -C "$work"
(cd "$work/sources" && sha256sum --check SHA256SUMS)
[[ -s $work/sources/scripts/windows-media.sh && -s $work/sources/BUILD.txt ]]
echo 'Windows installer, PE binaries, dependencies and matching source bundle inspected (not executed).'
