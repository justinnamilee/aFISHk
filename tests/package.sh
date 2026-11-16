#!/usr/bin/env bash
# Exact archive allowlist for both CPUs, plus native release-install smoke test.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
version=${1:?version required}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
for arch in amd64 arm64; do
  name="afishk_${version}_linux_${arch}"
  tar -xzf "dist/$name.tar.gz" -C "$work"
  root="$work/$name"
  # Compare files only; directories are implied by this allowlist.
  tar -tzf "dist/$name.tar.gz" | sed -e '/\/$/d' -e "s|^$name/||" | LC_ALL=C sort >"$work/actual"
  LC_ALL=C sort >"$work/expected" <<'FILES'
LICENSE
README.md
afishk
config.example.json
inc/linux/afishk.service.in
inc/linux/org.afishk.Companion.desktop.in
scripts/linux/doctor.sh
scripts/linux/install-dnf-plasma-wayland.sh
scripts/linux/install.sh
scripts/linux/setup-dnf-plasma-wayland.sh
FILES
  diff -u "$work/expected" "$work/actual"
  [[ -x $root/afishk ]]
  for script in "$root"/scripts/linux/*.sh; do
    [[ -x $script ]]
    bash -n "$script"
  done
  grep -Fq "aFISHk $version" "$root/README.md"
  if [[ $(uname -s) == Linux ]]; then
    case "$(uname -m):$arch" in
      x86_64:amd64 | aarch64:arm64)
        [[ $("$root/afishk" version) == "afishk $version" ]]
        ./tests/install.sh "$root"
        ;;
    esac
  fi
done
(cd dist && sha256sum --check SHA256SUMS)
echo 'Both release archives passed content and checksum checks.'
