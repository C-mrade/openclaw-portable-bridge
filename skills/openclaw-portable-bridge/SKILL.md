---
name: openclaw-portable-bridge
description: Operate, integrate, test, or troubleshoot OpenClaw Portable Bridge sessions from OpenClaw, Hermes, Codex, or another trusted operator agent. Use for pairing approval, capability validation, bounded remote commands, file transfer, revocation, audit review, or deployment diagnostics involving this repository and its broker API.
---

# OpenClaw Portable Bridge

Treat every session as temporary delegated access to another machine.

## Workflow

1. Verify the broker is the operator-controlled local instance. Keep its admin
   API on loopback or behind a trusted server-side adapter.
2. Obtain the request ID and six-character comparison code from trusted local
   output. Compare the code with the guest before approval.
3. Inspect requested capabilities, descriptive host data, and requested
   duration. Treat hostname and username as untrusted labels.
4. Approve only the profile and duration the user requested. Never add a
   capability after pairing.
5. Queue commands with unique IDs, explicit deadlines, and the narrowest
   suitable capability. Prefer fixed inspection capabilities over shell.
6. Consume results, report errors accurately, and avoid logging secrets or
   unnecessary file contents.
7. Revoke the session when the task finishes, consent changes, the comparison
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

## Platform selection

- Windows: full current capability set, including structured PowerShell and
  per-command UAC.
- Linux/macOS: information, user-level shell, process, and scoped file
  capabilities. No remote privilege elevation or ConPTY.

Read [references/broker-api.md](references/broker-api.md) when implementing an
adapter or directly mapping agent tools to broker endpoints. Read the root
`docs/THREAT_MODEL.md` before changing exposure, authentication, or capability
policy.
