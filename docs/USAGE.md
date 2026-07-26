# Usage

1. Insert the prepared USB and open its `OPENCLAW_BRIDGE` directory.
2. Open the clearly labelled root launcher for Windows, Linux, or macOS. The
   Linux convenience launcher is a native ELF file with an `.exe` suffix so
   FAT/VFAT `showexec` mounts expose it as executable; it never uses Wine.
   Architecture-specific launchers remain under `launchers/`. No administrator
   rights are required.
3. Choose **Audit** or **Developer**. Audit provides fixed,
   read-only system, network, disk, service and process inspections. It may
   also read files from one directory selected locally (use `C:\\` to grant
   the whole system volume). Developer requires typing `DEVELOPER`, grants
   terminal and file access across all available volumes with the current
   user's rights, and exposes a separate administrator command capability.
   On Windows, every administrator command displays a normal local UAC prompt.
   Linux and macOS builds intentionally provide no remote elevation.
4. Compare the six-character code shown locally with the approval channel.
5. Approve once or for a bounded duration. Keep the console visible.
6. Review every received command in the activity output.
7. Use `session.disconnect`, Ctrl+C, or close the window. The session token is
   revoked and launcher-owned `%TEMP%\\OpenClawBridge\\<session>` is removed.

Developer automation can use `shell.start`, `shell.status`, and `shell.cancel`
for long-running cancellable jobs. Large files can be transferred through
`files.read-chunk` and `files.write-chunk`; final writes require the expected
whole-file SHA-256. Directory listings accept `offset`, `limit`, and `filter`.
`service.list` and `process.list` accept the same optional pagination fields
and return typed `items`, a compact `summary`, and `hasMore`. Their default page
contains 100 items and the maximum accepted page contains 500.

Every received capability is shown in the guest activity stream, while bulk
transfer payloads are never printed. Developer intentionally retains complete
user-level freedom for the approved technician; use Audit when broad shell and
filesystem access are unnecessary.

Configure your own HTTPS broker endpoint in `bridge-public.json`. Expose only
the loopback pairing broker through a hardened TLS reverse proxy or Tailscale
Funnel. Starting the launcher requires explicit local consent and a separate
server-side approval before any capability becomes active.

The public repository intentionally contains no ready-to-use operational
configuration, private signing key, session token, or deployment-specific
binary. Operators create their own signed USB package once; Windows guests only
receive the resulting portable directory.
