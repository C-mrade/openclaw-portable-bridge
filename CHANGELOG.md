# Changelog

## 0.6.0-alpha.2

- Add automatic, probed MCP registration for OpenClaw and Hermes with
  selectable `auto`, `openclaw`, `hermes`, `both`, and `none` modes.
- Replace the broker-owned Telegram bot with a standalone stdio MCP adapter
  designed for the operator's existing OpenClaw, Hermes, or compatible agent.
- Add six narrow Bridge tools for pending requests, approval, rejection,
  commands, results, and revocation without exposing the administrator token.
- Validate command capability names and bounded deadlines inside the adapter
  before contacting the broker.
- Install `bridge-mcp`, the fallback `bridge-operator` CLI, the broker, and the
  shared agent skill through the same operator setup.
- Remove Telegram credentials and polling from the broker and standard setup.

## 0.6.0-alpha.1

- Persist broker sessions, queues, leases, command states, results, rejection,
  and revocation using atomic private snapshots.
- Recover expired leases as queued after restart while preserving acknowledged
  running commands as uncertain and never replaying them automatically.
- Add pending-request listing and explicit rejection APIs.
- Add an optional dedicated Telegram approval bot with numeric approver
  allowlisting and one-tap approve/reject callbacks.
- Add a reserved priority lane for cancellation, owned-process stop, and
  disconnect commands so control traffic bypasses saturated ordinary work.
- Bound every approval to the duration originally requested by the guest.
- Add the typed `bridge-operator` CLI so agents do not construct raw
  administrator API calls or receive the administrator token as an argument.
- Add `.env.example` and `setup-operator.sh` to install the broker, CLI,
  systemd user service, and bundled OpenClaw skill in one operation.
- Stop persisting the one-time pairing token in plaintext.

## 0.5.2-mvp-dev

- Build releases in a clean staging directory and publish them only after every
  target and checksum verifies, preventing mixed or partially updated USB
  packages.
- Sign the public USB configuration and reject endpoint or USB identifier
  tampering before contacting the broker.
- Embed the launcher version, sign the minimum compatible launcher version,
  and reject mismatches between the launcher, manifest, and `VERSION.txt`.
- Generate one portable checksum inventory for the complete package, including
  documentation and configuration signatures.
- Refuse release builds when the configured private and public signing keys do
  not form the same Ed25519 key pair.
- Return explicit protocol negotiation details when a client and broker use
  incompatible wire versions.

## 0.5.1-mvp-dev

- Give the staged client and every child process a stable bridge-owned working
  directory so shells continue working when the USB directory disappears.
- Add idempotent command IDs and reject conflicting reuse of an existing ID.
- Add delivery leases and explicit client acknowledgement before execution;
  unacknowledged deliveries return to the queue.
- Correlate results with acknowledged running commands and reject duplicate or
  unsolicited results.
- Return structured queue depth, limit, and retry information.
- Distinguish cancellation requests from jobs that already completed.
- Configure explicit HTTP read, write, header, and idle timeouts.
- Retry transient long-poll failures with bounded exponential backoff instead
  of converting a brief disconnect into a client-initiated revocation.
- Advance the wire protocol to version 2; broker and guest must be upgraded
  together.

## 0.5.0-mvp-dev

- Translate the launcher and client console experience to English.
- Add signed launcher/client targets for Windows ARM64, Linux x64/ARM64, and
  macOS Intel/Apple Silicon.
- Add platform-aware capability profiles and native read-only inventory
  commands for Linux and macOS.
- Add an OpenClaw/Hermes-compatible operator skill and broker adapter reference.

## 0.4.2-mvp-dev

- Force UTF-8 console input, output, and pipeline encoding for Windows
  PowerShell 5.1 while retaining BOM-based script parsing.
- Preserve Unicode characters outside OEM code pages in captured output.
- Document the operator deployment path and prerequisite-free guest workflow.

## 0.4.1-mvp-dev

- Write structured PowerShell scripts with a UTF-8 BOM for Windows PowerShell
  5.1 compatibility.

## 0.4.0-mvp-dev

- Add Windows Job Object process-tree containment.
- Add structured PowerShell execution and CLIXML filtering.
- Add OEM, UTF-8, and UTF-16 output normalization.
- Add bounded chunked transfers and native ConPTY lifecycle primitives.
