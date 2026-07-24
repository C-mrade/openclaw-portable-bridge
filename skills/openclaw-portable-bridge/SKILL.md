---
name: openclaw-portable-bridge
description: Operate, integrate, test, or troubleshoot OpenClaw Portable Bridge sessions from OpenClaw, Hermes, Codex, or another trusted operator agent. Use for pairing approval, capability validation, bounded remote commands, file transfer, revocation, audit review, or deployment diagnostics involving this repository and its broker API.
---

# OpenClaw Portable Bridge

Treat every session as temporary delegated access to another machine.

## Workflow

1. Use the installed `bridge-operator` command. Do not call the raw HTTP API or
   read `broker.env` unless troubleshooting the adapter itself.
2. Run `bridge-operator pending` and compare the six-character code with the
   guest before approval. Telegram approval is equivalent only when the
   configured, allowlisted approver verified the same code.
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
- Never approve a request whose duration exceeds what the guest expects.
- Treat a recovered `running` command as uncertain until its result arrives;
  never enqueue a replacement automatically.

## Operator commands

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
