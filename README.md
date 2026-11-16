# aFISHk

**Fish in WoW: Forever from your phone, using your own Windows or Linux desktop and VPN.**
Hear the game through your browser, tap to cast or reel, and take a screenshot
when you want to check the scene. Each tap sends one **F** to the focused WoW
window through FishFry. There is no automatic casting, bite detection or input
timer. Use Moonlight/Parsec/TeamViewer/Rustdesk/VNC/Remote Desktop separately to
manage your character or the desktop.

<p align="center"><img src=".github/screenshot.png" alt="aFISHk mobile fishing controls" width="700" /></p>

## Requirements

- WoW: Forever and [FishFry Forever](https://www.curseforge.com/wow/addons/fishfry-forever),
  installed by you, with manual cast/reel bound to **F**.
- A signed-in, unlocked desktop with WoW focused.
- Some means to keep the desktop session awake and signed in.
- A phone browser and your own LAN/VPN connection. You manage VPN, DNS, firewall
  rules, the addon and sleep/lock settings.
- Direct IPv4 UDP connectivity for audio, in addition to the web connection.

### Windows

**Windows 11 on an Intel/AMD x64 PC.** Run WoW and aFISHk as the same normal user;
do not use “Run as administrator.” Use WoW in windowed or borderless mode for
screenshots.  Windows ARM builds are not provided.

Audio captures the **Windows default playback device**, including other desktop
sound. After changing that device, stop listening and start listening again.

### Linux

**Nobara/Fedora, KDE Plasma 6 and Wayland**, with PipeWire Pulse compatibility,
FFmpeg supporting `pulse` input and `libopus`, and ydotool. The Linux setup helper
installs runtime dependencies. Other Linux desktop environments are not yet supported.
Audio captures the selected Pulse monitor, including other desktop sound.

## Install on Windows

1. Download `afishk_VERSION_windows_amd64_setup.exe` from
   [Releases](https://github.com/justinnamilee/aFISHk/releases) and run it as your
   normal desktop user. The installer includes FFmpeg and installs only for you.
2. Optionally select **Start aFISHk when I sign in**; it is off by default.
3. Open **aFISHk** from the finish page or Start Menu. First run opens a local
   browser setup page on this computer.
4. Click **Identify WoW**, then focus the game within five seconds. Return to
   setup, choose your desktop LAN/VPN address, and save the settings.
5. Copy the generated browser password, then click **Finish setup**. The companion
   starts. Open the displayed address on your phone and sign in with any username
   and that password.

The installer does not open firewall ports or install FishFry. Windows may warn
about an unsigned installer; **only install downloads you trust from this project's
releases**. `SHA256SUMS` is provided alongside the downloads.

| Item | Location |
| --- | --- |
| Program and bundled FFmpeg | `%LOCALAPPDATA%\Programs\aFISHk` |
| Configuration | `%APPDATA%\afishk\config.json` |
| Browser password | `%APPDATA%\afishk\token` |
| Last screenshot and logs | `%LOCALAPPDATA%\afishk` |

Start Menu includes **aFISHk**, **Stop aFISHk**, **Settings folder**, **Logs folder**
and **Uninstall aFISHk**. Opening aFISHk again opens its web page without starting
a second copy. Use Stop aFISHk before editing configuration, then launch it again.
The password can be retrieved from the `token` file if you lose it.

For updates, run the new installer. It stops the old copy and preserves your
configuration and password. Open aFISHk afterward. Rerunning the installer also
lets you change the start-at-sign-in option. Uninstall removes the program and
startup entry but keeps your configuration, password and cached files.

## Install on Linux

Run commands in your logged-in Plasma session as your normal user. Only system
preparation uses sudo; aFISHk runs as you with no special permissions (except
pressing **F**).

### From a release

Download the Linux archive for your CPU (`amd64` or `arm64`) and `SHA256SUMS` from
[Releases](https://github.com/justinnamilee/aFISHk/releases). Verify and extract it:

```bash
sha256sum --ignore-missing --check SHA256SUMS
tar -xzf afishk_v1.0.0_linux_amd64.tar.gz
cd afishk_v1.0.0_linux_amd64
./scripts/linux/install-dnf-plasma-wayland.sh
```

Replace the version and architecture with your download. Releases need no Go,
Make, compiler or formatter. Existing FFmpeg is preserved; otherwise setup
requests `ffmpeg-free` from your enabled repositories.

If you already manage ydotool, add `--existing-ydotool`. Set `YDOTOOL_SOCKET` in a
`systemctl --user edit afishk` override if needed; the dedicated daemon's default
socket is `/run/afishk-input-<UID>/socket`.

### From source

Install Git, Make and Go 1.25 or newer, then:

```bash
sudo dnf install git make golang
git clone https://github.com/justinnamilee/aFISHk.git
cd aFISHk
make build
./scripts/linux/setup-dnf-plasma-wayland.sh
make install
```

### Configure and start

Both installation methods use the same browser setup:

```bash
~/.local/bin/afishk setup --web
```

Click **Identify WoW**, focus the game within five seconds, return to the browser,
then choose your desktop LAN/VPN address. Save the generated password and click
**Finish setup**. The wizard is local to this computer and never overwrites an
existing configuration. If your browser does not open, use the local URL printed
in the terminal. Terminal setup remains available with `afishk inspect` followed
by `afishk setup`.

Start the installed user service:

```bash
systemctl --user import-environment WAYLAND_DISPLAY DISPLAY XDG_CURRENT_DESKTOP
systemctl --user start afishk
```

Optionally enable startup with your graphical session:

```bash
systemctl --user enable afishk
```

The binary is installed in `~/.local/bin/afishk`; config and password are in
`${XDG_CONFIG_HOME:-$HOME/.config}/afishk/`. The cache follows XDG cache settings.
Installation preserves settings and does not start or restart aFISHk. To change
an existing setup, edit `config.json` and restart the user service.

### Linux updates and removal

For source updates:

```bash
git pull --ff-only
make build
make install
systemctl --user restart afishk
```

For release updates, verify/extract the new archive, then:

```bash
./scripts/linux/install-dnf-plasma-wayland.sh --skip-setup
systemctl --user restart afishk
```

Do not rerun setup for an existing config. To disable startup and stop the app,
use `systemctl --user disable --now afishk`. If no longer needed, disable the
input daemon with `sudo systemctl disable --now "afishk-input-$(id -u).service"`.

## Connection and everyday use

Allow these **destination ports on the desktop**, restricted to your trusted
LAN/VPN clients:

| Protocol | Default ports | Purpose |
| --- | --- | --- |
| TCP | `8787` | Web page, controls, screenshots and audio negotiation |
| UDP | `49160–49180` | WebRTC audio |

The client (phone, etc) must resolve the hostname you chose and reach the desktop's
audio IPv4.  `public_origin` must exactly match the browser URL's scheme, hostname
and port, without a path. `ice_address` is the desktop IPv4 reachable by the client.
An HTTP reverse proxy does not carry the UDP audio stream; there is no public STUN/TURN
relay.

Open your configured browser URL. Sign in with **any username** and your generated
token as password. Tap **Listen to desktop**; if autoplay is blocked, tap Play in the
audio player. Keep the page in the foreground. Tap the fishing button to cast,
listen for a bite, then tap to reel. **Fresh screenshot** captures the game window.
Failed captures preserve the previous successful image. The green dot indicates
that the service is reachable, not that WoW is focused or a fish has been caught.

## Troubleshooting

| Symptom | What to do |
| --- | --- |
| Page unreachable | Check the chosen address, VPN routing and TCP firewall rule. On Windows, open aFISHk and inspect its Logs folder; on Linux, use `systemctl --user status afishk`. |
| Hostname/origin rejected | Open exactly `public_origin`; ensure phone DNS resolves that hostname. |
| Repeated login prompt | Use the contents of `token` as the password, not the filename. |
| Input rejected | Focus the exact configured WoW window and unlock the desktop. On Windows, release F and modifier keys and run both programs without administrator elevation. On Linux, use `./scripts/linux/doctor.sh`. |
| Game moved or title changed | Update the exact game class/title in `config.json`. On Windows, also update `game_executable` if WoW moved to a different path. |
| Windows screenshot blank or unavailable | Use windowed/borderless mode and keep WoW focused. Some game/rendering combinations cannot supply a window screenshot; aFISHk will not capture the whole desktop instead. |
| Linux screenshot denied | Use the installed binary and KDE desktop entry; log out/in if permissions are cached. Do not disable KWin permission checks globally. |
| Audio silent | Ensure sound plays on the default Windows playback device or selected Linux Pulse monitor. After switching devices, stop/start listening. |
| Audio never connects | Check `ice_address` and UDP firewall rules; opening TCP alone is insufficient. |
| Audio drops or flutters | Compare LAN/VPN and speaker/Bluetooth playback; keep the phone page foregrounded. Stop/start listening to reconnect. |
| Setup says config exists | Edit the existing config; setup deliberately does not replace it. |

For Linux logs use `journalctl --user -u afishk -f`. For a foreground run, stop
the user service, set `YDOTOOL_SOCKET=/run/afishk-input-$(id -u)/socket`, and run
`~/.local/bin/afishk`. Never run as root to fix desktop permissions.

## Advanced configuration

The browser wizard supports plain HTTP on a trusted LAN/VPN, built-in HTTPS with
your own PEM certificate/key files, or an external HTTPS proxy. **Basic
authentication alone does not encrypt HTTP.** For a proxy, bind the app to
loopback/lan and preserve Host, Origin, Authorization and X-CSRF-Token headers.
Forwarded headers are not trusted. UDP media still needs its **direct** route.
Certificate issuance and renewal are your responsibility; restart after replacing
certificate files.

To change the desktop audio port range, edit `udp_min` and `udp_max`, then restart.
The range is inclusive; equal values select a single available port.

Linux install paths accept `BINDIR`, `CONFIGDIR`, `SYSTEMD_USER_DIR`,
`APPLICATIONSDIR` and `BINARY` as environment variables or Make arguments. Directory
overrides must be absolute and contain only letters, digits, underscore, dot,
slash or hyphen. Custom systemd/KDE directories must be discoverable by those
services. Use the same overrides for upgrades, and pass a custom config path via
`--config` to setup or manual runs.

## License and credits

- aFISHk is GPL-3.0-or-later; see [LICENSE](LICENSE).
- [FishFry](https://www.curseforge.com/wow/addons/fishfry-forever) is installed separately.
- [FFmpeg](https://ffmpeg.org/) is bundled in the Windows installer under
  LGPL-2.1-or-later, with [Opus](https://opus-codec.org/) under its BSD license.
  The matching source and build script are in the Windows media source archive
  beside each installer on [Releases](https://github.com/justinnamilee/aFISHk/releases).
  Included license notices describe the components; FFmpeg remains replaceable.
- Linux integration uses [KWin](https://develop.kde.org/docs/plasma/kwin/),
  [ydotool](https://github.com/ReimuNotMoe/ydotool) and [PipeWire](https://pipewire.org/).
- Audio transport uses [Pion WebRTC](https://github.com/pion/webrtc).
- Dependencies retain their own licenses and copyrights.
