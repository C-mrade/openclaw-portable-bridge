# MVP test status

## Passed in the reference deployment

- Static Windows x64 client and launcher with no guest runtime dependency.
- Signed manifest and payload verification; modified payload rejected.
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

## Deferred

- Native Windows GUI/tray and per-command approval dialog.
- Official OpenClaw Node v4 adapter and Telegram approval buttons.
- Authenticode signing.
- Restrictive-network test matrix and HTTPS-port fallback.
- Forced termination, USB removal, junction abuse, large-transfer, output
  truncation, cancellation, and concurrent-command live tests.
- Runtime acceptance on Windows ARM64, Linux ARM64, and both macOS
  architectures; macOS signing and notarization.

Operators must repeat the full acceptance suite in their own environment; the
results above are not a security certification.
