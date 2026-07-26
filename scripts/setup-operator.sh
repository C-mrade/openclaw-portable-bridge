#!/usr/bin/env bash
set -euo pipefail
umask 077

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
env_file="${1:-$project_dir/.env}"
config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/openclaw-portable-bridge"
state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/openclaw-portable-bridge"
install_dir="${XDG_DATA_HOME:-$HOME/.local/share}/openclaw-portable-bridge"
bin_dir="$HOME/.local/bin"
unit_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
openclaw_workspace="${OPENCLAW_WORKSPACE:-$HOME/.openclaw/workspace}"

if [[ ! -f "$env_file" ]]; then
  printf 'missing %s; copy .env.example to .env and configure it first\n' "$env_file" >&2
  exit 2
fi

read_env() {
  local key="$1"
  local line
  line="$(grep -E "^${key}=" "$env_file" | tail -n 1 || true)"
  printf '%s' "${line#*=}"
}

admin_token="$(read_env BRIDGE_ADMIN_TOKEN)"
broker_url="$(read_env BRIDGE_BROKER_URL)"
approval_target="$(read_env BRIDGE_APPROVAL_TARGET)"
approver_ids="$(read_env BRIDGE_APPROVER_IDS)"
gateway_env_file="$(read_env BRIDGE_GATEWAY_ENV_FILE)"

if [[ -z "$admin_token" ]]; then
  if ! command -v openssl >/dev/null 2>&1; then
    printf 'BRIDGE_ADMIN_TOKEN is empty and openssl is unavailable for safe generation\n' >&2
    exit 2
  fi
  admin_token="$(openssl rand -hex 32)"
  printf 'Generated a new broker administrator token.\n'
