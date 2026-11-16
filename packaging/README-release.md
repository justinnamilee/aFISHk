# aFISHk @VERSION@

For a logged-in Nobara/Fedora **KDE Plasma 6 Wayland** desktop. Install
[FishFry Forever](https://www.curseforge.com/wow/addons/fishfry-forever) yourself
and bind manual cast/reel to **F**. Keep WoW focused and the desktop unlocked.

Verify the downloaded archive against `SHA256SUMS` before extracting it:
`sha256sum --ignore-missing --check SHA256SUMS`.
No Go, Make, compiler or formatting tools are required.

## First install

From this extracted directory, as your desktop user (not root):

```bash
./scripts/linux/install-dnf-plasma-wayland.sh
~/.local/bin/afishk setup --web
./scripts/linux/doctor.sh
systemctl --user import-environment WAYLAND_DISPLAY DISPLAY XDG_CURRENT_DESKTOP
systemctl --user start afishk
```

The installer uses sudo for runtime packages and a dedicated ydotool daemon.
Use `--existing-ydotool` to preserve your own daemon; set `YDOTOOL_SOCKET` in the
user service if its socket differs. Existing FFmpeg is preserved. Installation
never starts aFISHk and never overwrites configuration or tokens.

The local browser wizard identifies WoW after you focus it and generates your
browser password. Click Finish setup before starting the service. The sample `config.example.json` is a reference with placeholders;
`setup` creates your private config and token in `~/.config/afishk/` (or XDG config).
Open the configured browser origin; use any username and the token file contents
as the password. Plain HTTP Basic authentication is only appropriate on a trusted
LAN/VPN. Allow TCP 8787 and UDP 49160–49180 from trusted clients; adjust for your
config. The installer does not configure VPN, DNS or firewall rules.

## Upgrade

After verifying and extracting the new release:

```bash
./scripts/linux/install-dnf-plasma-wayland.sh --skip-setup
systemctl --user restart afishk
```

Do not rerun `afishk setup` for an existing config. Optional install environment
variables: `BINDIR`, `CONFIGDIR`, `SYSTEMD_USER_DIR`, `APPLICATIONSDIR`, `BINARY`.
Use the same paths on upgrades; default paths respect XDG config/data variables.
For a custom config directory, pass `--config /absolute/path/config.json`
to setup and foreground runs. Install directories must be absolute,
using letters, digits, underscore, dot, slash or hyphen.

[Full instructions and troubleshooting](https://github.com/justinnamilee/aFISHk/blob/@VERSION@/README.md).
[Corresponding source](https://github.com/justinnamilee/aFISHk/tree/@VERSION@).
Licensed under GPL-3.0-or-later; see LICENSE. Dependencies retain their own licenses.
