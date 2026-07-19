# Usage

1. Insert the USB and open `OPENCLAW_BRIDGE`.
2. Double-click `OPENCLAW BRIDGE.exe`; no administrator rights are required.
3. Choose **Information** or **Developer**. Developer requires typing one
   existing directory that the Bridge may access.
4. Compare the six-character code shown locally with the approval channel.
5. Approve once or for a bounded duration. Keep the console visible.
6. Review every received command in the activity output.
7. Use `session.disconnect`, Ctrl+C, or close the window. The session token is
   revoked and launcher-owned `%TEMP%\\OpenClawBridge\\<session>` is removed.

Configure your own HTTPS broker endpoint in `bridge-public.json`. Expose only
the loopback pairing broker through a hardened TLS reverse proxy or Tailscale
Funnel. Starting the launcher requires explicit local consent and a separate
server-side approval before any capability becomes active.
