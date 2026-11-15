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

<p align="center"><img src=".github/screenshot.png" alt="Project preview" width="700" /></p>


## Install

Run these commands **on your Nobara desktop as your normal logged-in user**.
The server never runs as root. Prepare your VPN separately.

### 1. Linux tools

```bash
./scripts/setup-plasma.sh
```

This helper installs Make, `ydotool`, FFmpeg, PipeWire's Pulse compatibility layer,
and Pulse tools through DNF. Existing repositories are used. If FFmpeg is already available, the helper preserves that installation. Otherwise
it requests `ffmpeg-free`; resolve any codec-package conflict using your distro's
package tooling.
FFmpeg must have the `libopus` encoder and `pulse` input.

It then creates a root-owned ydotool daemon with a **0600 socket owned
by your UID**, at `/run/afishk-input-UID/socket`. It does not add your account to
`input` or grant broad device permissions. Pass `--existing-ydotool` to keep an
existing daemon; set `YDOTOOL_SOCKET` in a user service override if its socket differs.
`ydotoold --help` must support `--socket-path`, `--socket-own`, `--socket-perm`.

### 2. Build or extract a release

For source builds, install **Go 1.25+**, Git and Make (`sudo dnf install golang git make`):

```bash
make build
./bin/afishk version
make install
```

For a release, download the Linux archive matching your CPU and `SHA256SUMS`,
verify it, extract it, enter its directory, then run:

```bash
sha256sum --ignore-missing --check SHA256SUMS  # beside the downloaded archive
# After extraction, inside afishk_vX.Y.Z_linux_amd64/:
make install
```

`make install` installs the built binary, a user service and a KDE desktop entry
for screenshot authorization. It preserves existing configuration and **does not
start** control. Run `make build` first for source checkouts; release archives
include the Makefile and prebuilt binary, so Go is only needed to build.
The target delegates to `scripts/install.sh`, which can also be run directly
with the same environment variables.

Install paths accept environment variables or Make arguments:

```bash
BINDIR="$HOME/.local/bin" CONFIGDIR="$HOME/.config/afishk" make install
# Also supported: SYSTEMD_USER_DIR, APPLICATIONSDIR, BINARY (prebuilt binary).
```

Defaults honor `XDG_CONFIG_HOME` and `XDG_DATA_HOME`. Directory overrides must be
absolute and use letters, digits, underscore, dot, slash or hyphen. Generated
service and desktop entries use the selected binary and config paths. Keep the
same overrides when upgrading. Custom service/application directories must be
discoverable by systemd/KDE. With a custom CONFIGDIR, pass its path to setup and
manual runs using `--config`.

Existing installs can upgrade with `make build && make install`, then
`systemctl --user restart afishk`; there is no need to rerun setup.

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
~/.local/bin/afishk setup
```

`inspect` gives you five seconds to focus WoW. Paste its exact `class` and `title`
into the setup prompts. Title changes cause rejection.

`setup` (also `firstrun`) asks for the listener IP/port, browser DNS name or IP,
TLS mode (`no`, `builtin`, or `proxy`), reachable IPv4 for audio, game identity,
and Pulse source. Built-in TLS requires existing PEM certificate/key files;
proxy mode expects externally managed HTTPS. DNS and certificate issuance remain
external. For example, listening on `10.6.6.34:8787` with browser origin
`http://suzuki:8787` and audio IP `10.6.6.34` allows that exact hostname.

It writes a complete validated config and a cryptographically random 256-bit
token beside it, both mode `0600`. The token itself is never printed; read the
token file when signing in. An existing private token is reused. Setup refuses
to overwrite an existing config; edit it manually or use
`afishk setup --config /absolute/path/new-config.json` for a separate config.
Ctrl-D cancels without creating files. Nothing is started automatically.

The default config for both setup and normal runs is
`${XDG_CONFIG_HOME:-$HOME/.config}/afishk/config.json`. To use a config in your
working directory, explicitly pass `--config ./config.json`.

For reference, a typical VPN configuration contains:

```json
{
  "listen": "100.64.0.10:8787",
  "public_origin": "http://100.64.0.10:8787",
  "token_file": "/home/YOUR_USER/.config/afishk/token",
  "ice_address": "100.64.0.10"
}
```

These are selected fields from the **complete generated config**, not a replacement
file. Setup substitutes your actual addresses and paths. Restrict TCP **8787**
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

Audio requests a 120 ms receive jitter buffer where the browser supports it,
trading a little delay for smoother Wi-Fi/VPN playback. Other browsers keep
their adaptive defaults. Opus permits in-band loss recovery when its encoding
mode supports it, and FFmpeg flushes each 20 ms RTP packet. These settings cannot
eliminate sustained network loss. See the [mobile audio checks](docs/validation.md)
for comparisons and diagnostics if Android playback still has gaps.

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

---

See [architecture and limitations](docs/architecture.md),
[validation](docs/validation.md) and [upstream references](docs/upstream.md).
Source is GNU GPL v3+ licensed; external tools and FishFry retain their own licenses.
