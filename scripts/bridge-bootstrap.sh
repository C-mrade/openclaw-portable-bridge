#!/usr/bin/env bash
set -euo pipefail
umask 077

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_root="${XDG_CONFIG_HOME:-$HOME/.config}/openclaw-portable-bridge"
state_root="${XDG_STATE_HOME:-$HOME/.local/state}/openclaw-portable-bridge"
state_file="$state_root/bootstrap.json"
public_config="$project_dir/packaging/usb/config/bridge-public.json"

command_name="${1:-discover}"
[[ $# -eq 0 ]] || shift
public_url="${BRIDGE_PUBLIC_URL:-}"
publisher="${BRIDGE_PUBLISHER:-existing}"
agent_target="${BRIDGE_AGENT_TARGET:-auto}"
approval_target="${BRIDGE_APPROVAL_TARGET:-}"
approver_ids="${BRIDGE_APPROVER_IDS:-}"
usb_dir="${BRIDGE_USB_DIR:-}"
version="${BRIDGE_VERSION:-0.6.2-beta.2}"
approve_publication=0
approve_approvers=0
force=0

usage() {
  cat <<'EOF'
Usage: ./scripts/bridge-bootstrap.sh COMMAND [options]

Agent-facing, resumable setup control plane.

Commands:
  discover   Inspect the host and emit JSON without changing it
  plan       Emit the exact actions and consent gates as JSON
  apply      Execute an approved plan, persist state, and emit JSON
  status     Emit persisted state and machine-readable diagnostics

Options:
  --publisher existing|tailscale-funnel
  --public-url HTTPS_URL
  --agent auto|openclaw|hermes|both|none
  --approval-target CHAT_ID
  --approver-ids ID[,ID...]
  --usb-dir DIRECTORY
  --version VERSION
  --approve-publication   Confirm the human approved public HTTPS publication
  --approve-approvers     Confirm the human approved the conversation/identity binding
  --force                 Reapply an otherwise identical completed plan
  -h, --help

`apply` never treats an agent decision as consent to expose the broker.
The publication gate must be approved by the human and represented by
--approve-publication. The broker itself always remains on loopback.
EOF
}

die_json() {
  local code="$1" message="$2"
  jq -cn --arg error "$code" --arg message "$message" \
    '{ok:false,error:$error,message:$message}'
  exit 2
}

have() { command -v "$1" >/dev/null 2>&1; }
bool_json() { [[ "$1" == 1 ]] && printf true || printf false; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --publisher) publisher="${2:-}"; shift 2 ;;
    --public-url) public_url="${2:-}"; shift 2 ;;
    --agent) agent_target="${2:-}"; shift 2 ;;
    --approval-target) approval_target="${2:-}"; shift 2 ;;
    --approver-ids) approver_ids="${2:-}"; shift 2 ;;
    --usb-dir) usb_dir="${2:-}"; shift 2 ;;
    --version) version="${2:-}"; shift 2 ;;
    --approve-publication) approve_publication=1; shift ;;
    --approve-approvers) approve_approvers=1; shift ;;
    --force) force=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die_json invalid_argument "unknown option: $1" ;;
  esac
done

[[ "$command_name" =~ ^(discover|plan|apply|status)$ ]] ||
  die_json invalid_command "expected discover, plan, apply, or status"
[[ "$publisher" =~ ^(existing|tailscale-funnel)$ ]] ||
  die_json invalid_publisher "publisher must be existing or tailscale-funnel"
[[ "$agent_target" =~ ^(auto|openclaw|hermes|both|none)$ ]] ||
  die_json invalid_agent "agent must be auto, openclaw, hermes, both, or none"
[[ -z "$approval_target" || "$approval_target" =~ ^[0-9]+$ ]] ||
  die_json invalid_approval_target "approval target must be a numeric private conversation ID"
[[ -z "$approver_ids" || "$approver_ids" =~ ^[0-9]+(,[0-9]+)*$ ]] ||
  die_json invalid_approvers "approver IDs must be comma-separated numbers"

existing_public_url() {
  [[ -s "$public_config" ]] || return 0
  jq -r '.brokerUrl // empty' "$public_config" 2>/dev/null || true
}

tailscale_dns_name() {
  have tailscale || return 0
  tailscale status --json 2>/dev/null |
    jq -r '.Self.DNSName // empty' 2>/dev/null |
    sed 's/\.$//' || true
}

