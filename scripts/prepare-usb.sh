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
backup_root="${BRIDGE_USB_BACKUP_ROOT:-${XDG_STATE_HOME:-$HOME/.local/state}/openclaw-portable-bridge/usb-backups}"

archive_from_usb() {
  local source="$1"
  local target="$backup_root/$(basename "$source")"
  [[ -e "$source" ]] || return 0
  if [[ -e "$target" ]]; then
    target="$target-$(date -u +%Y%m%dT%H%M%SZ)"
  fi
  mkdir -p "$backup_root"
  cp -a -- "$source" "$target"
  if [[ -d "$source" ]]; then
    diff -qr -- "$source" "$target" >/dev/null
  else
    cmp -s -- "$source" "$target"
  fi
  if [[ -d "$target" && -s "$target/SHA256SUMS.txt" ]] &&
     ! (cd "$target" && sha256sum -c SHA256SUMS.txt >/dev/null 2>&1); then
    printf 'WARNING: archived legacy package had pre-existing checksum mismatches: %s\n' "$source" >&2
  fi
  rm -rf -- "$source"
  printf 'Archived off the USB: %s -> %s\n' "$source" "$target"
}

rm -rf -- "$stage"
mkdir -p "$stage"
cp -a "$source_dir/." "$stage/"
(cd "$stage" && sha256sum -c SHA256SUMS.txt)

if [[ -e "$current" ]]; then
  archive_from_usb "$current"
fi
mv -- "$stage" "$current"
(cd "$current" && sha256sum -c SHA256SUMS.txt >/dev/null)

# Normalize only known Bridge-owned legacy entries. Unrelated user files are
# never touched.
shopt -s nullglob
for legacy in \
  "$destination_root"/OPENCLAW_BRIDGE-backup-* \
  "$destination_root"/OPENCLAW_BRIDGE.previous-* \
  "$destination_root"/OPENCLAW_BRIDGE.stage \
  "$destination_root"/OPENCLAW\ BRIDGE\ -\ CURRENT.lnk; do
  archive_from_usb "$legacy"
done
shopt -u nullglob

printf 'Prepared and verified: %s\n' "$current"
printf 'Historical Bridge packages are preserved under: %s\n' "$backup_root"
