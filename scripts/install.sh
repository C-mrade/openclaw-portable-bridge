#!/usr/bin/env bash
set -euo pipefail
umask 077

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_root="${XDG_CONFIG_HOME:-$HOME/.config}/openclaw-portable-bridge"
signing_dir="$config_root/signing"
env_file="$project_dir/.env"
public_config="$project_dir/packaging/usb/config/bridge-public.json"
version="${BRIDGE_VERSION:-0.6.2-beta.1}"
broker_public_url="${BRIDGE_PUBLIC_URL:-}"
approval_target="${BRIDGE_APPROVAL_TARGET:-}"
approver_ids="${BRIDGE_APPROVER_IDS:-}"
agent_target="${BRIDGE_AGENT_TARGET:-auto}"
usb_dir=""
non_interactive=0

usage() {
  cat <<'EOF'
Usage: ./scripts/install.sh [options]

Guided, secure operator installation and signed USB package creation.

Options:
  --public-url HTTPS_URL       Public HTTPS URL used by guest clients
  --approval-target CHAT_ID    Private Telegram chat ID for approvals
  --approver-ids ID[,ID...]    Allowed Telegram sender IDs
  --agent auto|openclaw|hermes|both|none
  --usb-dir DIRECTORY          Prepare DIRECTORY/OPENCLAW_BRIDGE after build
  --version VERSION            Release version (default: 0.6.2-beta.1)
  --non-interactive            Fail instead of prompting for missing values
  -h, --help                   Show this help

The broker remains bound to loopback. This script never publishes it directly.
EOF
}

die() {
  printf 'install: %s\n' "$*" >&2
  exit 2
}

prompt() {
  local variable="$1" message="$2" default="${3:-}" value
  if [[ "$non_interactive" == 1 ]]; then
    [[ -n "${!variable:-}" ]] || die "$variable is required in non-interactive mode"
    return
  fi
  read -r -p "$message${default:+ [$default]}: " value
  printf -v "$variable" '%s' "${value:-$default}"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --public-url) broker_public_url="${2:-}"; shift 2 ;;
    --approval-target) approval_target="${2:-}"; shift 2 ;;
    --approver-ids) approver_ids="${2:-}"; shift 2 ;;
    --agent) agent_target="${2:-}"; shift 2 ;;
    --usb-dir) usb_dir="${2:-}"; shift 2 ;;
    --version) version="${2:-}"; shift 2 ;;
    --non-interactive) non_interactive=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown option: $1" ;;
  esac
done

[[ "$(uname -s)" == Linux ]] || die "the operator installer currently supports Linux"
for command_name in openssl jq curl; do
  command -v "$command_name" >/dev/null || die "missing dependency: $command_name"
done
command -v docker >/dev/null ||
  die "Docker is currently required to build the reproducible multi-platform package"
[[ "$agent_target" =~ ^(auto|openclaw|hermes|both|none)$ ]] ||
  die "--agent must be auto, openclaw, hermes, both, or none"

existing_env_value() {
  local key="$1" line
  [[ -f "$env_file" ]] || return 0
  line="$(grep -E "^${key}=" "$env_file" | tail -n 1 || true)"
  printf '%s' "${line#*=}"
}
if [[ -z "$broker_public_url" && -s "$public_config" ]]; then
  broker_public_url="$(jq -r '.brokerUrl // empty' "$public_config" 2>/dev/null || true)"
fi
if [[ -z "$approval_target" ]]; then
  approval_target="$(existing_env_value BRIDGE_APPROVAL_TARGET)"
fi
if [[ -z "$approver_ids" ]]; then
  approver_ids="$(existing_env_value BRIDGE_APPROVER_IDS)"
fi

if [[ -z "$broker_public_url" ]]; then
  prompt broker_public_url "Public HTTPS broker URL"
