# aFISHk

A small, self-hosted **manual fishing companion for WoW: Forever**, built for
Nobara / KDE Plasma 6 / Wayland and a phone browser over your existing VPN.

- Listen to desktop audio through Opus/WebRTC.
- Tap **Fish** to send one F key to your approved, focused WoW window.
- Request a fresh, native-resolution PNG of that window.

**Status: implementation ready for desktop validation.** Automated checks cover
input rejection, HTTP protection, archive installation, PNG handling and a real
FFmpeg → WebRTC loopback stream. The Nobara/KWin session, FishFry and Android
Chrome over your VPN still require the [desktop acceptance checks](docs/validation.md).

No automatic cast/reel, bite detection, timers, movement or game telemetry.
Use Moonlight separately if you need to reposition or recover the character.

## Install

Run these commands **on your Nobara desktop as your normal logged-in user**.
The server never runs as root. Prepare your VPN separately.

### 1. Linux tools

```bash
./scripts/install-deps.sh
./scripts/setup-ydotool.sh
```

The first helper installs `ydotool`, FFmpeg, PipeWire's Pulse compatibility layer,
Pulse tools and OpenSSL through DNF. Existing repositories are used. If FFmpeg is already available, the helper preserves that installation. Otherwise
it requests `ffmpeg-free`; resolve any codec-package conflict using your distro's
package tooling.
FFmpeg must have the `libopus` encoder and `pulse` input.

The second helper creates a root-owned ydotool daemon with a **0600 socket owned
by your UID**, at `/run/afishk-input-UID/socket`. It does not add your account to
`input` or grant broad device permissions. Only run it if you need this dedicated
daemon; an existing ydotoold setup is also usable by setting `YDOTOOL_SOCKET`.
`ydotoold --help` must support `--socket-path`, `--socket-own`, `--socket-perm`.

### 2. Build or extract a release

For source builds, install **Go 1.25+**, Git and Make (`sudo dnf install golang git make`):

```bash
make build
./bin/afishk version
./scripts/install.sh
```

For a release, download the Linux archive matching your CPU and `SHA256SUMS`,
verify it, extract it, enter its directory, then run:

```bash
sha256sum --ignore-missing --check SHA256SUMS  # beside the downloaded archive
# After extraction, inside afishk_vX.Y.Z_linux_amd64/:
./scripts/install.sh ./bin/afishk
```

The installer creates `~/.local/bin/afishk`, a user service and a KDE desktop
entry for screenshot authorization. It preserves existing configuration and
creates a random token in `~/.config/afishk/token`. It **does not start** control.
Release packages include all runtime install scripts; Go is only needed to build.

### 3. FishFry addon

