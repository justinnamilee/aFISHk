# Maintaining aFISHk

## Scope

- FishFry is a user-managed prerequisite. Link to its publisher in README.md; do not install, download, update or manage addons.
- Keep Go packages small and frontend assets plain HTML/CSS/JS, embedded in the binary.
- Use Go and Bash for application tooling and the standalone dprint binary for frontend formatting. Do not add Node/npm, a frontend framework, bundler, or Python dependency.
- README.md is strictly end-user facing. Limit it to prerequisites, installation (including building from source), configuration, everyday use, updates and troubleshooting.
- Never add validation plans or results, test/acceptance checklists, pending manual checks, reviewer questions, approval requests or agent questions to README.md. Put these strictly in the relevant PR description or PR discussion, including skipped or blocked checks and outstanding decisions.
- Keep agent guidance, development workflows and backend architecture/extension guidance in AGENTS.md, not README.md. Do not create separate validation/architecture documents. Apply the same end-user-only rule to release README templates and generated archive READMEs.

## Behavior to preserve

- One explicit tap sends one F key. Never add automatic input, cast/reel scheduling, bite detection, arbitrary keys, or automatic input retries.
- Reject desktop actions when focus, exact game identity or lock state is unknown. Never activate a window to make an action succeed.
- Capture the verified KWin UUID; never fall back to active-window or whole-desktop capture, or bypass KWin permissions.
- Preserve the last successful PNG on failure and keep cache files private.
- Keep privileged ydotoold separate from the unprivileged desktop HTTP service. Installation must preserve configuration and must not start control.
- Preserve HTTP authentication, origin and CSRF checks. Never log tokens or place them in URLs.
- Keep audio to one peer with lease cleanup. Respect configured UDP bounds and render activity messages as text.

## Development and review

- Run `make tools` after installing Go, ShellCheck, curl and unzip; include the Go binary installation directory in PATH. dprint installs locally in `.tools/`.
- Use `make fmt`: gofmt for Go, shfmt for Bash, dprint for HTML/CSS/JS. Pin the dprint binary version/checksums in `scripts/dprint.sh` and plugin versions/checksums in `dprint.json`.
- Run `make check`, `make lint` and `make build` before delivery. FFmpeg must be available so the audio integration test runs. Record validation results and any skipped or blocked checks accurately in the PR, never in README.md.
- For packaging changes, run `make package VERSION=v1.0.0-rc.1` and inspect both archives. Keep build/install usable without Node or formatter dependencies. Release archives must contain only the root binary, sample config, README, LICENSE, `scripts/linux` runtime helpers and `inc/linux` integration files; no Makefile, source or developer tools. Installer tests use temporary destinations and mocked system commands; never invoke real desktop setup in CI.
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
- Maintain Windows cross-compilation as an architecture check; a successful build does not establish Windows runtime support.
- Add tests for the interface contracts and resource cleanup. Record backend-specific acceptance checks, results and remaining manual validation in the PR only, never in README.md.
