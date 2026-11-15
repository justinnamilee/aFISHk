# Architecture and limitations

The Go binary serves an embedded browser UI and fixed HTTP actions. Desktop
integration runs in the user's real Plasma session rather than a container.

## Desktop control

Every action queries KWin afresh through a transient script. KWin reports its
active window UUID, resource class and caption to the service's unique D-Bus
connection. Replies need both the nonce and KWin's verified bus sender.
ScreenSaver's lock state must be available and unlocked. Exact configured class
and title must match; empty/unknown targets are rejected. The script never sets
focus. Queries time out and their temporary files/scripts are removed.

Input sends Linux KEY_F (33), down then up, in a single ydotool invocation.
Concurrent desktop actions are rejected instead of queued; rapid repeats within
350 ms are rejected. The browser disables the button while a request is pending
and never retries input. If ydotool fails, a best-effort key-up is attempted,
with no second key-down.

**Remaining input limit:** KWin focus checking and ydotool injection are separate
operations. A focus change or lock between them is a small unavoidable race in
this backend. The guard is fail-closed for all observed wrong/unknown targets,
but is not compositor-atomic. Do not run another local user or focus-changing
applications while remotely fishing. Test this explicitly before use. Window
class/title are an operational guard, not protection against a malicious local
process impersonating the game. Keyboard layout must map KEY_F to the game's F
binding; nonstandard ydotool device layouts need local configuration.

## Screenshot

KWin's native `org.kde.KWin.ScreenShot2.CaptureWindow` is called with the verified
window **UUID**, never an active-window/whole-desktop fallback. It writes raw
QImage pixels through a Unix file descriptor. The service validates metadata,
bounds memory, converts supported SDR 32-bit pixel formats to PNG and fails on
unknown formats (including unsupported HDR layouts).

The PNG is encoded to a private temporary file, synced and renamed over
`latest.png` inside a 0700 cache directory. Files are 0600. Failed requests leave
the previous capture intact. Only the controlled image endpoint serves it, with
no-store and no conditional 304 responses. There is no directory listing.

Some Plasma releases authorize ScreenShot2 through a desktop entry whose Exec
matches the executable. The installer supplies that entry for
`~/.local/bin/afishk`. Use the installed binary, refresh the desktop database,
and log out/in if permission caching persists. Capture denial remains an error;
**never disable KWin's permission checks globally**. Availability and pixel
formats must be tested on the installed Plasma release. Spectacle is unnecessary
because its CLI active-window mode cannot target the verified UUID atomically.

## Audio

One FFmpeg process captures a Pulse monitor provided by pipewire-pulse and
encodes stereo Opus, 48 kHz, 20 ms frames. RTP travels only over an ephemeral
loopback UDP socket into Pion. Pion handles browser negotiation and encrypted
WebRTC media, using IPv4 host candidates and the configured UDP port range.
There are no cloud services, STUN or TURN servers.

A single peer is permitted. The browser sends a five-second heartbeat; the
server closes the peer, UDP socket and FFmpeg process after a 20-second lease
expires. Network failure, explicit Stop, hidden-page handling, FFmpeg exit and
server shutdown release resources. Restart listening after output-device changes.
The browser receives no microphone permission request. A connected WebRTC peer
is transport feedback, not proof that an audible desktop source is producing
sound. Real splash latency is a desktop/VPN acceptance gate.

## HTTP

The listener uses a literal IP. The origin's host is checked on every request;
configure `public_origin` for DNS names, wildcard binds or reverse proxies.
All routes share optional token-as-Basic-password authentication. State changes
are POST-only, require a process-random request token, and reject cross-site
origins. Tokens are read from files, not process arguments or URL parameters.
No shell commands, key names, paths or input schedules are accepted by the API.

Desktop audio may include other applications. Screenshots and control are
sensitive. Bind to the actual VPN address and use existing firewall restrictions.
The application does not create VPNs, open firewall ports, change sleep/lock
settings or recover a logged-out graphical session. Optional HTTPS uses PEM
files only; secrets and certificates are not logged.
