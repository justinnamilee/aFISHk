#!/usr/bin/env bash
# Build only the Windows audio encoder on Linux, from verified upstream sources.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
[[ $(uname -s) == Linux ]] || {
  echo 'Build Windows releases on Linux.' >&2
  exit 1
}
for tool in curl sha256sum tar make pkg-config x86_64-w64-mingw32-gcc x86_64-w64-mingw32-ar; do
  command -v "$tool" >/dev/null || {
    echo "Missing Linux build tool: $tool" >&2
    exit 1
  }
done
cache=${WINDOWS_MEDIA_DIR:-$PWD/.tools/windows-media}
mkdir -p "$cache"
cache=$(cd "$cache" && pwd)
stamp=$(sha256sum scripts/windows-media.sh | cut -d ' ' -f 1)
if [[ -f $cache/build.stamp && $(cat "$cache/build.stamp") == "$stamp" && -s $cache/ffmpeg.exe && -s $cache/sources.tar.gz ]]; then exit 0; fi
ffmpeg_version=9.0.2
ffmpeg_sha=8c3850283eb25fa026482078a04051e0be17347b09ef81a0849bec15a96e002e
opus_version=1.5.2
opus_sha=65c1d2f78b9f2fb20082c38cbe47c951ad5839345876e46941612ee87f9a7ce1
fetch() {
  local url=$1 name=$2 checksum=$3
  if [[ ! -f $cache/$name ]] || ! printf '%s  %s\n' "$checksum" "$cache/$name" | sha256sum --check --status; then
    curl --fail --silent --show-error --location --retry 3 "$url" -o "$cache/$name"
  fi
  printf '%s  %s\n' "$checksum" "$cache/$name" | sha256sum --check --status
}
fetch "https://ffmpeg.org/releases/ffmpeg-$ffmpeg_version.tar.xz" "ffmpeg-$ffmpeg_version.tar.xz" "$ffmpeg_sha"
fetch "https://downloads.xiph.org/releases/opus/opus-$opus_version.tar.gz" "opus-$opus_version.tar.gz" "$opus_sha"
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
tar -xf "$cache/opus-$opus_version.tar.gz" -C "$stage"
tar -xf "$cache/ffmpeg-$ffmpeg_version.tar.xz" -C "$stage"
jobs=${JOBS:-2}
(
  cd "$stage/opus-$opus_version"
  ./configure --host=x86_64-w64-mingw32 --prefix="$stage/prefix" --disable-shared --enable-static --disable-extra-programs --disable-doc
  make -j "$jobs"
  make install
)
(
  cd "$stage/ffmpeg-$ffmpeg_version"
  export PKG_CONFIG_LIBDIR="$stage/prefix/lib/pkgconfig"
  ./configure --target-os=mingw32 --arch=x86_64 --enable-cross-compile --cross-prefix=x86_64-w64-mingw32- \
    --pkg-config=pkg-config --pkg-config-flags=--static --disable-autodetect --disable-everything \
    --disable-doc --disable-debug --disable-shared --enable-static --disable-x86asm \
    --disable-gpl --disable-nonfree --disable-pthreads --enable-w32threads \
    --enable-ffmpeg --enable-avcodec --enable-avformat --enable-avfilter --enable-swresample \
    --enable-network --enable-libopus --enable-encoder=libopus \
    --enable-decoder=pcm_s16le,pcm_s24le,pcm_s32le,pcm_f32le,pcm_f64le \
    --enable-demuxer=pcm_s16le,pcm_s24le,pcm_s32le,pcm_f32le,pcm_f64le \
    --enable-muxer=rtp --enable-protocol=pipe,udp,rtp --enable-filter=aresample,aformat,anull \
    --extra-cflags="-I$stage/prefix/include" --extra-ldflags="-static -L$stage/prefix/lib"
  make -j "$jobs" ffmpeg.exe
)
mkdir -p "$cache/licenses" "$stage/sources/scripts"
install -m 0755 "$stage/ffmpeg-$ffmpeg_version/ffmpeg.exe" "$cache/ffmpeg.exe"
cp "$stage/ffmpeg-$ffmpeg_version/COPYING.LGPLv2.1" "$cache/licenses/FFmpeg-LGPL-2.1.txt"
cp "$stage/opus-$opus_version/COPYING" "$cache/licenses/Opus-BSD.txt"
# MinGW runtime notices travel with the executable as well as its source bundle.
mingw_notice=${MINGW_COPYRIGHT:-/usr/share/doc/mingw-w64-common/copyright}
[[ -r $mingw_notice ]] || {
  echo 'Install mingw-w64-common (or set MINGW_COPYRIGHT).' >&2
  exit 1
}
cp "$mingw_notice" "$cache/licenses/MinGW-copyright.txt"
gcc_notice=${GCC_COPYRIGHT:-/usr/share/doc/gcc-mingw-w64-base/copyright}
gpl_notice=${GPL3_LICENSE:-/usr/share/common-licenses/GPL-3}
[[ -r $gcc_notice && -r $gpl_notice ]] || {
  echo 'Install GCC runtime notices (or set GCC_COPYRIGHT and GPL3_LICENSE).' >&2
  exit 1
}
cp "$gcc_notice" "$cache/licenses/GCC-runtime-copyright.txt"
cp "$gpl_notice" "$cache/licenses/GPL-3.txt"
cat >"$cache/licenses/NOTICE.txt" <<NOTICE
This application invokes FFmpeg $ffmpeg_version as a separate process.
FFmpeg: https://ffmpeg.org/ — LGPL version 2.1 or later.
Opus $opus_version: https://opus-codec.org/ — BSD license.
FFmpeg is built without GPL or nonfree components. See the accompanying licenses.
Compiler runtime portions use the GCC Runtime Library Exception; notices included.
The matching, unmodified upstream source archives and exact Linux build script
are provided beside the installer as afishk_VERSION_windows_media_sources.tar.gz.
Rebuild it, then replace ffmpeg.exe in the per-user application directory if desired.
aFISHk does not link to FFmpeg libraries. FFmpeg and Opus remain their authors' work.
NOTICE
cp "$cache/ffmpeg-$ffmpeg_version.tar.xz" "$cache/opus-$opus_version.tar.gz" "$stage/sources/"
cp scripts/windows-media.sh "$stage/sources/scripts/"
cp -R "$cache/licenses" "$stage/sources/"
cat >"$stage/sources/BUILD.txt" <<'BUILD'
Unmodified upstream source archives for the corresponding bundled ffmpeg.exe.
On Linux, install make, pkg-config, curl, xz-utils, gcc-mingw-w64-x86-64,
mingw-w64-common and binutils-mingw-w64-x86-64, then run:
  WINDOWS_MEDIA_DIR="$PWD" ./scripts/windows-media.sh
The build script records the exact configure arguments and verified source hashes.
Only the required PCM decoders/demuxers, libopus encoder, audio filters and RTP/UDP
output are enabled. All FFmpeg functionality is in the replaceable ffmpeg.exe;
no FFmpeg library is linked into aFISHk. No source modifications are applied.
BUILD
printf '%s  ffmpeg-%s.tar.xz\n%s  opus-%s.tar.gz\n' "$ffmpeg_sha" "$ffmpeg_version" "$opus_sha" "$opus_version" >"$stage/sources/SHA256SUMS"
cp "$stage/ffmpeg-$ffmpeg_version/ffbuild/config.log" "$stage/sources/ffmpeg-config.log"
x86_64-w64-mingw32-gcc --version >"$stage/sources/compiler.txt"
tar -czf "$cache/sources.tar.gz" -C "$stage" sources
printf '%s\n' "$stamp" >"$cache/build.stamp"
