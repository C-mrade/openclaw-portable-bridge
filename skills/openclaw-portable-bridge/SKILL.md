---
name: openclaw-portable-bridge
description: Operate, integrate, test, or troubleshoot OpenClaw Portable Bridge sessions from OpenClaw, Hermes, Codex, or another trusted operator agent. Use for pairing approval, capability validation, bounded remote commands, file transfer, revocation, audit review, or deployment diagnostics involving this repository and its broker API.
---

# OpenClaw Portable Bridge

Treat every session as temporary delegated access to another machine.

## Workflow

### First-time agent setup

When the user asks you to install or configure the Bridge, use the repository's
machine-readable bootstrap interface instead of interviewing the user for
values one at a time:

```sh
./scripts/bridge-bootstrap.sh discover
./scripts/bridge-bootstrap.sh plan --publisher existing --public-url https://bridge.example.com
./scripts/bridge-bootstrap.sh apply ...approved plan arguments...
./scripts/bridge-bootstrap.sh status
```

`discover` and `plan` are read-only. Show every item in `consentGates` to the
human. Run `apply` with `--approve-publication` only after the human explicitly
approves trusting an existing HTTPS endpoint or enabling Tailscale Funnel.
Never infer that approval from the original installation request. The
bootstrap is idempotent and returns `changed:false` when an identical completed
plan is already installed. Do not read or reproduce generated secrets.
When approval conversation or sender identities are supplied, separately show
the `approval_identity_binding` gate and add `--approve-approvers` only after
the human confirms them.

Prefer `--publisher tailscale-funnel` when discovery reports a connected
Tailscale node and the human approves public publication. Otherwise use a
pre-existing hardened HTTPS endpoint. Do not bind the broker to a public
interface and do not invent a shared relay or trust root.

### Session operation

1. Use the installed `bridge-operator` command. Do not call the raw HTTP API or
   read `broker.env` unless troubleshooting the adapter itself.
2. Use `bridge_list_pending` through the local MCP adapter (or run
   `bridge-operator pending` for recovery) and compare the six-character code
   with the guest before approval. Bind approval to the existing agent's
   private conversation and require the human to verify the same code.
3. Inspect requested capabilities, descriptive host data, and requested
   duration. Treat hostname and username as untrusted labels.
4. Approve only the profile and duration the user requested. Never add a
   capability after pairing.
5. Resolve the original approval message in place after every callback: show
   `✅ APPROVED`, `❌ REJECTED`, or `⌛ EXPIRED`, include the authoritative
   expiry returned by the broker, and remove all inline buttons. A duplicate
   callback must remain a no-op and must never restore a resolved keyboard.
6. Queue commands with unique IDs, explicit deadlines, and the narrowest
   suitable capability. Prefer fixed inspection capabilities over shell.
7. Consume results, report errors accurately, and avoid logging secrets or
   unnecessary file contents.
8. Revoke the session when the task finishes, consent changes, the comparison
   code differs, or behavior is unexpected.

## Safety rules

- Never request, print, store, or commit the broker administrator token.
- Never approve a session solely from client-supplied hostname or username.
- Never bypass local consent, UAC, operating-system permissions, or endpoint
  verification.
- Ask before destructive, public, access-control, or security-sensitive work.
- Use `shell.run-admin` only on Windows and only when the guest expects a local
  UAC prompt. Linux and macOS builds intentionally expose no remote elevation.
- Preserve audit records and remove only Bridge-owned temporary files.
- Do not treat a successful HTTP response as proof a command succeeded; inspect
  the returned command result and exit code.
- Never approve a request whose duration exceeds what the guest expects.
- Treat a recovered `running` command as uncertain until its result arrives;
  never enqueue a replacement automatically.

## Operator commands

Prefer the equivalent `bridge_*` MCP tools when they are available. The CLI is
the recovery and diagnostics path.

```sh
bridge-operator pending
bridge-operator approve REQUEST_ID MINUTES
bridge-operator reject REQUEST_ID
bridge-operator command --id UNIQUE_ID --name system.info REQUEST_ID
bridge-operator results --consume REQUEST_ID
bridge-operator revoke REQUEST_ID
```

Use a JSON `--params` value only for the documented typed capability. Keep
timeouts short. The CLI reads its protected local configuration itself; never
pass the administrator token on the command line.

## Platform selection

- Windows: full current capability set, including structured PowerShell and
  per-command UAC.
- Linux/macOS: information, user-level shell, process, and scoped file
  capabilities. No remote privilege elevation or ConPTY.

Read [references/broker-api.md](references/broker-api.md) when implementing an
adapter or directly mapping agent tools to broker endpoints. Read the root
`docs/THREAT_MODEL.md` before changing exposure, authentication, or capability
policy.
