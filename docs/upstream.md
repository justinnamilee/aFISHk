# Upstream references

External components are installed independently; their code is not vendored.
Go versions and integrity hashes are recorded in `go.mod` and `go.sum`.

- [FishFry Forever publisher](https://www.curseforge.com/wow/addons/fishfry-forever): select a game-compatible ZIP and review the publisher's license.
- [KWin scripting API](https://develop.kde.org/docs/plasma/kwin/api/): activeWindow, internalId, resourceClass, caption and callDBus.
- [KWin Plasma 6.4 screenshot implementation](https://invent.kde.org/plasma/kwin/-/blob/Plasma/6.4/src/plugins/screenshot/screenshotdbusinterface2.cpp): CaptureWindow, raw pixels, dimensions, format, stride, descriptor transport and authorization.
- [KWin scripting implementation](https://invent.kde.org/plasma/kwin/-/blob/Plasma/6.4/src/scripting/scripting.cpp): transient loadScript/run/unloadScript lifecycle.
- [Qt QImage formats](https://doc.qt.io/qt-6/qimage.html#Format-enum): supported SDR pixel layouts.
- [ydotool](https://github.com/ReimuNotMoe/ydotool): Linux input keycodes, persistent daemon and socket permissions.
- [FFmpeg devices](https://ffmpeg.org/ffmpeg-devices.html#pulse): Pulse capture input.
- [Pion WebRTC](https://github.com/pion/webrtc): Go WebRTC transport; [RTP example](https://github.com/pion/webrtc/tree/main/examples/rtp-to-webrtc).

FishFry retains its publisher's license. ydotool's upstream repository identifies
its license as AGPL-3.0; it is invoked as a separately installed executable.
FFmpeg, PipeWire, KDE and Pion retain their respective upstream licenses.
