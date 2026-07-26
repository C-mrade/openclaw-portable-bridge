#!/usr/bin/env bash
set -euo pipefail
umask 022

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_root="${XDG_CONFIG_HOME:-$HOME/.config}"
signing_dir="${BRIDGE_RELEASE_SIGNING_DIR:-$config_root/openclaw-portable-bridge/signing}"
key_file="${BRIDGE_RELEASE_KEY_FILE:-$signing_dir/release.key}"
public_key_file="${BRIDGE_RELEASE_PUBLIC_KEY_FILE:-$signing_dir/release.pub}"
if [[ -n "${BRIDGE_RELEASE_PUBLIC_KEY:-}" ]]; then
  public_key="$BRIDGE_RELEASE_PUBLIC_KEY"
elif [[ -f "$public_key_file" ]]; then
  public_key="$(tr -d '\r\n' < "$public_key_file")"
else
  printf 'missing release public key: set BRIDGE_RELEASE_PUBLIC_KEY or create %s\n' \
    "$public_key_file" >&2
  exit 2
fi
version="${1:-0.1.0-mvp-dev}"
image="${BRIDGE_BUILD_IMAGE:-golang:1.24-bookworm}"
container_user="$(id -u):$(id -g)"
package_dir="$project_dir/packaging/usb"
staging_dir="$(mktemp -d "$project_dir/packaging/.usb-staging.XXXXXX")"
staging_rel="${staging_dir#"$project_dir/"}"

cleanup() {
  rm -f -- "$project_dir"/cmd/usb-launcher/rsrc_windows_*.syso
  if [[ -d "$staging_dir" ]]; then
    rm -rf -- "$staging_dir"
  fi
}
trap cleanup EXIT

if [[ ! "$version" =~ ^[0-9]+(\.[0-9]+){1,3}([.-][0-9A-Za-z]+)*$ ]]; then
  printf 'invalid release version: %s\n' "$version" >&2
  exit 2
fi
if [[ ! "$public_key" =~ ^[A-Za-z0-9+/]+$ ]]; then
  printf 'invalid base64 release public key\n' >&2
  exit 2
fi
if [[ ! -f "$key_file" ]]; then
  printf 'missing release private key: %s\n' "$key_file" >&2
  exit 2
fi

docker run --rm --user "$container_user" \
  -e GOCACHE=/tmp/go-cache \
  -v "$project_dir:/src" \
  -v "$(dirname "$key_file"):/keys:ro" \
  -w /src "$image" \
  sh -c "export PATH=/usr/local/go/bin:\$PATH
go run -buildvcs=false ./cmd/release-tool \
  -mode check-keypair \
  -key '/keys/$(basename "$key_file")' \
  -public-key '$public_key'"

mkdir -p \
  "$staging_dir/icons" \
  "$staging_dir/bin" \
  "$staging_dir/config" \
  "$staging_dir/docs" \
  "$staging_dir/launchers" \
  "$staging_dir/payload"
printf '%s\n' "$version" > "$staging_dir/VERSION.txt"
cp "$package_dir/README.md" "$staging_dir/README.md"
cp "$project_dir/packaging/START-HERE.txt" "$staging_dir/START HERE.txt"
docker run --rm --user "$container_user" \
  -e GOCACHE=/tmp/go-cache \
  -v "$project_dir:/src" -w /src "$image" \
  sh -c "export PATH=/usr/local/go/bin:\$PATH
go run -buildvcs=false ./scripts/generate-platform-icons.go '$staging_rel/icons'"

# Generate architecture-specific Windows resources before cross-compilation.
# The pinned generator is a build-time tool only.
docker run --rm --user "$container_user" \
  -e GOCACHE=/tmp/go-cache \
  -v "$project_dir:/src" -w /src "$image" \
  sh -c "export PATH=/usr/local/go/bin:\$PATH
go run github.com/tc-hib/go-winres@v0.3.3 simply \
  --arch amd64,arm64 \
  --out cmd/usb-launcher/rsrc \
  --icon '$staging_rel/icons/windows.png' \
  --manifest cli \
  --file-description 'OpenClaw Portable Bridge' \
  --product-name 'OpenClaw Portable Bridge' \
  --original-filename 'OPENCLAW BRIDGE.exe'"
cp "$package_dir/config/bridge-public.example.json" "$staging_dir/config/bridge-public.example.json"
if [[ -f "$package_dir/config/bridge-public.json" ]]; then
  cp "$package_dir/config/bridge-public.json" "$staging_dir/config/bridge-public.json"
else
  printf 'missing %s\n' "$package_dir/config/bridge-public.json" >&2
  exit 2
fi

