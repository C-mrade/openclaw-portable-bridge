# Project status

The MVP includes a signed Windows launcher, portable client, loopback broker,
ephemeral Ed25519 pairing, explicit approval, bounded capability profiles,
rate limiting, replay protection, revocation, scoped file operations, and
audit logging. Developer mode additionally supports locally confirmed UAC,
cancellable asynchronous shell jobs, output normalization, paginated directory
listings, chunked transfers with SHA-256, consumable broker results, and
low-latency long polling.

The current development release adds Windows Job Object containment for owned
process trees, bounded chunked uploads, structured `powershell.run`, OEM and
UTF-16 decoding, CLIXML filtering, explicit UTF-8 input/output for Windows
PowerShell 5.1, and native ConPTY lifecycle primitives.
ConPTY is not yet exposed as a persistent remote terminal: process attachment,
input/resize protocol messages, delivery acknowledgements, and backpressure
remain required before that capability is advertised.

A Windows x64 proof of concept has exercised pairing, `system.info`, process
listing, a harmless user-level shell command, scoped file operations,
disconnect, signature rejection, and launcher-owned temporary cleanup.

Outstanding work includes a native graphical UI, complete ConPTY streaming, an
OpenClaw Node v4 adapter, Telegram approval flow, Authenticode signing, and the
remaining network and Windows adversarial test cases listed in `TEST_REPORT.md`.