fi
if [[ ${#admin_token} -lt 24 || ! "$admin_token" =~ ^[A-Za-z0-9._~-]+$ ]]; then
  printf 'BRIDGE_ADMIN_TOKEN must contain at least 24 safe URL characters\n' >&2
  exit 2
fi
if [[ -z "$broker_url" ]]; then
  broker_url="http://127.0.0.1:17443"
fi
if [[ "$broker_url" != "http://127.0.0.1:17443" && "$broker_url" != https://* ]]; then
  printf 'BRIDGE_BROKER_URL must be loopback HTTP or HTTPS\n' >&2
  exit 2
fi
mkdir -p "$config_dir" "$state_dir" "$install_dir" "$bin_dir" "$unit_dir" "$openclaw_workspace/skills"
build_dir="$(mktemp -d)"
trap 'rm -rf -- "$build_dir"' EXIT

if command -v go >/dev/null 2>&1; then
  (
    cd "$project_dir"
    CGO_ENABLED=0 go build -buildvcs=false -trimpath -o "$build_dir/pairing-broker" ./cmd/pairing-broker
    CGO_ENABLED=0 go build -buildvcs=false -trimpath -o "$build_dir/bridge-operator" ./cmd/bridge-operator
    CGO_ENABLED=0 go build -buildvcs=false -trimpath -o "$build_dir/bridge-mcp" ./cmd/bridge-mcp
  )
elif command -v docker >/dev/null 2>&1; then
  docker run --rm --user "$(id -u):$(id -g)" \
    -e GOCACHE=/tmp/go-cache \
    -v "$project_dir:/src:ro" -v "$build_dir:/out" -w /src golang:1.24-bookworm \
    sh -c 'CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /out/pairing-broker ./cmd/pairing-broker &&
           CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /out/bridge-operator ./cmd/bridge-operator &&
           CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /out/bridge-mcp ./cmd/bridge-mcp'
else
  printf 'operator setup requires Go 1.24+ or Docker; guest machines require neither\n' >&2
  exit 2
fi

broker_restart_required=0
if [[ -f "$install_dir/pairing-broker" ]] &&
   ! cmp -s "$build_dir/pairing-broker" "$install_dir/pairing-broker"; then
  broker_restart_required=1
fi
if [[ -f "$unit_dir/openclaw-portable-bridge-broker.service" ]] &&
   ! cmp -s "$project_dir/packaging/systemd/openclaw-portable-bridge-broker.service.example" \
     "$unit_dir/openclaw-portable-bridge-broker.service"; then
  broker_restart_required=1
fi

install -m 0755 "$build_dir/pairing-broker" "$install_dir/pairing-broker"
install -m 0755 "$build_dir/bridge-operator" "$install_dir/bridge-operator"
install -m 0755 "$build_dir/bridge-mcp" "$install_dir/bridge-mcp"
install -m 0755 "$project_dir/scripts/openclaw-approval-notifier.sh" \
  "$install_dir/openclaw-approval-notifier"
ln -sfn "$install_dir/bridge-operator" "$bin_dir/bridge-operator"
ln -sfn "$install_dir/bridge-mcp" "$bin_dir/bridge-mcp"

{
  printf 'BRIDGE_ADMIN_TOKEN=%s\n' "$admin_token"
  printf 'BRIDGE_BROKER_URL=%s\n' "$broker_url"
} > "$config_dir/broker.env"
chmod 0600 "$config_dir/broker.env"

install -m 0644 "$project_dir/packaging/systemd/openclaw-portable-bridge-broker.service.example" \
  "$unit_dir/openclaw-portable-bridge-broker.service"
rm -rf -- "$openclaw_workspace/skills/openclaw-portable-bridge"
cp -R "$project_dir/skills/openclaw-portable-bridge" "$openclaw_workspace/skills/openclaw-portable-bridge"

if [[ ${BRIDGE_SETUP_NO_START:-0} != 1 ]]; then
  systemctl --user daemon-reload
  if systemctl --user is-active --quiet openclaw-portable-bridge-broker.service; then
    if [[ "$broker_restart_required" == 1 ]]; then
      systemctl --user restart openclaw-portable-bridge-broker.service
    fi
  else
    systemctl --user enable --now openclaw-portable-bridge-broker.service
  fi
  systemctl --user --no-pager --full status openclaw-portable-bridge-broker.service
fi

if [[ ${BRIDGE_SETUP_NO_REGISTER:-0} != 1 ]]; then
  "$project_dir/scripts/register-agent.sh" \
    "${BRIDGE_AGENT_TARGET:-auto}" "$install_dir/bridge-mcp"
fi

if [[ -n "$approval_target" || -n "$approver_ids" ]]; then
  if [[ ! "$approval_target" =~ ^[0-9]+$ ]]; then
    printf 'BRIDGE_APPROVAL_TARGET must be a numeric private Telegram chat ID\n' >&2
    exit 2
  fi
  if [[ ! "$approver_ids" =~ ^[0-9]+(,[0-9]+)*$ ]]; then
    printf 'BRIDGE_APPROVER_IDS must contain comma-separated numeric Telegram sender IDs\n' >&2
    exit 2
  fi
  if [[ -z "$gateway_env_file" ]]; then
    gateway_env_file="$HOME/.openclaw/openclaw.env"
  fi
  if [[ ! "$gateway_env_file" =~ ^[A-Za-z0-9_./-]+$ ]]; then
    printf 'BRIDGE_GATEWAY_ENV_FILE contains unsupported characters\n' >&2
    exit 2
  fi

  approver_json="$(jq -cn --arg ids "$approver_ids" '$ids | split(",")')"
  plugin_config="$(jq -cn \
    --argjson approvers "$approver_json" \
    --arg conversation "$approval_target" \
    --arg operator "$bin_dir/bridge-operator" \
    '{allowedApproverIds:$approvers,allowedConversationIds:[$conversation],operatorPath:$operator}')"

  openclaw plugins install --force \
    "$project_dir/integrations/openclaw-approval-plugin"
  openclaw config set 'plugins.entries["portable-bridge-approval"].config' \
    "$plugin_config" --strict-json

  sed \
    -e "s|TELEGRAM_CHAT_ID|$approval_target|" \
    -e "s|OPENCLAW_GATEWAY_ENV_FILE|$gateway_env_file|" \
    "$project_dir/packaging/systemd/openclaw-portable-bridge-notifier.service.example" \
    > "$unit_dir/openclaw-portable-bridge-notifier.service"
  install -m 0644 \
    "$project_dir/packaging/systemd/openclaw-portable-bridge-notifier.timer.example" \
    "$unit_dir/openclaw-portable-bridge-notifier.timer"

  if [[ ${BRIDGE_SETUP_NO_START:-0} != 1 ]]; then
    systemctl --user daemon-reload
    systemctl --user enable --now openclaw-portable-bridge-notifier.timer
  fi
  printf 'Installed proactive approval notifier and direct callback plugin.\n'
  printf 'Restart the OpenClaw Gateway after setup to activate the callback plugin.\n'
fi

printf '\nInstalled broker, bridge-operator, bridge-mcp, and agent skill.\n'
printf 'Run: bridge-operator pending\n'
