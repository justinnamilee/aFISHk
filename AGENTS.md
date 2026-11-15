# Maintaining aFISHk

- Keep Go packages small and frontend assets plain HTML/CSS/JS.
- Use Go and Bash for project tooling. Do not add Python or npm dependencies.
- Never add automatic input, cast/reel scheduling, bite detection or arbitrary keys.
- Reject desktop actions when focus, identity or lock state is unknown.
- Capture a verified KWin UUID; do not fall back to active-window or desktop capture.
- Preserve the latest successful PNG on failure and keep cache permissions private.
- Keep privileged ydotoold separate from the unprivileged desktop HTTP service.
- Run `make check`, `make lint` and `shellcheck scripts/*.sh` for relevant changes.
- Desktop acceptance gates must be recorded on real Nobara/KDE hardware.
- Release workflows must leave releases in draft and refuse published-release edits.
