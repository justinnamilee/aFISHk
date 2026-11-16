# Maintaining aFISHk

## Scope

- FishFry is a user-managed prerequisite. Link to its publisher in README.md; do not install, download, update or manage addons.
- Keep Go packages small and frontend assets plain HTML/CSS/JS, embedded in the binary.
- Use Go and Bash for application tooling and the standalone dprint binary for frontend formatting. Do not add Node/npm, a frontend framework, bundler, or Python dependency.
- Maintain user instructions, troubleshooting and manual checks in README.md. Keep agent guidance here; do not recreate separate validation/architecture documents.

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
- Use `make fmt`: gofmt for Go, shfmt for Bash, dprint for HTML/CSS/JS. Pin the dprint binary version/checksums in `scripts/install-dprint.sh` and plugin versions/checksums in `dprint.json`.
- Run `make check`, `make lint` and `make build` before delivery. FFmpeg must be available so the audio integration test runs. Report any skipped or blocked checks accurately.
- For packaging changes, run `make package VERSION=v1.0.0-rc.1` and inspect both archives. Keep build/install usable without Node or formatter dependencies. Release archives must contain only the root binary, sample config, README, LICENSE, `scripts/linux` runtime helpers and `inc/linux` integration files; no Makefile, source or developer tools. Installer tests use temporary destinations and mocked system commands; never invoke real desktop setup in CI.
- Automated tests cannot validate Nobara/KDE hardware, FishFry, Android playback or VPN latency. Use the README checklist and state which manual checks remain.
- Keep GitHub Actions pinned to commit SHAs. Release workflows must create drafts and refuse to modify published releases.
