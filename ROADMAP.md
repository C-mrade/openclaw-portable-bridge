# Roadmap

The roadmap is organized around evidence-producing milestones. A milestone is
complete only when its acceptance tests, failure behavior, documentation, and
upgrade path are complete. The long-term gates live in
[Production vision](docs/PRODUCTION_VISION.md).

## M1 — agent-native operator path (`0.6`)

- Provide one guided installer that creates deployment secrets, configures the
  agent integration, builds the signed package, prepares USB media, and runs
  actionable diagnostics.
- Support a fully explicit non-interactive setup path for automation and clean
  CI acceptance.
- Keep HTTPS publication an explicit operator choice; never trade onboarding
  convenience for direct broker exposure or shared trust roots.
- Automatically register and probe the local MCP adapter in OpenClaw or Hermes.
- Deliver pending-pairing events into an existing private agent conversation
  without owning a Telegram bot or exposing the administrator token.
- Bind every approval to its originating conversation and verified comparison
  code.
- Add end-to-end tests for approve, reject, timeout, revoke, restart, and
  duplicate delivery.

Exit gate: a fresh operator can run one guided command, provide their HTTPS
endpoint and optional private approval IDs, prepare a verified USB, and approve
a real guest from their existing agent conversation.

### M1.1 — agent-owned bootstrap (`0.6.1`)

- Expose read-only host discovery and deterministic install plans as JSON.
- Separate agent-executable work from human consent gates.
- Make apply resumable and idempotent without persisting generated secrets.
- Automate an approved Tailscale Funnel publication while keeping the broker
  loopback-only.
- Provide machine-readable status and diagnostics for autonomous recovery.
- Keep the interactive installer as a human fallback.

Exit gate: an OpenClaw or Hermes agent can take a clean supported Linux host
from clone to a verified signed package, asking the human only to approve a new
HTTPS trust/publication boundary and other genuinely sensitive actions.

### M1.2 — hostile guest boundary (`0.6.2`)

- Never persist clear session tokens while preserving restart recovery.
- Provide credential-free session discovery and command-state summaries.
- Mark, sanitize, hash, and bound guest results before they reach an agent.
- Preserve an explicit raw diagnostic path without making it the default.
- Show every received capability in the visible guest activity stream.
- Keep full Developer freedom while making the read-only Audit path clear.

Exit gate: a technician retains complete approved control, while a malicious
guest cannot silently turn bulk output into unbounded trusted agent input or
recover reusable operator credentials from portable media or broker state.

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
