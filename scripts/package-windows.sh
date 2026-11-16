#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
[[ $(uname -s) == Linux ]] || {
  echo 'Build and package Windows releases on Linux.' >&2
  exit 1
}
version=${1:?Usage: scripts/package-windows.sh v1.0.0}
[[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || {
  echo 'Expected a semver tag.' >&2
  exit 1
}
command -v makensis >/dev/null || {
  echo 'Install NSIS on the Linux build host.' >&2
  exit 1
}
./scripts/windows-media.sh
media=${WINDOWS_MEDIA_DIR:-$PWD/.tools/windows-media}
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
mkdir -p dist
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-H windowsgui -s -w -X main.version=$version" -o "$stage/afishk.exe" ./cmd/afishk
cp "$media/ffmpeg.exe" "$stage/"
cp -R "$media/licenses" "$stage/"
nsis_notice=${NSIS_COPYRIGHT:-/usr/share/doc/nsis/copyright}
[[ -r $nsis_notice ]] || {
  echo 'Install NSIS notices (or set NSIS_COPYRIGHT).' >&2
  exit 1
}
cp "$nsis_notice" "$stage/licenses/NSIS-copyright.txt"
sed -i "s/afishk_VERSION/afishk_$version/g" "$stage/licenses/NOTICE.txt"
cp LICENSE "$stage/"
sed "s/@VERSION@/$version/g" packaging/README-windows.txt >"$stage/README.txt"
output="$PWD/dist/afishk_${version}_windows_amd64_setup.exe"
makensis -V2 -DVERSION="$version" -DSTAGE="$stage" -DOUTPUT="$output" inc/windows/installer.nsi
cp "$media/sources.tar.gz" "dist/afishk_${version}_windows_media_sources.tar.gz"
./tests/windows-package.sh "$version" "$stage"
