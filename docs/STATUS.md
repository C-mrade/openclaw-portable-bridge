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

Release `0.5.1-mvp-dev` additionally fixes removable-drive current-directory
failures, requires command-delivery acknowledgement, makes command IDs
idempotent within each session, validates results against running commands,
returns structured queue-pressure information, and keeps transient polling
failures from closing and revoking an otherwise valid session.

Release `0.5.2-mvp-dev` makes packaging fail closed: releases are assembled in
a clean staging directory, checksummed as a complete unit, and published only
after verification. The launcher now enforces signed version compatibility and
a signed public USB configuration, preventing mixed-version packages and
silent broker endpoint replacement.

The current `0.6.0-alpha.2` work adds atomically persisted broker state,
restart recovery that requeues only unacknowledged leases, explicit pending
and reject APIs, a typed local operator CLI, a standalone MCP adapter for
agent-native approval, a reserved control-command lane, and one-command
operator/service/skill installation with automatic OpenClaw/Hermes MCP
registration and probing.

The broker keeps a live in-memory model that is durably mirrored and recovered;
a future schema
migration layer is still required before the durable format is declared
stable. OpenClaw now has proactive private-conversation notification and a
direct Telegram callback plugin that validates explicit sender/chat allowlists
before the agent queue. Broker identity pinning, a dedicated updater, broader
conversation adapters, and real network-failure injection remain immediate
work; see `PRODUCTION_VISION.md`.

A Windows x64 proof of concept has exercised pairing, `system.info`, process
listing, a harmless user-level shell command, scoped file operations,
disconnect, signature rejection, and launcher-owned temporary cleanup.

Linux x64 has compile and local smoke coverage. Linux ARM64, Windows ARM64,
macOS Intel, and macOS Apple Silicon are cross-compiled and manifest-verified,
but still require representative hardware acceptance testing. Non-Windows
Developer mode provides user-level shell, process, and file capabilities only.

Outstanding work includes a native graphical UI, complete ConPTY streaming,
equivalent direct approval handlers outside OpenClaw Telegram, Authenticode
signing, and the remaining network and Windows adversarial test cases listed in
`TEST_REPORT.md`.