docs=(
  AGENT_INTEGRATION.md
  ARCHITECTURE.md
  SECURITY.md
  STATUS.md
  TEST_REPORT.md
  THREAT_MODEL.md
  TROUBLESHOOTING.md
  USAGE.md
)
for doc in "${docs[@]}"; do
  cp "$project_dir/docs/$doc" "$staging_dir/docs/$doc"
done
cp "$project_dir/CHANGELOG.md" "$staging_dir/docs/CHANGELOG.md"

targets=(windows-amd64 windows-arm64 linux-amd64 linux-arm64 darwin-amd64 darwin-arm64)

docker run --rm --user "$container_user" \
  -e GOCACHE=/tmp/go-cache \
  -v "$project_dir:/src" -w /src "$image" \
  sh -c 'export PATH=/usr/local/go/bin:$PATH; go test -buildvcs=false ./...'

docker run --rm --user "$container_user" \
  -e GOCACHE=/tmp/go-cache \
  -v "$project_dir:/src" \
  -v "$(dirname "$key_file"):/keys:ro" \
  -w /src "$image" \
  sh -c "export PATH=/usr/local/go/bin:\$PATH
go run -buildvcs=false ./cmd/release-tool \
  -mode sign-file \
  -key '/keys/$(basename "$key_file")' \
  -input '$staging_rel/config/bridge-public.json' \
  -signature '$staging_rel/config/bridge-public.json.sig'"

for target in "${targets[@]}"; do
  target_os="${target%-*}"
  target_arch="${target#*-}"
  extension=""
  if [[ "$target_os" == windows ]]; then extension=".exe"; fi
  client="bridge-client${extension}"
  launcher="OPENCLAW BRIDGE${extension}"
  mkdir -p "$staging_dir/payload/$target" "$staging_dir/launchers/$target"

  docker run --rm --user "$container_user" \
    -e GOCACHE=/tmp/go-cache \
    -v "$project_dir:/src" -w /src "$image" \
    sh -c "export PATH=/usr/local/go/bin:\$PATH
CGO_ENABLED=0 GOOS=$target_os GOARCH=$target_arch go build -buildvcs=false -trimpath -ldflags='-s -w' -o '$staging_rel/bin/bridge-client-$target$extension' ./cmd/bridge-client
CGO_ENABLED=0 GOOS=$target_os GOARCH=$target_arch go build -buildvcs=false -trimpath -ldflags='-s -w -X main.releasePublicKey=$public_key -X main.launcherVersion=$version' -o '$staging_rel/launchers/$target/$launcher' ./cmd/usb-launcher"

  docker run --rm --user "$container_user" \
    -e GOCACHE=/tmp/go-cache \
    -v "$project_dir:/src" \
    -v "$(dirname "$key_file"):/keys:ro" \
    -w /src "$image" \
    sh -c "export PATH=/usr/local/go/bin:\$PATH
go run -buildvcs=false ./cmd/release-tool \
  -key '/keys/$(basename "$key_file")' \
  -payload '$staging_rel/bin/bridge-client-$target$extension' \
  -out '$staging_rel/payload/$target' \
  -version '$version' \
  -launcher-version '$version' \
  -target-os '$target_os' \
  -target-arch '$target_arch' \
  -filename '$client'"
done

cp "$staging_dir/launchers/windows-amd64/OPENCLAW BRIDGE.exe" "$staging_dir/OPENCLAW BRIDGE - WINDOWS.exe"
cp "$staging_dir/launchers/linux-amd64/OPENCLAW BRIDGE" "$staging_dir/OPENCLAW BRIDGE - LINUX.exe"
cp "$staging_dir/launchers/darwin-arm64/OPENCLAW BRIDGE" "$staging_dir/OPENCLAW BRIDGE - MACOS.command"
rm -f -- "$project_dir"/cmd/usb-launcher/rsrc_windows_*.syso
rm -rf -- "$staging_dir/bin"

(
  cd "$staging_dir"
  find . -type f ! -name SHA256SUMS.txt -print0 |
    LC_ALL=C sort -z |
    xargs -0 sha256sum
) > "$staging_dir/SHA256SUMS.txt"
(
  cd "$staging_dir"
  sha256sum -c SHA256SUMS.txt
)

previous_dir="$project_dir/packaging/.usb-previous"
rm -rf -- "$previous_dir"
mv -- "$package_dir" "$previous_dir"
if ! mv -- "$staging_dir" "$package_dir"; then
  mv -- "$previous_dir" "$package_dir"
  exit 1
fi
staging_dir=""
rm -rf -- "$previous_dir"
trap - EXIT

printf 'release %s built and verified at %s\n' "$version" "$package_dir"
