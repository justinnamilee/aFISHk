# aFISHk

**Fish in WoW: Forever from your phone, using your own Linux desktop and VPN.**
Hear the game through your browser, tap to cast or reel, and grab a screenshot
when you want to check the scene. Every tap sends one **F** key to the focused
WoW window through FishFry.

This app lets you fish remotely; it doesn’t fish for you. It has no automatic
casting, bite detection, or input timers. Use Moonlight separately when you need
to move your character or manage the desktop.

<p align="center"><img src=".github/screenshot.png" alt="aFISHk mobile fishing controls" width="700" /></p>

## What you need

- A logged-in **Nobara / KDE Plasma 6 / Wayland** desktop running WoW: Forever.
  Other desktop environments are not supported by this backend.
- [FishFry Forever](https://www.curseforge.com/wow/addons/fishfry-forever), with
  manual cast/reel bound to **F** and tested in the game. Install and enable it
  yourself before using aFISHk; addon installation and updates are your responsibility.
- PipeWire with Pulse compatibility, FFmpeg with `pulse` input and `libopus`
  encoding, and ydotool. The setup helper below prepares these.
- A phone browser and an existing VPN route to the desktop. WebRTC audio needs
  direct IPv4 UDP connectivity as well as access to the web page.

Keep WoW focused, the desktop unlocked, and the phone page in the foreground.
Audio includes **all desktop sound**, not just WoW. The app does not configure
your VPN, DNS, firewall, or sleep/lock settings.

## Install

Run commands on the desktop as your normal logged-in user. Only the runtime
preparation helper uses sudo; the aFISHk service runs as you.

### 1. Install from source or a release

**From source:** install Git, Make and Go 1.25 or newer, then:

```bash
sudo dnf install git make golang
git clone https://github.com/justinnamilee/aFISHk.git
cd aFISHk
make build
./scripts/linux/setup-dnf-plasma-wayland.sh
make install
```

**From a release:** download the Linux archive matching your CPU (`amd64` for
most PCs, `arm64` for ARM) and `SHA256SUMS` from
[Releases](https://github.com/justinnamilee/aFISHk/releases). Verify, extract and
run the installer as your desktop user:

```bash
sha256sum --ignore-missing --check SHA256SUMS
tar -xzf afishk_v1.0.0_linux_amd64.tar.gz
cd afishk_v1.0.0_linux_amd64
./scripts/linux/install-dnf-plasma-wayland.sh
```

Replace the version/architecture with your download. Release installation needs
Bash and standard Linux utilities, **not Make, Go, a compiler or formatters**.
The archive contains only the binary, sample configuration, Linux install/setup
and diagnostic scripts, Linux integration templates, a short README and LICENSE.

### 2. What installation prepares

The DNF + Plasma Wayland setup helper installs runtime dependencies from your
enabled repositories and creates a dedicated ydotool service with a socket owned
by your desktop user. Existing FFmpeg is preserved; otherwise it requests
`ffmpeg-free`. Resolve codec-package conflicts through your distro. It does not
configure VPN, DNS, firewall rules or FishFry.

If you manage ydotool yourself, pass `--existing-ydotool` to the setup helper or
release installer. Set `YDOTOOL_SOCKET` in a `systemctl --user edit afishk`
override if needed. The dedicated socket defaults to
`/run/afishk-input-UID/socket`.

The release installer checks the binary and templates, runs system preparation,
then calls `scripts/linux/install.sh`. `make install` calls that same shared
installer without system preparation. It installs `~/.local/bin/afishk`, the
systemd user service and the KDE desktop entry required for screenshot access.
It preserves existing configuration and tokens and **never starts or restarts
aFISHk**. Binary replacement is atomic, including while the old binary is running;
restart explicitly after validation to use the new version.

Use `--skip-setup` on the release installer for updates or an already-prepared
desktop. Repeated installation is supported. The old `scripts/install.sh`,
`scripts/setup-plasma.sh` and `scripts/doctor.sh` source entry points remain as
compatibility wrappers; they are not shipped in releases.

### 3. Configure the connection

```bash
~/.local/bin/afishk inspect
~/.local/bin/afishk setup
```

`inspect` gives you five seconds to focus WoW and prints its window identity.
Copy the exact `class` and `title` into setup. A changed title needs a config
update.

Setup asks for your listener, browser address, TLS mode, audio IPv4, game identity,
and audio source. For a routed LAN behind your VPN, example answers are:

| Prompt | Example |
| --- | --- |
| Listen IP:port | `10.6.6.34:8787` |
| TLS | `no` (inside the trusted VPN) |
| Browser origin | `http://suzuki:8787` |
| Desktop IPv4 for audio | `10.6.6.34` |
| Pulse source | `@DEFAULT_MONITOR@` |

Use your own addresses. The phone must resolve the hostname and reach the audio
IPv4. `public_origin` must match the URL you actually open, including scheme and
port, without a path. The default loopback listener cannot accept remote clients.

Setup writes private `config.json` and `token` files under
`${XDG_CONFIG_HOME:-$HOME/.config}/afishk/`. It refuses to overwrite an existing
config. For an installed system, edit that file directly and restart the service.
Use `--config /absolute/path/config.json` for a custom location.

### 4. Allow web and audio traffic

Allow these **destination ports on the desktop**, restricted to your VPN clients:

| Protocol | Default port(s) | Purpose |
| --- | --- | --- |
| TCP | `8787` | Web page, controls, screenshots, audio negotiation |
| UDP | `49160–49180` | WebRTC audio |

Allow UDP traffic for sound; the browser chooses its own UDP source port from the range.
WebRTC will not be carried over HTTP, and thus requires its own Firewall rules.
For HTTPS, only the TCP port is affected, setup your proxy accordingly, but the
WebRTC ports must be open directly to the host running WoW.

To use a smaller UDP port range, edit these fields in your existing config, then restart:

```json
"udp_min": 49160,
"udp_max": 49164
```

The range is inclusive; equal values select one port, which must be available.
`ice_address` must be the desktop IPv4 reachable by your device. To reiterate,
there is no public STUN/TURN service, and an HTTP reverse proxy does not carry
the audio stream.

### 5. Start and sign in

```bash
./scripts/linux/doctor.sh
~/.local/bin/afishk --check
systemctl --user import-environment WAYLAND_DISPLAY DISPLAY XDG_CURRENT_DESKTOP
systemctl --user start afishk
```

Open your configured browser URL. For the login prompt, use any username and the
contents of the token file as the password:

```bash
cat "${XDG_CONFIG_HOME:-$HOME/.config}/afishk/token"
```

Tap **Listen to desktop**; if autoplay is blocked, tap Play in the audio player.
Tap the fishing button to cast, listen for a bite, then tap to reel. The green
dot means the service is reachable; it does not verify game focus or a catch.
**Fresh screenshot** captures the game window; open the result full size to zoom.
The activity log shows newest entries first; ▼/▲ switches the order.

After checking the setup below, optionally start aFISHk with your graphical session:

```bash
systemctl --user enable --now afishk
```

## Check your setup before using it remotely

These checks need your actual desktop and phone; automated tests cannot establish
KWin permissions, FishFry behavior, or audible latency on your VPN.

- **Input:** with WoW focused, one tap should cause one cast/reel action. Focus a
  different app, or lock the desktop: taps must be rejected without typing F.
- **Capture:** take a fresh game screenshot, then try with another app focused.
  The second request should fail and retain the previous image. Capture must work
  through the installed service without a permission prompt or desktop fallback.
- **Audio:** listen for at least five minutes on Android Chrome / mobile browser
  over your VPN (or home network), including actual splashes. Confirm that delay
  and continuity allow you to react. Try Stop/Listen and test again.
- **Recovery:** background the phone page or drop its network for over 20 seconds.
  Return and restart listening. Restart the desktop service and confirm the page
  reconnects and controls recover.

## Updates and service management

For an existing source install:

```bash
git pull --ff-only
make build
make install
systemctl --user restart afishk
```

For a release update, verify and extract the new archive, then:

```bash
./scripts/linux/install-dnf-plasma-wayland.sh --skip-setup
~/.local/bin/afishk --check
systemctl --user restart afishk
journalctl --user -u afishk -f
```

Do not rerun `afishk setup` for an existing config. Use the same directory
overrides for updates as on first install.

To disable startup, use `systemctl --user disable --now afishk`. If no longer
needed, also disable the dedicated input daemon with
`sudo systemctl disable --now "afishk-input-$(id -u).service"`.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Page unreachable | Listener address, VPN routing, TCP firewall rule, and `systemctl --user status afishk`. |
| Rejected hostname/origin | Open exactly `public_origin`; make sure phone DNS resolves it correctly. |
| Repeated login prompt | Use the token contents as password, not its filename. |
| Input rejected | WoW focus, exact class/title, unlocked Wayland session, and `./scripts/linux/doctor.sh`. |
| Screenshot denied | Run the installed binary with its matching KDE desktop entry; log out/in if permissions are cached. Never disable KWin permission checks globally. |
| Audio connects but is silent | Play desktop sound; inspect `pactl list short sources`, select the correct monitor in `pulse_source`, then restart. |
| Audio never connects | Check `ice_address` and UDP rules; opening TCP alone is insufficient. |
| Audio flutters or drops | Compare LAN vs VPN and speaker vs Bluetooth; keep the page foregrounded. Stop/Listen after changing desktop output devices. |

For a foreground diagnostic run, stop the service first, then:

```bash
export YDOTOOL_SOCKET="/run/afishk-input-$(id -u)/socket"
~/.local/bin/afishk
```

Do not run the app as root to fix session-bus or audio access. Capture supports
SDR 32-bit image layouts; unsupported HDR formats are rejected. Failed captures
preserve the last successful PNG in the private cache directory.

## Advanced configuration

**HTTPS:** setup supports built-in TLS using existing PEM certificate/key files,
or an external HTTPS proxy. Set `tls_cert`, `tls_key`, and `public_origin` for
built-in TLS. Certificate issuance and renewal are external; restart after
replacement. For a proxy, bind HTTP to loopback and preserve Host, Origin,
Authorization and X-CSRF-Token headers. Forwarded headers are not trusted, and
UDP media still needs its direct VPN route. Use plain HTTP only on a trusted
LAN/VPN; **Basic authentication alone does not encrypt HTTP**.

**Install paths:** `make install` accepts `BINDIR`, `CONFIGDIR`,
`SYSTEMD_USER_DIR`, `APPLICATIONSDIR` and `BINARY` as environment variables or
Make arguments. Release installers accept the same environment variables;
`BINARY` defaults to the archive's root binary (or `bin/afishk` in source). Defaults respect XDG config/data directories. Directory overrides
must be absolute and contain only letters, digits, underscore, dot, slash or
hyphen. Custom systemd/KDE directories must be discoverable by those services.
For a custom config directory, pass its full config path to setup and manual runs.

## Development

Install Go 1.25+, Make, a C compiler (for race tests), ShellCheck, curl, unzip,
and FFmpeg. On Nobara/Fedora, the development packages are:

```bash
sudo dnf install golang make gcc ShellCheck curl unzip
export PATH="$(go env GOPATH)/bin:$PATH"
make tools
make fmt
make check
make lint
make build
```

`make tools` installs pinned shfmt and golangci-lint versions plus a standalone
**dprint** binary in `.tools/`. The dprint installer supports Linux x86_64 and
aarch64 with glibc (including Nobara/Fedora and Ubuntu), verifies the archive's
SHA-256 checksum, and reuses the installed version on subsequent runs.
Plugin versions and checksums are pinned in `dprint.json`; setup downloads and
caches them. Initial setup needs network access. No Node.js, npm or Rust
toolchain is required.

Formatters are **gofmt** for Go, **shfmt** for Bash, and **dprint** for HTML/CSS/JS.
`make fmt` writes formatting; `make fmt-check` checks without editing.
`make check` checks formatting, runs `go vet`, race tests, Bash syntax checks and isolated installer tests.
`make lint` runs golangci-lint and ShellCheck. dprint parses frontend files during
formatting checks, but it does not replace browser behavior tests. CI runs the
same checks. The plain frontend is embedded directly in the Go binary with no
bundling step.

Tests use fake desktop backends. The audio integration test runs real FFmpeg
encoding and a WebRTC loopback stream; install FFmpeg so it runs rather than skips.
Run `go test -v ./internal/audio` to see its result. Local tests do not replace
the desktop/phone checks above.

| Location | Purpose |
| --- | --- |
| `cmd/afishk` | CLI and service lifecycle |
| `internal/config`, `internal/setup` | Config validation and guided setup |
| `internal/desktop` | Backend interfaces, guarded actions, Linux KWin and ydotool |
| `internal/audio` | Capture interface, Linux Pulse/FFmpeg and a single WebRTC session |
| `internal/platform` | OS-specific backend selection and lifetime |
| `internal/server` | HTTP protections and embedded mobile UI |
| `scripts/linux`, `inc/linux` | Shipped runtime scripts and integration templates |
| `scripts`, `packaging`, `tests` | Development tooling, release README and installer/archive checks |

`make package VERSION=v1.0.0` builds Linux amd64/arm64 archives and checksums in
`dist/`. Packaging checks both archives against an exact runtime-only file list,
verifies checksums, and smoke-tests installation of the host-architecture binary
using temporary destinations and mocked system commands. CI runs these checks.
The other architecture is built and inspected but requires its own runtime test.
Tag releases trigger CI and create a **draft** release for review; the
workflow refuses to modify published releases.

## License and credits

- Project is GNU GPL v3 or later; see [LICENSE](LICENSE).
- [FishFry](https://www.curseforge.com/wow/addons/fishfry-forever) is installed separately and retains its publisher's license.
- aFISHk uses:
  - [KWin](https://develop.kde.org/docs/plasma/kwin/),
  - [ydotool](https://github.com/ReimuNotMoe/ydotool),
  - [FFmpeg](https://ffmpeg.org/),
  - [PipeWire](https://pipewire.org/), and
  - [Pion WebRTC](https://github.com/pion/webrtc).
  - These projects retain their own licenses.

### Adding a desktop backend

Backends are compiled-in Go implementations selected in `internal/platform`,
not runtime-loaded plugins. Linux KDE Plasma Wayland remains the only supported
platform; existing JSON configuration keys and audio settings are unchanged.

| Interface | Contract |
| --- | --- |
| `desktop.WindowSource` | Return the focused native window only after verifying the session is unlocked; fail if either state is unknown. |
| `desktop.Capturer` | Capture the supplied opaque window ID only; never fall back to an active window or desktop. |
| `desktop.Input` | Send one F press/release, with no activation or key-down retries; permit best-effort key-up on failure. |
| `audio.Capture` / `audio.Process` | Start an Opus RTP producer; context cancellation must terminate it and unblock its single `Wait` call. |

The shared desktop controller owns exact class/title matching, action exclusion,
input throttling and private screenshot persistence. The shared audio manager
owns WebRTC, UDP port bounds, the single-peer limit and lease cleanup. The HTTP
server consumes interfaces for both desktop actions and audio sessions.

Add implementations behind OS-specific files/build constraints, wire them in
`internal/platform`, and add platform defaults/validation in `internal/config`
and prompts in `internal/setup`. KWin, ydotool and Pulse/FFmpeg implementations
are Linux-only. Non-Linux builds explicitly reject desktop startup and setup;
a successful Windows cross-build is an architecture check, **not Windows support**.
Preserve the contracts above and add backend-specific acceptance checks before
supporting another platform.
