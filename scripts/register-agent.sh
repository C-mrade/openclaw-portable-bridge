#!/usr/bin/env bash
set -euo pipefail

server_name="openclaw_portable_bridge"
target="${1:-${BRIDGE_AGENT_TARGET:-auto}}"
bridge_mcp="${2:-${XDG_DATA_HOME:-$HOME/.local/share}/openclaw-portable-bridge/bridge-mcp}"

if [[ ! -x "$bridge_mcp" ]]; then
  printf 'bridge-mcp is not executable: %s\n' "$bridge_mcp" >&2
  exit 2
fi
if [[ "$bridge_mcp" == *'"'* || "$bridge_mcp" == *$'\n'* || "$bridge_mcp" == *$'\r'* ]]; then
  printf 'bridge-mcp path contains unsupported control or quote characters\n' >&2
  exit 2
fi

has_openclaw=0
has_hermes=0
command -v openclaw >/dev/null 2>&1 && has_openclaw=1
command -v hermes >/dev/null 2>&1 && has_hermes=1

case "$target" in
  auto)
    if (( has_openclaw )); then
      target="openclaw"
    elif (( has_hermes )); then
      target="hermes"
    else
      printf 'No supported agent CLI found; skipping MCP registration.\n'
      exit 0
    fi
    ;;
  openclaw|hermes|both|none) ;;
  *)
    printf 'BRIDGE_AGENT_TARGET must be auto, openclaw, hermes, both, or none\n' >&2
    exit 2
    ;;
esac

if [[ "$target" == "none" ]]; then
  printf 'Agent MCP registration skipped by configuration.\n'
  exit 0
fi

register_openclaw() {
  if (( ! has_openclaw )); then
    printf 'OpenClaw CLI not found\n' >&2
    return 1
  fi
  local value
  value="$(printf '{"command":"%s"}' "${bridge_mcp//\\/\\\\}")"
  openclaw mcp set "$server_name" "$value"
  openclaw mcp probe "$server_name" >/dev/null
  openclaw mcp reload >/dev/null
  printf 'Registered and probed Bridge MCP in OpenClaw.\n'
}

register_hermes() {
  if (( ! has_hermes )); then
    printf 'Hermes CLI not found\n' >&2
    return 1
  fi
  local add_output test_output
  # First answer accepts replacement when the entry exists; the second
  # enables all six fixed Bridge tools after discovery.
  if ! add_output="$(printf 'y\ny\n' | hermes mcp add "$server_name" --command "$bridge_mcp" 2>&1)"; then
    printf '%s\n' "$add_output" >&2
    return 1
  fi
  printf '%s\n' "$add_output"
  if grep -Eqi 'failed to connect|requires the .mcp. Python SDK|saved .*disabled' <<<"$add_output"; then
    printf 'Hermes did not enable the Bridge MCP server\n' >&2
    return 1
  fi
  if ! test_output="$(hermes mcp test "$server_name" 2>&1)"; then
    printf '%s\n' "$test_output" >&2
    return 1
  fi
  if grep -Eqi 'connection failed|requires the .mcp. Python SDK' <<<"$test_output"; then
    printf '%s\n' "$test_output" >&2
    return 1
  fi
  printf 'Registered and tested Bridge MCP in Hermes.\n'
}

case "$target" in
  openclaw) register_openclaw ;;
  hermes) register_hermes ;;
  both)
    register_openclaw
    register_hermes
    ;;
esac
