#!/usr/bin/env bash
set -euo pipefail
umask 077

config_root="${XDG_CONFIG_HOME:-$HOME/.config}/openclaw-portable-bridge"
signing_dir="$config_root/signing"
key_file="$signing_dir/release.key"
public_key_file="$signing_dir/release.pub"
recipient="${1:-${BRIDGE_BACKUP_AGE_RECIPIENT:-}}"
output="${2:-$signing_dir/release-key-backup.age}"

[[ -s "$key_file" && -s "$public_key_file" ]] || {
  printf 'release key pair is missing under %s\n' "$signing_dir" >&2
  exit 2
}
command -v age >/dev/null || {
  printf 'age is required for encrypted key backup: https://age-encryption.org\n' >&2
  exit 2
}
[[ -n "$recipient" ]] || {
  printf 'Usage: %s AGE_RECIPIENT [OUTPUT_FILE]\n' "$0" >&2
  exit 2
}

archive="$(mktemp)"
trap 'rm -f -- "$archive"' EXIT
tar -C "$signing_dir" -cf "$archive" release.key release.pub
age -r "$recipient" -o "$output.tmp" "$archive"
mv -f -- "$output.tmp" "$output"
chmod 600 "$output"
printf 'Encrypted release-key backup written to %s\n' "$output"
printf 'Verify recovery on a separate temporary directory before relying on it.\n'
