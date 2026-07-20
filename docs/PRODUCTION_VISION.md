# Production vision

OpenClaw Portable Bridge should become a portable, consent-driven remote
assistance tool that leaves no standing access on the guest. A non-technical
guest should be able to understand who is connected, what they may do, what is
happening now, and how to stop it. An operator should be able to recover from
disconnects and crashes without guessing whether an action ran twice.

The project is not intended to become a silent remote administration agent.
Its defining properties are explicit local consent, least privilege, visible
activity, short-lived authority, verifiable releases, and clean removal after
the session.

## Production invariants

A production-stable release must demonstrate all of the following:

1. **No standing access.** The USB contains no reusable Gateway, tailnet, or
   administrator credential. Closing or revoking a session removes its remote
   authority.
2. **Informed consent.** The guest sees the operator identity, requested
   capabilities, comparison code, expiry, current activity, and a local stop
   control. Sensitive actions require specific confirmation.
3. **Crash-safe delivery.** Durable command state distinguishes queued,
   leased, running, completed, cancelled, expired, and uncertain operations.
   Recovery never blindly re-executes an uncertain non-idempotent action.
4. **Authenticated bootstrap.** Release payloads and public USB configuration
   are signed. The guest pins the broker identity established by the trusted
   release rather than trusting an editable endpoint alone.
5. **Least authority.** Approval, execution, auditing, and revocation use
   separate scoped identities. Agent-facing adapters never expose a broker
   administrator credential to a model.
6. **Safe filesystem semantics.** Transfers are bounded, staged, hashed, and
   committed atomically. Symlinks, junctions, reparse points, device files,
   races, locked files, full disks, and concurrent writers are handled by
   explicit policy and adversarial tests.
7. **Immediate control.** Revoke, cancel, and heartbeat traffic cannot be
   starved by ordinary command or transfer queues. Owned process trees are
   terminated reliably on every supported platform.
8. **Tamper-evident accountability.** Audit events are minimized, redacted,
   monotonically ordered, hash chained, retained by policy, and checkpointed
   outside the broker's mutable working state.
9. **Verifiable delivery.** Supported packages are reproducible, signed with
   platform-appropriate identities, accompanied by SBOM and provenance, and
   exercised on representative hardware.
10. **Failure is a tested mode.** Network loss, broker restart, guest restart,
    USB removal, storage corruption, expiry, clock skew, saturation, and
    interrupted upgrades have automated acceptance criteria.

## Delivery strategy

Work proceeds in small, reviewable slices:

```text
requirement -> threat analysis -> implementation -> unit/race/fuzz tests
-> cross-build -> representative runtime test -> failure injection
-> documentation -> CI evidence -> release candidate
```

Features that increase authority or hide operational ambiguity do not ship
merely because the happy path works. A graphical UI, Telegram approval, and
agent integrations are built only on top of durable recovery and an explicit
trust model.

## Release horizons

### 0.6 — durable foundation

- durable broker state and schema migrations;
- restart recovery for sessions, commands, leases, results, and revocations;
- dedicated atomic updater with bounded rollback;
- real network-loss and broker-restart tests;
- authenticated USB configuration and broker identity;
- priority control path for revoke, cancel, and heartbeat.

### 0.7 — understandable consent

- native guest activity and consent UI;
- granular profiles and per-action sensitive approval;
- typed operator CLI and Telegram approval built on scoped credentials;
- safe large-output artifacts and transfer progress.

### 0.8 — agent and terminal integration

- typed local MCP/Node adapter with a policy boundary outside the model;
- persistent ConPTY/PTY sessions with sequencing, flow control, resize,
  reconnect, transcript visibility, and immediate stop;
- complete cross-platform process-tree containment.

### 1.0 — production-stable claim

- signed and reproducible platform packages with SBOM and provenance;
- representative Windows, Linux, and macOS hardware validation;
- continuous fuzzing and adversarial failure matrix;
- documented upgrade, rollback, key rotation, and incident procedures;
- independent security review with release-blocking findings resolved.
