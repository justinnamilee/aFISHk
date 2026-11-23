aFISHk @VERSION@ for Windows 11 (Intel/AMD x64)

Open aFISHk from the Start Menu. On first run, its local browser wizard helps
you identify the focused WoW window, choose your LAN/VPN address and save a
random browser password. Click Finish setup to start the companion.
Use the displayed address on your phone; any username works with that password.

You provide FishFry Forever (manual cast/reel bound to your fishing key, default F), VPN, DNS and firewall
rules. No AutoIt, developer tools or separate FFmpeg installation is needed.
Run WoW and aFISHk as your normal signed-in user, not as administrator.
Keep WoW focused and the desktop unlocked. Windowed/borderless mode is recommended.
Audio includes all sound on the Windows default playback device.
Default ports: TCP 8787 and UDP 49160-49180, restricted to your trusted LAN/VPN.
Plain HTTP Basic authentication is suitable only inside a trusted LAN/VPN.

Start Menu has aFISHk, Stop aFISHk, Settings folder and Logs folder shortcuts.
The installer offers optional start at sign-in, off by default. Rerun it to change
that setting. Upgrades stop the running copy and preserve your configuration.
Choose Open aFISHk on the finish page or start it yourself after an upgrade.
Uninstall removes the program and shortcuts, but preserves your user data.

Program: %LOCALAPPDATA%\Programs\aFISHk
Config/password: %APPDATA%\afishk\config.json and token
Screenshot/logs: %LOCALAPPDATA%\afishk

FFmpeg is included under LGPL-2.1-or-later, using Opus under its BSD license.
See licenses\NOTICE.txt and the matching media source archive beside the
installer. All dependencies retain their own copyrights and license terms.
aFISHk: GPL-3.0-or-later; see LICENSE.

Instructions and matching source:
https://github.com/justinnamilee/aFISHk/tree/@VERSION@