tailscale_funnel_port() {
  local funnel_status existing_port candidate
  have tailscale || return 0
  funnel_status="$(tailscale funnel status --json 2>/dev/null || true)"
  [[ -n "$funnel_status" ]] || return 0
  existing_port="$(jq -r '
    . as $root
    | .Web // {}
    | to_entries[]
    | select(.value.Handlers["/"].Proxy == "http://127.0.0.1:17443")
    | select($root.AllowFunnel[.key] == true)
    | (.key | capture(":(?<port>[0-9]+)$").port)
  ' <<<"$funnel_status" 2>/dev/null | head -n 1)"
  if [[ -n "$existing_port" ]]; then
    printf '%s' "$existing_port"
    return
  fi
  for candidate in 8443 10000 443; do
    if ! jq -e --arg port "$candidate" '(.TCP // {}) | has($port)' \
      <<<"$funnel_status" >/dev/null 2>&1; then
      printf '%s' "$candidate"
      return
    fi
  done
}

resolve_public_url() {
  if [[ -n "$public_url" ]]; then
    printf '%s' "$public_url"
  elif [[ "$publisher" == tailscale-funnel ]]; then
    local dns_name funnel_port
    dns_name="$(tailscale_dns_name)"
    funnel_port="$(tailscale_funnel_port)"
    if [[ -n "$dns_name" && -n "$funnel_port" ]]; then
      if [[ "$funnel_port" == 443 ]]; then
        printf 'https://%s' "$dns_name"
      else
        printf 'https://%s:%s' "$dns_name" "$funnel_port"
      fi
    fi
  else
    existing_public_url
  fi
}

discovery_json() {
  local current_url tailscale_dns tailscale_port
  current_url="$(existing_public_url)"
  tailscale_dns="$(tailscale_dns_name)"
  tailscale_port="$(tailscale_funnel_port)"
  jq -cn \
    --arg os "$(uname -s 2>/dev/null || printf unknown)" \
    --arg arch "$(uname -m 2>/dev/null || printf unknown)" \
    --arg currentUrl "$current_url" \
    --arg tailscaleDns "$tailscale_dns" \
    --arg tailscalePort "$tailscale_port" \
    --arg stateFile "$state_file" \
    --argjson go "$(bool_json "$(have go && printf 1 || printf 0)")" \
    --argjson docker "$(bool_json "$(have docker && printf 1 || printf 0)")" \
    --argjson openclaw "$(bool_json "$(have openclaw && printf 1 || printf 0)")" \
    --argjson hermes "$(bool_json "$(have hermes && printf 1 || printf 0)")" \
    --argjson tailscale "$(bool_json "$(have tailscale && printf 1 || printf 0)")" \
    --argjson jq "$(bool_json "$(have jq && printf 1 || printf 0)")" \
    --argjson curl "$(bool_json "$(have curl && printf 1 || printf 0)")" \
    --argjson openssl "$(bool_json "$(have openssl && printf 1 || printf 0)")" \
    --argjson configured "$(bool_json "$([[ -s "$state_file" ]] && printf 1 || printf 0)")" \
    '{
      schemaVersion:1,
      platform:{os:$os,arch:$arch},
      prerequisites:{go:$go,docker:$docker,jq:$jq,curl:$curl,openssl:$openssl},
      agents:{openclaw:$openclaw,hermes:$hermes},
      publishers:{
        existingEndpoint:($currentUrl | if length > 0 then . else null end),
        tailscale:{
          available:$tailscale,
          dnsName:($tailscaleDns | if length > 0 then . else null end),
          bridgeOrFreePort:($tailscalePort | if length > 0 then tonumber else null end)
        }
      },
      bootstrap:{configured:$configured,stateFile:$stateFile}
    }'
}

