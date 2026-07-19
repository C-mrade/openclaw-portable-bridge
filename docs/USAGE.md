# Usage

1. Insert the USB and open `OPENCLAW_BRIDGE`.
2. Double-click `OPENCLAW BRIDGE.exe`; no administrator rights are required.
3. Choose **Information** or **Developer**. Information provides fixed,
   read-only system, network, disk, service and process inspections. It may
   also read files from one directory selected locally (use `C:\\` to grant
   the whole system volume). Developer requires typing `SVILUPPATORE`, grants
   terminal and file access across all available volumes with the current
   user's rights, and exposes a separate administrator command capability.
   Every administrator command displays a normal local Windows UAC prompt.
4. Compare the six-character code shown locally with the approval channel.
5. Approve once or for a bounded duration. Keep the console visible.
6. Review every received command in the activity output.
7. Use `session.disconnect`, Ctrl+C, or close the window. The session token is
   revoked and launcher-owned `%TEMP%\\OpenClawBridge\\<session>` is removed.

Developer automation can use `shell.start`, `shell.status`, and `shell.cancel`
for long-running cancellable jobs. Large files can be transferred through
`files.read-chunk` and `files.write-chunk`; final writes require the expected
whole-file SHA-256. Directory listings accept `offset`, `limit`, and `filter`.

Configure your own HTTPS broker endpoint in `bridge-public.json`. Expose only
the loopback pairing broker through a hardened TLS reverse proxy or Tailscale
Funnel. Starting the launcher requires explicit local consent and a separate
server-side approval before any capability becomes active.
