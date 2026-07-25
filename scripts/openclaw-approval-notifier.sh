#!/usr/bin/env bash
set -euo pipefail

bridge_operator="${BRIDGE_OPERATOR:-bridge-operator}"
openclaw_cli="${OPENCLAW_CLI:-openclaw}"
target="${BRIDGE_APPROVAL_TARGET:?BRIDGE_APPROVAL_TARGET is required}"
channel="${BRIDGE_APPROVAL_CHANNEL:-telegram}"
state_dir="${BRIDGE_APPROVAL_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/openclaw-portable-bridge/notifier}"
gateway_token_env="${BRIDGE_GATEWAY_TOKEN_ENV:-OPENCLAW_GATEWAY_AUTH_TOKEN_SECRET}"
gateway_env_file="${BRIDGE_GATEWAY_ENV_FILE:-}"

if [[ -z "${OPENCLAW_GATEWAY_TOKEN:-}" && -n "$gateway_env_file" ]]; then
    # shellcheck disable=SC1090
    source "$gateway_env_file"
fi

if [[ -z "${OPENCLAW_GATEWAY_TOKEN:-}" && -n "${!gateway_token_env:-}" ]]; then
    export OPENCLAW_GATEWAY_TOKEN="${!gateway_token_env}"
fi

command -v "$bridge_operator" >/dev/null
command -v "$openclaw_cli" >/dev/null
command -v jq >/dev/null

umask 077
mkdir -p "$state_dir"

pending="$("$bridge_operator" pending)"
jq -c '.[]' <<<"$pending" | while IFS= read -r request; do
    request_id="$(jq -r '.requestId' <<<"$request")"
    marker="$state_dir/$request_id.sent"
    [[ -e "$marker" ]] && continue

    hostname="$(jq -r '.hostname // "unknown"' <<<"$request")"
    os="$(jq -r '.os // "unknown"' <<<"$request")"
    arch="$(jq -r '.arch // "unknown"' <<<"$request")"
    user="$(jq -r '.user // "unknown"' <<<"$request")"
    code="$(jq -r '.compareCode' <<<"$request")"
    duration_seconds="$(jq -r '.durationSeconds' <<<"$request")"
    minutes="$((duration_seconds / 60))"
    capabilities="$(jq -r '.requested | join(", ")' <<<"$request")"

    message="🔐 NUOVA RICHIESTA OPENCLAW PORTABLE BRIDGE

Host dichiarato: ${hostname}
Sistema: ${os}/${arch}
Utente dichiarato: ${user}
Durata richiesta: ${minutes} min
Codice di confronto: ${code}

Capacità richieste:
${capabilities}

Approva soltanto se il codice coincide con quello mostrato sul desktop."

    presentation="$(jq -cn \
        --arg approve "bridge:approve:${request_id}:${code}" \
        --arg reject "bridge:reject:${request_id}:${code}" \
        '{blocks:[{type:"buttons",buttons:[
          {label:"Approva",value:$approve,style:"success"},
          {label:"Rifiuta",value:$reject,style:"danger"}
        ]}]}')"

    "$openclaw_cli" message send \
        --channel "$channel" \
        --target "$target" \
        --message "$message" \
        --presentation "$presentation" \
        --json >/dev/null

    : >"$marker"
done
