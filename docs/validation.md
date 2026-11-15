# Desktop acceptance checks

Do these on the target Nobara/KDE/Wayland machine before unattended use. A build
or synthetic audio test is not evidence that the desktop acceptance gates pass.
Record the date, versions, VPN and phone details, and outcome in a local ticket.
No target-desktop acceptance results are claimed by this repository.

| Gate | Procedure | Pass condition |
| --- | --- | --- |
| G1 identity | Run `afishk inspect`, focus WoW within five seconds; configure exact class/title | Active game identity is reported; another application differs |
| G1 input | With FishFry F binding tested locally, tap Fish once | Exactly one short F action; no follow-up or repeated input |
| G1 rejection | Focus a terminal, tap Fish; stop KWin script availability if practical; lock the desktop | Visible rejection and no injected F for wrong/unknown/locked target |
| G1 race | Switch focus immediately around a tap in a controlled local test | Document the non-atomic limitation; never treat this backend as an atomic target guarantee |
| G2 capture | Repeatedly request capture with WoW focused, with other windows nearby | Only game UUID contents, fresh PNG each time, no prompts, no game changes |
| G2 failure | Capture once, focus another app, capture again | Visible error; previous valid PNG is unchanged and remains private |
| G2 permission | Start using installed binary and desktop entry | Capture needs no request-time interaction; no global permission bypass |
| G3 audio | Play desktop sound then real fishing splashes; listen in Android Chrome over VPN | Both local and phone playback work; practical manual bite response |
| G3 recovery | Stop/start audio, change default output device, reconnect | No stale FFmpeg process/peer; new session captures chosen output |
| G4 transport | Test configured HTTP/PEM HTTPS, auth on every endpoint and any proxy | Controls and PNG work, direct UDP audio works, wrong token rejected |
| G5 loss | Drop phone network or close/background page; wait at least 20 seconds; return | Controls show disconnected; audio stops; Listen can establish a new session |

Useful diagnostics:

```bash
./scripts/doctor.sh
journalctl --user -u afishk --since '10 minutes ago'
systemctl status "afishk-input-$(id -u)"
pactl list short sources
ffmpeg -hide_banner -encoders 2>/dev/null | grep libopus
ffmpeg -hide_banner -devices 2>/dev/null | grep pulse
```

For direct foreground runs with the dedicated daemon:

```bash
export YDOTOOL_SOCKET="/run/afishk-input-$(id -u)/socket"
~/.local/bin/afishk --config ~/.config/afishk/config.json
```

Stop the user service first to avoid a listener conflict. Do not use root to fix
session-bus or audio access; these are logged-in-user resources.

For teardown:

```bash
systemctl --user disable --now afishk
sudo systemctl disable --now "afishk-input-$(id -u).service"
```

Remove the exact installed binary, desktop entry and service files manually if
you no longer need them. Keep configuration/tokens only if you intend to reuse
them. The dedicated ydotool service may be retained independently of the app.
