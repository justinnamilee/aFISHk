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
  mkdir -p "$root/bin"
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags="-s -w -X main.version=$version" -o "$root/bin/afishk" ./cmd/afishk
  cp Makefile README.md AGENTS.md LICENSE config.example.json "$root/"
  cp -R scripts packaging "$root/"
  tar -czf "dist/$name.tar.gz" -C "$stage" "$name"
done
(
  cd dist
  sha256sum "afishk_${version}_linux_amd64.tar.gz" "afishk_${version}_linux_arm64.tar.gz" >SHA256SUMS
)