fi
[[ "$broker_public_url" =~ ^https://[^[:space:]]+$ ]] ||
  die "the guest broker URL must be HTTPS"

if [[ "$non_interactive" == 0 && -z "$approval_target" ]]; then
  read -r -p "Telegram private chat ID (blank to configure approvals later): " approval_target
fi
if [[ -n "$approval_target" && -z "$approver_ids" ]]; then
  if [[ "$non_interactive" == 1 ]]; then
    die "--approver-ids is required when --approval-target is set"
  fi
  prompt approver_ids "Allowed Telegram sender ID(s), comma-separated"
fi
[[ -z "$approval_target" || "$approval_target" =~ ^[0-9]+$ ]] ||
  die "Telegram chat ID must be numeric"
[[ -z "$approver_ids" || "$approver_ids" =~ ^[0-9]+(,[0-9]+)*$ ]] ||
  die "Telegram approver IDs must be comma-separated numbers"

mkdir -p "$signing_dir"
chmod 700 "$config_root" "$signing_dir"

admin_token_file="$config_root/admin-token"
if [[ ! -s "$admin_token_file" ]]; then
  openssl rand -hex 32 > "$admin_token_file"
fi
chmod 600 "$admin_token_file"

key_file="$signing_dir/release.key"
public_key_file="$signing_dir/release.pub"
if [[ -s "$key_file" && ! -s "$public_key_file" ]] ||
   [[ ! -s "$key_file" && -s "$public_key_file" ]]; then
  die "incomplete release key pair; recover the missing file instead of rotating trust implicitly"
fi
if [[ ! -s "$key_file" && ! -s "$public_key_file" ]]; then
  if command -v go >/dev/null; then
    (cd "$project_dir" && go run ./cmd/release-tool -mode keygen -key "$key_file") > "$public_key_file"
  else
    docker run --rm --user "$(id -u):$(id -g)" \
      -e GOCACHE=/tmp/go-cache \
      -v "$project_dir:/src:ro" -v "$signing_dir:/signing" -w /src \
      golang:1.24-bookworm \
      sh -c 'go run ./cmd/release-tool -mode keygen -key /signing/release.key' \
      > "$public_key_file"
  fi
fi
chmod 600 "$key_file" "$public_key_file"

gateway_env_file="${BRIDGE_GATEWAY_ENV_FILE:-}"
{
  printf 'BRIDGE_ADMIN_TOKEN=%s\n' "$(tr -d '\r\n' < "$admin_token_file")"
  printf 'BRIDGE_BROKER_URL=http://127.0.0.1:17443\n'
  printf 'BRIDGE_APPROVAL_TARGET=%s\n' "$approval_target"
  printf 'BRIDGE_APPROVER_IDS=%s\n' "$approver_ids"
  printf 'BRIDGE_GATEWAY_ENV_FILE=%s\n' "$gateway_env_file"
  printf 'BRIDGE_RELEASE_KEY_FILE=%s\n' "$key_file"
  printf 'BRIDGE_RELEASE_PUBLIC_KEY_FILE=%s\n' "$public_key_file"
} > "$env_file"
chmod 600 "$env_file"

usb_id=""
if [[ -s "$public_config" ]]; then
  usb_id="$(jq -r '.usbId // empty' "$public_config" 2>/dev/null || true)"
fi
if [[ -z "$usb_id" || "$usb_id" == "portable-bridge-example" ]]; then
  usb_id="bridge-$(openssl rand -hex 8)"
fi
jq -n --arg usb_id "$usb_id" --arg url "$broker_public_url" \
  '{usbId:$usb_id,brokerUrl:$url}' > "$public_config"

BRIDGE_AGENT_TARGET="$agent_target" "$project_dir/scripts/setup-operator.sh" "$env_file"
BRIDGE_RELEASE_KEY_FILE="$key_file" \
BRIDGE_RELEASE_PUBLIC_KEY_FILE="$public_key_file" \
  "$project_dir/scripts/build-release.sh" "$version"

if [[ -n "$usb_dir" ]]; then
  "$project_dir/scripts/prepare-usb.sh" "$usb_dir"
fi

if [[ ${BRIDGE_INSTALL_SKIP_DOCTOR:-0} != 1 ]]; then
  "$project_dir/scripts/bridge-doctor.sh" --public-url "$broker_public_url"
fi

cat <<EOF

Installation complete.
  Operator CLI:  $HOME/.local/bin/bridge-operator
  Signed package: $project_dir/packaging/usb
  Release key:   $key_file

Back up the release key now:
  $project_dir/scripts/backup-release-key.sh
EOF
