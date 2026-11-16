# Maintaining aFISHk

## Scope

- FishFry is a user-managed prerequisite. Link to its publisher in README.md; do not install, download, update or manage addons.
- Keep Go packages small and frontend assets plain HTML/CSS/JS, embedded in the binary.
- Use Go and Bash for application tooling, NSIS for the Windows installer and the standalone dprint binary for frontend formatting. Do not add Node/npm, a frontend framework, bundler, or Python dependency.
- README.md is strictly end-user facing. Limit it to prerequisites, installation (including building from source), configuration, everyday use, updates and troubleshooting.
- Never add validation plans or results, test/acceptance checklists, pending manual checks, reviewer questions, approval requests or agent questions to README.md. Put these strictly in the relevant PR description or PR discussion, including skipped or blocked checks and outstanding decisions.
- Keep agent guidance, development workflows and backend architecture/extension guidance in AGENTS.md, not README.md. Do not create separate validation/architecture documents. Apply the same end-user-only rule to release README templates and generated archive READMEs.

## Behavior to preserve

- One explicit tap sends one F key. Never add automatic input, cast/reel scheduling, bite detection, arbitrary keys, or automatic input retries.
- Reject desktop actions when focus, exact game identity or lock state is unknown. Never activate a window to make an action succeed.
- Capture the verified native window identity (KWin UUID on Linux; HWND/process identity on Windows); never fall back to active-window or whole-desktop capture, or bypass platform permissions.
- Preserve the last successful PNG on failure and keep cache files private.
- Keep privileged Linux ydotoold separate from the unprivileged desktop HTTP service. Run Windows in the signed-in user's session, never as a system service or elevated process. Installation must preserve configuration and must not start control unless the user explicitly opts to launch the app.
- Preserve HTTP authentication, origin and CSRF checks. Never log tokens or place them in URLs.
- Keep audio to one peer with lease cleanup. Respect configured UDP bounds and render activity messages as text.

## Development and review

- Run `make tools` after installing Go, ShellCheck, curl and unzip; include the Go binary installation directory in PATH. dprint installs locally in `.tools/`.
- Use `make fmt`: gofmt for Go, shfmt for Bash, dprint for HTML/CSS/JS. Pin the dprint binary version/checksums in `scripts/dprint.sh` and plugin versions/checksums in `dprint.json`.
- Run `make check`, `make lint` and `make build` before delivery. FFmpeg must be available so the audio integration test runs. Record validation results and any skipped or blocked checks accurately in the PR, never in README.md.
- For packaging changes, run `make package VERSION=v1.0.0-rc.1` and inspect both archives. Keep build/install usable without Node or formatter dependencies. Linux runtime archives must contain only the root binary, sample config, README, LICENSE, `scripts/linux` runtime helpers and `inc/linux` integration files; no Makefile, source or developer tools. Installer tests use temporary destinations and mocked system commands; never invoke real desktop setup in CI.
- Automated tests cannot validate Nobara/KDE hardware, FishFry, Android playback or VPN latency. Put the required manual checks and their completion status in the PR: focused one-tap/one-F input, rejection on wrong focus or locked/unknown session, targeted screenshots and previous-image preservation, desktop/Android audio, and VPN/reconnection behavior. Do not add or refer contributors to a README validation checklist.
- Keep GitHub Actions pinned to commit SHAs. Release workflows must create drafts and refuse to modify published releases.

## Backend architecture and extension

Backends are compiled-in Go implementations selected in `internal/platform`,
not runtime-loaded plugins. Preserve existing JSON configuration keys
and audio settings when extending or refactoring backends.

| Interface | Contract |
| --- | --- |
| `desktop.WindowSource` | Return the focused native window only after verifying the session is unlocked; fail if either state is unknown. Never activate a window. |
| `desktop.Capturer` | Capture only the supplied opaque native window ID; never fall back to the active window or whole desktop, or bypass platform permissions. |
| `desktop.Input` | Send one F press/release to the verified target, without window activation or key-down retries. Best-effort key-up is permitted after failure. |
| `audio.Capture` / `audio.Process` | Start an Opus RTP producer for the supplied loopback destination (48 kHz, stereo, payload type 111). Cancellation must stop the producer and unblock its single `Wait` call; failed startup must release resources. |

