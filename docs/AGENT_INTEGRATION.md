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
skill-compatible agent can install or reference that directory. The skill
defines the safe operating workflow and links to an endpoint/parameter
reference only when adapter work requires it.

## Current limitation

The broker MVP still lacks first-class pending-list and reject endpoints. Add
those before presenting the adapter as a complete approval interface. Until
then, derive pending notifications only from a trusted local event stream and
use revocation/expiry conservatively.
