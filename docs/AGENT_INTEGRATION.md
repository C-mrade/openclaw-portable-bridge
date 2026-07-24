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

The agent's existing Telegram, Discord, CLI, or web conversation is the
approval surface. The Bridge does not own or poll a messaging bot. This avoids
duplicate bots, `getUpdates` conflicts, and transport-specific credentials in
the broker.

The repository includes an agent skill at
`skills/openclaw-portable-bridge/SKILL.md`. OpenClaw, Hermes, Codex, or another
skill-compatible agent can install or reference that directory.
`scripts/setup-operator.sh` installs the skill, the typed `bridge-operator`
fallback CLI, and the dependency-free `bridge-mcp` executable. Both executables
read the protected local environment.

## MCP registration

Register the installed executable as a local stdio MCP server:

```json
{
  "mcpServers": {
    "openclaw_portable_bridge": {
      "command": "/home/USER/.local/share/openclaw-portable-bridge/bridge-mcp"
    }
  }
}
```

OpenClaw and Hermes may store MCP registration in different configuration
files; their installer integration should write the platform-native entry
rather than requiring a second messaging bot.

## Fallback transport

Pending listing, approval, rejection, commands, result consumption, and
revocation remain available through the restricted typed CLI for diagnostics.
Neither MCP nor the CLI is a generic HTTP proxy.