The shared desktop controller owns exact class/title matching, action exclusion,
input throttling and private screenshot persistence. The shared audio manager
owns WebRTC, UDP port bounds, the single-peer limit, cancellation and lease
cleanup. The HTTP server consumes interfaces for desktop actions and audio
sessions. Keep these shared safeguards independent of backend implementations.

To add a backend:

- Put platform implementations behind OS-specific files or build constraints and wire them in `internal/platform`.
- Add platform defaults and validation in `internal/config`, and platform-specific prompts in `internal/setup`.
- Keep KWin, ydotool and Pulse/FFmpeg implementations Linux-only. Preserve explicit unsupported-platform errors until a platform has a working implementation.
- Compile Windows code and test binaries on Linux; compilation is not Windows runtime validation. Keep actual Windows acceptance results and outstanding checks in the PR.
- Add tests for the interface contracts and resource cleanup. Record backend-specific acceptance checks, results and remaining manual validation in the PR only, never in README.md.

## Windows and shared first-run setup

- All development, builds and packaging run on Linux. Do not add Windows-hosted build scripts, CI runners or toolchain setup. `make windows` cross-compiles the GUI executable; `make release VERSION=vX.Y.Z` builds both Linux archives and the Windows x64 installer/source companion. `make package` still builds Linux archives only.
- Linux cross-packaging prerequisites: NSIS, MinGW-w64 x86-64 GCC/binutils and runtime notices, pkg-config, make, curl and xz-utils. On Debian/Ubuntu: `apt-get install nsis gcc-mingw-w64-x86-64 binutils-mingw-w64-x86-64 mingw-w64-common pkg-config make curl xz-utils`. No AutoIt or Windows build environment is needed.
- Keep Windows users' flow to one per-user installer, Start Menu launch and browser setup. Start at sign-in is optional and off by default; upgrades preserve the existing choice. Never configure firewall rules, VPN, DNS or addons.
- Install under `%LOCALAPPDATA%\Programs\aFISHk`; store config/token under `%APPDATA%\afishk` and screenshots/logs under `%LOCALAPPDATA%\afishk`. Protect data with user/SYSTEM DACLs, not POSIX chmod assumptions. Preserve user data on upgrades and uninstall.
- Native Windows integration uses Win32 focus/session checks, `SendInput`, a bounded targeted `PrintWindow` helper and WASAPI playback loopback. Reject locked, disconnected, secure or unknown sessions. Match the executable path as well as class/title. Do not elevate to reach an elevated game.
- Windows screenshots must never use screen/desktop capture as a fallback. Terminate the disposable capture worker on timeout; preserve the last successful image on failure. Keep Windows audio in a COM-initialized pinned thread and release every native buffer/object before thread teardown.
- The bundled FFmpeg only encodes/muxes audio. Build it on Linux from pinned, SHA-256-verified FFmpeg/Opus sources with GPL and nonfree options disabled. Ship notices and the matching upstream sources plus exact build script beside every installer. Do not replace this with an untracked third-party binary or omit the source companion from releases.
- First-run browser setup is shared by Windows and Linux (`afishk setup --web`). Bind it only to an ephemeral loopback port, enforce exact Host/Origin and CSRF checks, prohibit framing/cross-origin reads, expire it, and never overwrite existing configuration. Do not put passwords in URLs or logs. Generate a strong password and show it only in the local completion page.
- Keep existing terminal setup and Linux configuration compatible. Keep Windows-only defaults and prompts in platform files. The wizard is not a remotely accessible administration interface.
- Windows lifecycle uses one per-user/session instance and an explicit stop event. Installer upgrades stop the old copy before replacement. Do not automatically restart control after installation; use the optional finish-page launch or Start Menu.
- Run Linux formatting, lint, race tests, installer tests and release checks. Also run `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...`, the Windows linter, and cross-compile Windows tests with `go test -exec /bin/true`; label that last command as compilation only in the PR. Inspect installer payloads, PE imports and source/license bundles on Linux.
- Runtime acceptance belongs in the PR: Windows 11 install/upgrade/uninstall, sign-in startup, paths with spaces/non-ASCII, DACLs, focused input and rejection paths, screenshot behavior/DPI/timeout, WASAPI device changes and cleanup, phone/VPN playback, and the browser wizard on both platforms. Do not represent Linux compilation or Wine as a substitute for these checks.
