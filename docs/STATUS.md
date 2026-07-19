# Project status

The MVP includes a signed Windows launcher, portable client, loopback broker,
ephemeral Ed25519 pairing, explicit approval, bounded capability profiles,
rate limiting, replay protection, revocation, scoped file operations, and
audit logging.

A Windows x64 proof of concept has exercised pairing, `system.info`, process
listing, a harmless user-level shell command, scoped file operations,
disconnect, signature rejection, and launcher-owned temporary cleanup.

Outstanding work includes a native graphical UI, per-command approval,
OpenClaw Node v4 adapter, Telegram approval flow, Authenticode signing, and the
remaining network and Windows adversarial test cases listed in
`TEST_REPORT.md`.
