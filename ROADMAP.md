# Roadmap

The roadmap is organized around evidence-producing milestones. A milestone is
complete only when its acceptance tests, failure behavior, documentation, and
upgrade path are complete. The long-term gates live in
[Production vision](docs/PRODUCTION_VISION.md).

## M1 — agent-native operator path (`0.6`)

- Automatically register and probe the local MCP adapter in OpenClaw or Hermes.
- Deliver pending-pairing events into an existing private agent conversation
  without owning a Telegram bot or exposing the administrator token.
- Bind every approval to its originating conversation and verified comparison
  code.
- Add end-to-end tests for approve, reject, timeout, revoke, restart, and
  duplicate delivery.

Exit gate: a fresh operator can configure environment variables, run one setup
command, and approve a real USB guest from their existing agent conversation.

## M2 — trustworthy guest experience (`0.7`)

- Replace the console with a native, dependency-free guest window and tray
  indicator showing identity, capabilities, expiry, activity, and Stop.
- Add granular sensitive-action consent, pause/resume, transfer progress, and
  accessible diagnostics.
- Keep the portable profile installation-free and clean up all Bridge-owned
  temporary state after exit or USB removal.

Exit gate: a non-technical guest can understand and terminate the session
without using a terminal.

## M3 — resilient transport and updates (`0.8`)

- Finish broker identity pinning and signed public USB configuration.
- Add an atomic updater with signature verification, rollback, and cleanup.
- Complete ConPTY/PTY sequencing, resize, backpressure, reconnect, and
  transcript visibility.
- Run fault injection for network loss, broker/guest restart, USB removal,
  interrupted transfer, locked files, full disks, and clock skew.

Exit gate: recovery never silently duplicates uncertain work and every update
can be rolled back.

## M4 — explicit trusted-device profile (`0.9`)

- Offer a separate, opt-in installation with normal UAC/OS consent, visible
  tray state, outbound-only connectivity, OS-keystore device credentials,
  expiry, rotation, pause, revoke, and uninstall.
- Require a fresh agent approval for each remote session.
- Never turn the portable profile into silent persistence.

Exit gate: removing the USB does not break an explicitly trusted session, and
the guest can revoke or uninstall it locally at any time.

## M5 — production-stable release (`1.0`)

- Reproducible releases, SBOM, provenance, platform signing, key rotation, and
  incident/rollback runbooks.
- Continuous fuzzing and adversarial tests plus representative Windows, Linux,
  and macOS hardware validation.
- Independent security review with release-blocking findings resolved.
- Stable protocol compatibility policy and telemetry-free diagnostics bundle.

## Release-quality gates

- Complete adversarial network, authentication, command, transfer, and cleanup
  test matrix.
- Reproducible release procedure, SBOM, provenance, and signed source releases.
- Stable protocol and documented upgrade/rollback path.
- Independent security review before a production-stable claim.

See [docs/STATUS.md](docs/STATUS.md) for current implementation details.
