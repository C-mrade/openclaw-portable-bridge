# OpenClaw and Hermes agent integration

The recommended integration is a server-side adapter that turns the loopback
broker administration API into typed agent tools. The agent must never receive
or persist the raw broker administrator token.

## Recommended tool surface

- `bridge_list_pending`
- `bridge_approve`
- `bridge_reject`
- `bridge_command`
- `bridge_results`
- `bridge_revoke`

Bind approvals to the originating private conversation, show the comparison
code and exact capability list, enforce maximum durations, validate typed
command parameters, and redact credentials from every log. Keep destructive or
elevated work subject to the agent platform's normal approval policy.

The repository includes an agent skill at
`skills/openclaw-portable-bridge/SKILL.md`. OpenClaw, Hermes, Codex, or another
skill-compatible agent can install or reference that directory.
`scripts/setup-operator.sh` installs both the skill and the typed
`bridge-operator` CLI. The CLI reads its protected local environment, so the
agent does not need the administrator token in its prompt or command line.

## Current transport

Pending listing, approval, rejection, commands, result consumption, and
revocation are first-class broker operations. The current agent transport is
the restricted typed CLI. A native MCP adapter should preserve the same tool
surface and keep the token server-side; it must not become a generic HTTP proxy.