Download a **Forever-compatible** FishFry ZIP from its
[publisher's page](https://www.curseforge.com/wow/addons/fishfry-forever).
The addon is neither bundled nor silently downloaded from a guessed URL.

```bash
~/.local/bin/afishk install-addon \
  --zip ~/Downloads/FishFry-VERSION.zip \
  --wow-dir '/path/to/WoW/client'
```

`--wow-dir` is the client directory containing `Interface/AddOns`, including a
Wine/Proton installation if applicable. The ZIP must have `FishFry/` at its root
and a `.toc` manifest inside that folder. Unsafe paths, symlinks, special files,
large archives and existing installations are rejected. To update, close WoW
and move the old `FishFry` folder aside first; backups are your responsibility.

Enable FishFry in WoW, configure its manual cast/reel binding to **F**, position
your character and test the binding locally. No custom Lua addon is needed.

### 4. Configure your desktop and VPN

```bash
~/.local/bin/afishk inspect
```

You have five seconds to focus WoW. Copy the reported exact `class` and `title`
into `~/.config/afishk/config.json` as `game_class` and `game_title`.
Keep this configuration private. The example uses a placeholder class so input
cannot work until you choose the actual target. Title changes cause rejection.

For phone access, edit:

```json
{
  "listen": "100.64.0.10:8787",
  "public_origin": "http://100.64.0.10:8787",
  "token_file": "/home/YOUR_USER/.config/afishk/token",
  "ice_address": "100.64.0.10"
}
```

These are fields to change in the **complete installed config**, not a replacement
file. Substitute your actual desktop VPN IP and home path. Restrict TCP **8787**
and UDP **49160–49180** to your trusted VPN clients in your existing firewall.
No public STUN/TURN service is used. Browser peers must be able to reach the
advertised VPN IP directly over UDP. Only IPv4 media transport is supported.

The default listener is `127.0.0.1:8787`. Non-loopback listening requires a token
or explicit `allow_unauthenticated_lan: true`; use the token. Open the exact
configured origin, including port. Basic auth accepts any username and the token
as password; all UI, API, image and health routes are protected consistently.

The default `pulse_source` is `@DEFAULT_MONITOR@`. Use `pactl list short sources`
to select an explicit `SINK.monitor` if needed. This streams **all desktop sound**.
Restart the audio session after changing output devices.

### 5. Validate and start

```bash
./scripts/doctor.sh
~/.local/bin/afishk --config ~/.config/afishk/config.json --check
systemctl --user import-environment WAYLAND_DISPLAY DISPLAY XDG_CURRENT_DESKTOP
systemctl --user start afishk
journalctl --user -u afishk -f
```

Run the [acceptance checks](docs/validation.md) before leaving the PC. Once these
pass, optionally start it with your graphical session:

```bash
systemctl --user enable afishk
```

Keep the desktop unlocked, WoW focused and the phone page in the foreground.
Tap **Listen to desktop**, use the audio player's Play control if Chrome asks,
then tap **Fish** manually when you hear a bite. The UI reports only whether it
sent F; it does not know whether a cast or catch succeeded. A separate screenshot
button saves the latest successful PNG; open it full size for mobile zoom.

## HTTPS and reverse proxies

For built-in HTTPS, set `tls_cert` and `tls_key` to readable PEM files and set
`public_origin` to your `https://HOST:PORT`. Certificate issuance and renewal are
external. For a PFX/P12 bundle, convert outside the service:

```bash
openssl pkcs12 -in certificate.pfx -clcerts -nokeys -out certificate.pem
openssl pkcs12 -in certificate.pfx -nocerts -noenc -out private-key.pem
chmod 600 private-key.pem
```

Supply the full chain if your issuer requires it. Restart after replacement.
A reverse proxy may terminate TLS and forward HTTP to `127.0.0.1:8787`; set
`public_origin` to the public HTTPS origin and preserve its **Host**, **Origin**,
**Authorization** and **X-CSRF-Token** headers. Forwarded headers are not trusted.
The UDP WebRTC ports still require a direct VPN path; HTTP proxying alone does
not carry the audio stream. Plain HTTP is intended only inside your trusted VPN.

## Development

```bash
make tools             # pinned golangci-lint; put $(go env GOPATH)/bin in PATH
make fmt
make check             # formatting, go vet, race-enabled tests, shell syntax
make lint              # errcheck, govet, ineffassign, staticcheck, unused
shellcheck scripts/*.sh
make package VERSION=v0.1.0
```

No frontend build chain, generated framework files, Python tooling or container
is needed. HTML/CSS/JS are embedded in the binary. Tests use fake desktop backends;
the audio integration test uses FFmpeg with a synthetic tone, skipping only if
FFmpeg is absent. CI explicitly installs it.

| Location | Purpose |
| --- | --- |
| `cmd/afishk` | CLI, listener, signals and shutdown |
| `internal/config` | Strict JSON configuration and validation |
| `internal/desktop` | KWin focus queries, single-key input, targeted PNG capture |
| `internal/audio` | One WebRTC peer, FFmpeg process and expiring audio lease |
| `internal/server/web` | Mobile UI, no npm dependencies |
| `internal/addon` | Bounded, staged FishFry ZIP installation |
| `scripts`, `packaging` | Install helpers, service templates and release archives |
| `docs` | Architecture, limits, acceptance and upstream references |

## GitHub releases

Create an empty GitHub repository and push this source to it. For example:

```bash
git init -b main  # if this directory is not already initialized
git add .
git commit -m "Initial aFISHk implementation"
git remote add origin git@github.com:YOUR_ACCOUNT/afishk.git
git push -u origin main
git tag v0.1.0
git push origin v0.1.0
```

CI checks branches and PRs. A `v*` tag starts the release workflow: checks,
Linux **amd64 + arm64** builds, tarballs and SHA-256 checksums, then attachment
to a **draft release**. Publishing is a separate manual GitHub action. You can
also dispatch it for an existing semver tag; reruns update draft assets and
refuse to touch a published release. Actions are pinned to commit SHAs, with
Dependabot configured for actions and Go modules.

See [architecture and limitations](docs/architecture.md),
[validation](docs/validation.md) and [upstream references](docs/upstream.md).
Source is MIT licensed; external tools and FishFry retain their own licenses.
