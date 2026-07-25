#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_dir="$project_dir/packaging/usb"
destination_root="${1:-}"

[[ -n "$destination_root" ]] || {
  printf 'Usage: %s USB_MOUNT_DIRECTORY\n' "$0" >&2
  exit 2
}
[[ -d "$destination_root" ]] || {
  printf 'USB mount directory does not exist: %s\n' "$destination_root" >&2
  exit 2
}
destination_root="$(cd "$destination_root" && pwd -P)"
case "$destination_root" in
  /|/home|"$HOME"|/usr|/etc|/var) printf 'refusing unsafe destination: %s\n' "$destination_root" >&2; exit 2 ;;
esac
[[ -s "$source_dir/SHA256SUMS.txt" ]] || {
  printf 'build the signed release before preparing the USB\n' >&2
  exit 2
}

stage="$destination_root/OPENCLAW_BRIDGE.stage"
current="$destination_root/OPENCLAW_BRIDGE"
backup="$destination_root/OPENCLAW_BRIDGE.backup-$(date -u +%Y%m%dT%H%M%SZ)"
rm -rf -- "$stage"
mkdir -p "$stage"
cp -a "$source_dir/." "$stage/"
(cd "$stage" && sha256sum -c SHA256SUMS.txt)

if [[ -e "$current" ]]; then
  mv -- "$current" "$backup"
fi
mv -- "$stage" "$current"
(cd "$current" && sha256sum -c SHA256SUMS.txt >/dev/null)

printf 'Prepared and verified: %s\n' "$current"
if [[ -e "$backup" ]]; then
  printf 'Previous package preserved: %s\n' "$backup"
fi
