# MVP test status

## Passed in the reference deployment

- Static Windows x64 client and launcher with no guest runtime dependency.
- Signed manifest and payload verification; modified payload rejected.
- Signed public configuration verification; modified endpoint or USB identity
  rejected before network access.
- Clean staged release assembly, complete checksum verification, and
  mixed-version package rejection.
- Canonical signing-key discovery plus a full `0.6.0-beta.1` six-target build
  under a rotated trust root, with key-pair and complete checksum verification.
- Atomic broker-state snapshots with restrictive permissions.
- Clear session tokens excluded from durable snapshots, with approval-token
  recovery after restart and established-session authentication by hash.
- Restart recovery requeues unacknowledged leases without replaying commands
  already acknowledged as running.
- Corrupt or incompatible state fails closed.
- MCP tool calls keep the administrator token inside the local adapter.
- MCP result retrieval always requests the bounded `untrusted_guest_data`
  view; malicious control characters, oversized output, integrity metadata,
  and explicit raw fallback have regression coverage.
- Credential-free active-session discovery exposes capabilities, expiry,
  queue depth, and command-state counts without token/hash disclosure.
- OpenClaw proactive approval delivery through the existing private
  conversation, with idempotent notification markers written only after
  successful delivery.
- OpenClaw Telegram callbacks handled before the agent queue with explicit
  sender/chat allowlists, exact request/code matching, visible terminal state,
  stale-button removal, and fail-closed tests for unauthorized, expired, and
  mismatched callbacks.
- Live Windows x64 approval on `STANPC` followed by an independently queued
  and consumed `system.info` result.
- Authentic payload staged under `%TEMP%` and removed after exit.
- Ephemeral Ed25519 pairing, comparison code, approval, expiry, and distinct
  pairing/session tokens.
- Replay cache and per-source pairing rate limit.
- `system.info`, `process.list`, harmless `shell.run`, and scoped file
  write/read/list operations.
- `session.disconnect`, server-side revocation, and application-owned cleanup.
- Unit tests for signatures, path boundaries, traversal, no-overwrite, and
  token hashing.
- Public TLS reverse-proxy path to a loopback-only broker.
- Stable child-process working directory independent of the removable drive.
- Idempotent retries, conflicting-ID rejection, delivery leases,
  acknowledgement, result correlation, and structured queue saturation.
- Transient poll failures retry with bounded backoff; authoritative session
  rejection remains terminal and does not trigger a redundant client revoke.
- Windows x64 protocol-v2 acceptance on `STANPC`: delivery ACK/result
  correlation, stable bridge-owned working directory, Unicode PowerShell,
  idempotent command replay and conflicting-ID rejection, resumable 600 KiB
  transfer with final SHA-256, asynchronous process-tree cancellation, queue
  saturation/backpressure, survival across multiple idle long-poll cycles,
  cleanup, and explicit disconnect/revocation.
- Windows x64 live technician audit on `STANPC`: typed system, network, disk,
  service and process inspection; non-elevated shell and PowerShell; visible
  scoped/full-volume filesystem behavior; immediate revocation and confirmed
  post-revoke rejection.
- Clean `0.6.2-beta.2` agent-first bootstrap, signed six-target package,
  checksum inventory, diagnostics, and idempotent second apply.
- Typed, filterable, bounded service and process inventories, including
  pagination validation and compact match/return summaries.

## Deferred

- Native Windows GUI/tray and per-command approval dialog.
- Equivalent proactive approval/callback adapters for Hermes and agents other
  than OpenClaw Telegram.
- Remaining live approval cases: untouched-request timeout and duplicate
  callback delivery.
- Authenticode signing.
- Restrictive-network test matrix and HTTPS-port fallback.
- Forced termination, USB removal, junction abuse, large-transfer, output
  truncation, cancellation, and concurrent-command live tests.
- Runtime acceptance on Windows ARM64, Linux ARM64, and both macOS
  architectures; macOS signing and notarization.

Operators must repeat the full acceptance suite in their own environment; the
results above are not a security certification.
