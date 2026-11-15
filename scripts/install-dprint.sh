#!/usr/bin/env bash
# Install a pinned development formatter locally; no root or language runtime.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

version=0.64.3
destination=.tools/dprint
if [[ -x $destination ]] && [[ $("$destination" --version) == "dprint $version" ]]; then
  exit 0
fi

[[ $(uname -s) == Linux ]] || {
  echo 'dprint bootstrap supports Linux x86_64 and aarch64.' >&2
  exit 1
}
case "$(uname -m)" in
  x86_64)
    target=x86_64-unknown-linux-gnu
    checksum=f71cb7e22d6776439eba168d5cb296092d6413fea7148fe4d70684a2702a4fee
    ;;
  aarch64 | arm64)
    target=aarch64-unknown-linux-gnu
    checksum=c820bfc67b0cd6f5f0fa4d00ecdcb056944c86030859f5e50151c88d9164b30d
    ;;
  *)
    echo 'dprint bootstrap supports Linux x86_64 and aarch64.' >&2
    exit 1
    ;;
esac
for tool in curl unzip sha256sum; do
  command -v "$tool" >/dev/null || {
    echo "Install $tool before running make tools." >&2
    exit 1
  }
done

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
curl --fail --silent --show-error --location --retry 3 \
  "https://github.com/dprint/dprint/releases/download/$version/dprint-$target.zip" \
  --output "$stage/dprint.zip"
printf '%s  %s\n' "$checksum" "$stage/dprint.zip" | sha256sum --check --status || {
  echo 'dprint archive checksum mismatch; nothing installed.' >&2
  exit 1
}
unzip -q "$stage/dprint.zip" dprint -d "$stage"
chmod 0755 "$stage/dprint"
[[ $("$stage/dprint" --version) == "dprint $version" ]]
mkdir -p .tools
install -m 0755 "$stage/dprint" "$destination"
printf 'Installed dprint %s in %s\n' "$version" "$destination"
