#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
version=${1:?Usage: scripts/package.sh v1.0.0}
[[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || {
  echo 'Expected semver tag vX.Y.Z (optional prerelease)' >&2
  exit 1
}
mkdir -p dist
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
for arch in amd64 arm64; do
  name="afishk_${version}_linux_${arch}"
  root="$stage/$name"
  mkdir -p "$root/scripts/linux" "$root/inc/linux"
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="-s -w -X main.version=$version" -o "$root/afishk" ./cmd/afishk
  cp LICENSE config.example.json "$root/"
  # Explicit allowlist: never ship developer tools, Makefiles, tests or source.
  for script in install.sh install-dnf-plasma-wayland.sh setup-dnf-plasma-wayland.sh doctor.sh; do
    install -m 0755 "scripts/linux/$script" "$root/scripts/linux/"
  done
  cp inc/linux/afishk.service.in inc/linux/org.afishk.Companion.desktop.in "$root/inc/linux/"
  sed "s/@VERSION@/$version/g" packaging/README-release.md >"$root/README.md"
  tar -czf "dist/$name.tar.gz" -C "$stage" "$name"
done
(
  cd dist
  sha256sum "afishk_${version}_linux_amd64.tar.gz" "afishk_${version}_linux_arm64.tar.gz" >SHA256SUMS
)
./tests/package.sh "$version"
