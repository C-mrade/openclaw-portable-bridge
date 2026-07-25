# OpenClaw and Hermes agent integration

The included `bridge-mcp` adapter turns the loopback broker administration API
into typed agent tools. The agent never receives or persists the raw broker
administrator token: the standalone adapter reads the protected operator
configuration itself.

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

Treat an approval prompt as a small stateful UI, not as an immutable message.
After a callback succeeds, edit the original message in place, prefix it with
an unambiguous terminal state (`✅ APPROVED` or `❌ REJECTED`), include the
resolved duration and expiry when approved, and remove every action button.
Expired requests must likewise be marked `⌛ EXPIRED`. Duplicate callbacks are
idempotent and must not restore buttons or create a second approval. The
approval API returns `requestId`, `minutes`, and `expiresAt` so adapters can
render the authoritative broker state rather than estimating it locally.

The agent's existing Telegram, Discord, CLI, or web conversation is the
approval surface. The Bridge does not own or poll a messaging bot. This avoids
duplicate bots, `getUpdates` conflicts, and transport-specific credentials in
the broker.

For OpenClaw deployments, `scripts/openclaw-approval-notifier.sh` is the
optional proactive adapter. Run it from the example user timer with an
allowlisted private destination. It polls only the typed `bridge-operator`
surface, sends through OpenClaw's existing channel, and records a local marker
only after successful delivery. The broker still owns request validity and
expiry; the notifier owns neither credentials nor approval state.

The companion `portable-bridge-approval` OpenClaw plugin claims the `bridge:`
Telegram callback namespace before callbacks enter the agent/LLM queue. It
requires OpenClaw sender authorization plus explicit sender and private-chat
allowlists, re-reads the authoritative pending request, verifies request ID,
comparison code, and guest duration, and then invokes only the typed operator
CLI. Terminal outcomes edit the original message and clear every button.
Transient failures leave the buttons available for retry.

The repository includes an agent skill at
`skills/openclaw-portable-bridge/SKILL.md`. OpenClaw, Hermes, Codex, or another
skill-compatible agent can install or reference that directory.
`scripts/setup-operator.sh` installs the skill, the typed `bridge-operator`
fallback CLI, and the dependency-free `bridge-mcp` executable. Both executables
read the protected local environment.

## MCP registration

The standard setup registers and probes the installed executable automatically.
Select the desired agent with:

```sh
BRIDGE_AGENT_TARGET=openclaw ./scripts/setup-operator.sh
BRIDGE_AGENT_TARGET=hermes ./scripts/setup-operator.sh
BRIDGE_AGENT_TARGET=both ./scripts/setup-operator.sh
```

`auto` (the default) prefers OpenClaw and otherwise selects Hermes. Use
`BRIDGE_SETUP_NO_REGISTER=1` for an unsupported agent or a deliberately manual
installation. The equivalent local stdio entry is:

```json
{
  "mcpServers": {
    "openclaw_portable_bridge": {
      "command": "/home/USER/.local/share/openclaw-portable-bridge/bridge-mcp"
    }
  }
}
```

The installer uses each platform's native CLI and probes all six tools before
reporting success. It stores only the executable path in agent configuration;
the administrator token remains in the protected Bridge operator environment.
When `BRIDGE_APPROVAL_TARGET` and `BRIDGE_APPROVER_IDS` are configured, setup
also installs the proactive notifier, user timer, and direct callback plugin.
Restart the OpenClaw Gateway once after setup to activate a newly installed
plugin.

## Fallback transport

Pending listing, approval, rejection, commands, result consumption, and
revocation remain available through the restricted typed CLI for diagnostics.
Neither MCP nor the CLI is a generic HTTP proxy.
