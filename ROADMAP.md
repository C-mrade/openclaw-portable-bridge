# Roadmap

OpenClaw Portable Bridge is an experimental MVP. Roadmap items are ordered by
what most improves recovery safety and understandable consent. The long-term
release criteria live in [Production vision](docs/PRODUCTION_VISION.md).

## 0.6 — durable foundation

- Durable broker storage for sessions, token hashes, commands, leases,
  results, revocations, and schema migrations.
- Explicit restart recovery for queued, leased, running, expired, and
  uncertain commands; uncertain non-idempotent work is never retried blindly.
- A dedicated updater that verifies, swaps, rolls back, and cleans up after the
  launcher exits.
- Failure-injection tests for physical network loss, broker restart, USB
  removal, interrupted transfer, locked files, and full disks.
- Signed public USB configuration and broker identity pinning.
- Priority handling for revoke, cancel, and heartbeat.
- Protocol version negotiation and compatibility tests.

## 0.7 and later

- Native visible guest UI for activity, transfers, granular consent, pause,
  and revocation.
- Typed operator CLI, scoped local adapter, and Telegram approval flow.
- Complete persistent ConPTY transport with input, resize, acknowledgements,
  cancellation, and backpressure.
- macOS `.app` packaging, Developer ID signing, and notarization.
- Authenticode signing support for Windows releases.
- Representative Windows ARM64, Linux ARM64, and macOS hardware validation.

## Release-quality gates

- Complete adversarial network, authentication, command, transfer, and cleanup
  test matrix.
- Reproducible release procedure, SBOM, provenance, and signed source releases.
- Stable protocol and documented upgrade/rollback path.
- Independent security review before a production-stable claim.

See [docs/STATUS.md](docs/STATUS.md) for current implementation details.