plan_json() {
  local resolved_url="$1" ready=1
  local missing='[]' gates='[]' actions

  for dependency in jq curl openssl; do
    if ! have "$dependency"; then
      missing="$(jq -cn --argjson old "$missing" --arg value "$dependency" '$old + [$value]')"
      ready=0
    fi
  done
  if ! have go && ! have docker; then
    missing="$(jq -cn --argjson old "$missing" '$old + ["go-or-docker"]')"
    ready=0
  fi
  if [[ "$publisher" == tailscale-funnel ]] && [[ -z "$(tailscale_dns_name)" ]]; then
    missing="$(jq -cn --argjson old "$missing" '$old + ["connected-tailscale"]')"
    ready=0
  fi
  if [[ "$publisher" == tailscale-funnel ]] && [[ -z "$(tailscale_funnel_port)" ]]; then
    missing="$(jq -cn --argjson old "$missing" '$old + ["available-tailscale-funnel-port"]')"
    ready=0
  fi
  if [[ -z "$resolved_url" ]]; then
    missing="$(jq -cn --argjson old "$missing" '$old + ["public-https-url"]')"
    ready=0
  elif [[ ! "$resolved_url" =~ ^https://[^[:space:]]+$ ]]; then
    missing="$(jq -cn --argjson old "$missing" '$old + ["valid-public-https-url"]')"
    ready=0
  fi
  if [[ "$publisher" == tailscale-funnel ]]; then
    gates='[{
      "id":"public_https_publication",
      "required":true,
      "reason":"Tailscale Funnel makes the loopback broker reachable through public HTTPS.",
      "approvalFlag":"--approve-publication"
    }]'
  elif [[ -n "$resolved_url" && "$resolved_url" != "$(existing_public_url)" ]]; then
    gates='[{
      "id":"trust_existing_https_endpoint",
      "required":true,
      "reason":"The signed guest package will trust this HTTPS endpoint.",
      "approvalFlag":"--approve-publication"
    }]'
  fi
  if [[ -n "$approval_target" || -n "$approver_ids" ]]; then
    gates="$(jq -cn --argjson old "$gates" \
      '$old + [{
        id:"approval_identity_binding",
        required:true,
        reason:"These identities can approve temporary Bridge sessions from the configured private conversation.",
        approvalFlag:"--approve-approvers"
      }]')"
  fi
  actions="$(jq -cn \
    --arg publisher "$publisher" \
    --arg url "$resolved_url" \
    --arg agent "$agent_target" \
    --arg usb "$usb_dir" \
    '[
      (if $publisher == "tailscale-funnel" then
        {id:"publish_https",description:"Publish loopback broker with Tailscale Funnel",mutatesExternalState:true}
       else empty end),
      {id:"generate_secrets",description:"Create or reuse protected administrator token and Ed25519 trust root",mutatesExternalState:false},
      {id:"install_operator",description:("Install broker, MCP adapter, skill, and register " + $agent),mutatesExternalState:false},
      {id:"build_package",description:("Build and sign portable package for " + $url),mutatesExternalState:false},
      (if $usb != "" then {id:"prepare_usb",description:("Atomically prepare " + $usb),mutatesExternalState:false} else empty end),
      {id:"verify",description:"Run machine-readable diagnostics",mutatesExternalState:false}
    ]')"
  jq -cn \
    --arg publisher "$publisher" \
    --arg url "$resolved_url" \
    --arg agent "$agent_target" \
    --arg version "$version" \
    --argjson approvalConfigured "$([[ -n "$approval_target" ]] && printf true || printf false)" \
    --argjson approverCount "$([[ -z "$approver_ids" ]] && printf 0 || tr ',' '\n' <<<"$approver_ids" | wc -l)" \
    --argjson ready "$(bool_json "$ready")" \
    --argjson missing "$missing" \
    --argjson gates "$gates" \
    --argjson actions "$actions" \
    '{
      schemaVersion:1,
      ready:$ready,
      inputs:{
        publisher:$publisher,
        publicUrl:($url | if length > 0 then . else null end),
        agent:$agent,
        version:$version,
        approvalConversationConfigured:$approvalConfigured,
        approverCount:$approverCount
      },
      missing:$missing,
      consentGates:$gates,
      actions:$actions
    }'
}

if [[ "$command_name" == discover ]]; then
  discovery_json
  exit 0
fi

if [[ "$command_name" == status ]]; then
  doctor_output="$("$project_dir/scripts/bridge-doctor.sh" --json 2>/dev/null || true)"
  [[ -n "$doctor_output" ]] || doctor_output='{"ok":false,"failures":1,"warnings":0,"checks":[]}'
  if [[ -s "$state_file" ]]; then
    jq -cn --slurpfile state "$state_file" --argjson doctor "$doctor_output" \
      '{schemaVersion:1,configured:true,state:$state[0],doctor:$doctor}'
  else
    jq -cn --argjson doctor "$doctor_output" \
      '{schemaVersion:1,configured:false,state:null,doctor:$doctor}'
  fi
  exit 0
