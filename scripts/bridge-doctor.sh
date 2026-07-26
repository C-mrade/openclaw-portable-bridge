#!/usr/bin/env bash
set -u

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_root="${XDG_CONFIG_HOME:-$HOME/.config}/openclaw-portable-bridge"
public_url=""
failures=0
warnings=0
json=0
checks='[]'

while [[ $# -gt 0 ]]; do
  case "$1" in
    --public-url) public_url="${2:-}"; shift 2 ;;
    --json) json=1; shift ;;
    *)
      printf 'Usage: %s [--json] [--public-url HTTPS_URL]\n' "$0" >&2
      exit 2
      ;;
  esac
done

record() {
  local status="$1" message="$2"
  checks="$(jq -cn --argjson old "$checks" --arg status "$status" --arg message "$message" \
    '$old + [{status:$status,message:$message}]')"
  [[ "$json" == 1 ]] || printf '%-5s %s\n' "${status^^}" "$message"
}
pass() { record pass "$*"; }
warn() { record warn "$*"; warnings=$((warnings + 1)); }
fail() { record fail "$*"; failures=$((failures + 1)); }

for command_name in curl jq openssl; do
  command -v "$command_name" >/dev/null && pass "$command_name available" ||
    fail "$command_name missing"
done

[[ -x "$HOME/.local/bin/bridge-operator" ]] &&
  pass "bridge-operator installed" || fail "bridge-operator not installed"
[[ -x "$HOME/.local/bin/bridge-mcp" ]] &&
  pass "bridge-mcp installed" || fail "bridge-mcp not installed"
[[ -s "$config_root/broker.env" ]] &&
  pass "protected broker configuration exists" || fail "broker configuration missing"

key_file="$config_root/signing/release.key"
public_key_file="$config_root/signing/release.pub"
[[ -s "$key_file" && -s "$public_key_file" ]] &&
  pass "release key pair exists" || fail "release key pair missing"
if [[ -e "$key_file" ]]; then
  mode="$(stat -c '%a' "$key_file" 2>/dev/null || true)"
  [[ "$mode" == 600 ]] && pass "release private key mode is 0600" ||
    fail "release private key mode is ${mode:-unknown}, expected 600"
fi

if [[ ${BRIDGE_SETUP_NO_START:-0} == 1 ]]; then
  warn "service and live broker checks skipped by BRIDGE_SETUP_NO_START"
elif command -v systemctl >/dev/null; then
  systemctl --user is-active --quiet openclaw-portable-bridge-broker.service &&
    pass "broker service active" || fail "broker service inactive"
fi

if [[ -x "$HOME/.local/bin/bridge-operator" && ${BRIDGE_SETUP_NO_START:-0} != 1 ]]; then
  "$HOME/.local/bin/bridge-operator" pending >/dev/null 2>&1 &&
    pass "broker operator API reachable" || fail "broker operator API unreachable"
fi

checksums="$project_dir/packaging/usb/SHA256SUMS.txt"
if [[ -s "$checksums" ]]; then
  if (cd "$project_dir/packaging/usb" && sha256sum -c SHA256SUMS.txt >/dev/null 2>&1); then
    pass "signed portable package checksum inventory valid"
  else
    fail "portable package checksum verification failed"
  fi
else
  warn "portable package has not been built"
fi

if [[ -n "$public_url" ]]; then
  if [[ ! "$public_url" =~ ^https:// ]]; then
    fail "public guest endpoint is not HTTPS"
  elif curl --silent --show-error --output /dev/null --max-time 10 "$public_url/v1/pair/status" 2>/dev/null; then
    pass "public guest endpoint reachable"
  else
    warn "public endpoint not reachable at $public_url"
  fi
fi

if [[ "$json" == 1 ]]; then
  jq -cn \
    --argjson ok "$([[ "$failures" -eq 0 ]] && printf true || printf false)" \
    --argjson failures "$failures" \
    --argjson warnings "$warnings" \
    --argjson checks "$checks" \
    '{schemaVersion:1,ok:$ok,failures:$failures,warnings:$warnings,checks:$checks}'
else
  printf '\nDoctor summary: %d failure(s), %d warning(s)\n' "$failures" "$warnings"
fi
[[ "$failures" -eq 0 ]]