fi

resolved_url="$(resolve_public_url)"
plan="$(plan_json "$resolved_url")"
if [[ "$command_name" == plan ]]; then
  printf '%s\n' "$plan"
  exit 0
fi

[[ "$(jq -r '.ready' <<<"$plan")" == true ]] ||
  die_json plan_not_ready "run plan and resolve every missing prerequisite before apply"
if jq -e '.consentGates[] | select(.id == "public_https_publication" or .id == "trust_existing_https_endpoint")' \
  <<<"$plan" >/dev/null && [[ "$approve_publication" != 1 ]]; then
  die_json consent_required "human endpoint/publication approval is required; rerun with --approve-publication only after approval"
fi
if jq -e '.consentGates[] | select(.id == "approval_identity_binding")' \
  <<<"$plan" >/dev/null && [[ "$approve_approvers" != 1 ]]; then
  die_json consent_required "human approval of the approver identities is required; rerun with --approve-approvers only after approval"
fi

plan_fingerprint="$(
  jq -cn -cS \
    --argjson plan "$plan" \
    --arg approvalTarget "$approval_target" \
    --arg approverIds "$approver_ids" \
    --arg usbDir "$usb_dir" \
    '{inputs:$plan.inputs,actions:$plan.actions,approvalTarget:$approvalTarget,approverIds:$approverIds,usbDir:$usbDir}' |
    sha256sum | cut -d' ' -f1
)"
if [[ "$force" != 1 && -s "$state_file" ]] &&
   [[ "$(jq -r '.planFingerprint // empty' "$state_file")" == "$plan_fingerprint" ]] &&
   [[ "$(jq -r '.status // empty' "$state_file")" == complete ]]; then
  jq -cn --arg fingerprint "$plan_fingerprint" --slurpfile state "$state_file" \
    '{ok:true,changed:false,resumed:true,planFingerprint:$fingerprint,state:$state[0]}'
  exit 0
fi

mkdir -p "$state_root"
if [[ "$publisher" == tailscale-funnel ]]; then
  funnel_port="$(tailscale_funnel_port)"
  tailscale funnel --yes --bg --https="$funnel_port" 17443 >/dev/null
fi

install_args=(
  --non-interactive
  --public-url "$resolved_url"
  --agent "$agent_target"
  --version "$version"
)
[[ -z "$approval_target" ]] || install_args+=(--approval-target "$approval_target")
[[ -z "$approver_ids" ]] || install_args+=(--approver-ids "$approver_ids")
[[ -z "$usb_dir" ]] || install_args+=(--usb-dir "$usb_dir")
"$project_dir/scripts/install.sh" "${install_args[@]}" >/dev/null

doctor_output="$("$project_dir/scripts/bridge-doctor.sh" --json --public-url "$resolved_url" || true)"
if [[ "$(jq -r '.ok' <<<"$doctor_output")" != true ]]; then
  jq -cn --argjson doctor "$doctor_output" \
    '{ok:false,error:"verification_failed",message:"installation completed but verification failed",doctor:$doctor}'
  exit 1
fi

completed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
state_tmp="$(mktemp "$state_root/bootstrap.json.XXXXXX")"
jq -cn \
  --arg status complete \
  --arg completedAt "$completed_at" \
  --arg fingerprint "$plan_fingerprint" \
  --arg publisher "$publisher" \
  --arg url "$resolved_url" \
  --arg agent "$agent_target" \
  --arg version "$version" \
  '{
    schemaVersion:1,status:$status,completedAt:$completedAt,
    planFingerprint:$fingerprint,publisher:$publisher,publicUrl:$url,
    agent:$agent,version:$version
  }' > "$state_tmp"
chmod 600 "$state_tmp"
mv -f "$state_tmp" "$state_file"

jq -cn \
  --arg fingerprint "$plan_fingerprint" \
  --argjson doctor "$doctor_output" \
  --slurpfile state "$state_file" \
  '{ok:true,changed:true,resumed:false,planFingerprint:$fingerprint,state:$state[0],doctor:$doctor}'
