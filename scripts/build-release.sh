#!/usr/bin/env bash
set -euo pipefail
project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
key_file="${BRIDGE_RELEASE_KEY_FILE:?set BRIDGE_RELEASE_KEY_FILE to an Ed25519 private key outside the repository}"
public_key="${BRIDGE_RELEASE_PUBLIC_KEY:?set BRIDGE_RELEASE_PUBLIC_KEY to the matching base64 public key}"
version="${1:-0.1.0-mvp-dev}"
image="golang:1.24-bookworm"
container_user="$(id -u):$(id -g)"

printf '%s\n' "$version" > "$project_dir/packaging/usb/VERSION.txt"

test -f "$key_file"
mkdir -p "$project_dir/bin" "$project_dir/packaging/usb/payload" "$project_dir/packaging/usb/launchers"
mkdir -p "$project_dir/packaging/usb/docs"
cp "$project_dir/docs/ARCHITECTURE.md" "$project_dir/packaging/usb/docs/ARCHITECTURE.md"
cp "$project_dir/docs/SECURITY.md" "$project_dir/packaging/usb/docs/SECURITY.md"
cp "$project_dir/docs/USAGE.md" "$project_dir/packaging/usb/docs/USAGE.md"
cp "$project_dir/docs/TROUBLESHOOTING.md" "$project_dir/packaging/usb/docs/TROUBLESHOOTING.md"
cp "$project_dir/docs/TEST_REPORT.md" "$project_dir/packaging/usb/docs/TEST_REPORT.md"
cp "$project_dir/docs/STATUS.md" "$project_dir/packaging/usb/docs/STATUS.md"
cp "$project_dir/CHANGELOG.md" "$project_dir/packaging/usb/docs/CHANGELOG.md"
targets=(windows-amd64 windows-arm64 linux-amd64 linux-arm64 darwin-amd64 darwin-arm64)

docker run --rm --user "$container_user" -e GOCACHE=/tmp/go-cache -v "$project_dir:/src" -w /src "$image" sh -c \
  'export PATH=/usr/local/go/bin:$PATH; go test -buildvcs=false ./...'

for target in "${targets[@]}"; do
  target_os="${target%-*}"
  target_arch="${target#*-}"
  extension=""
  if [[ "$target_os" == windows ]]; then extension=".exe"; fi
  client="bridge-client${extension}"
  launcher="OPENCLAW BRIDGE${extension}"
  mkdir -p "$project_dir/packaging/usb/payload/$target" "$project_dir/packaging/usb/launchers/$target"
  docker run --rm --user "$container_user" -e GOCACHE=/tmp/go-cache -v "$project_dir:/src" -w /src "$image" sh -c \
    "export PATH=/usr/local/go/bin:\$PATH; CGO_ENABLED=0 GOOS=$target_os GOARCH=$target_arch go build -buildvcs=false -trimpath -ldflags='-s -w' -o 'bin/bridge-client-$target$extension' ./cmd/bridge-client; CGO_ENABLED=0 GOOS=$target_os GOARCH=$target_arch go build -buildvcs=false -trimpath -ldflags='-s -w -X main.releasePublicKey=$public_key' -o 'packaging/usb/launchers/$target/$launcher' ./cmd/usb-launcher"
  docker run --rm --user "$container_user" -e GOCACHE=/tmp/go-cache -v "$project_dir:/src" -v "$(dirname "$key_file"):/keys:ro" -w /src "$image" sh -c \
    "export PATH=/usr/local/go/bin:\$PATH; go run -buildvcs=false ./cmd/release-tool -key /keys/$(basename "$key_file") -payload 'bin/bridge-client-$target$extension' -out 'packaging/usb/payload/$target' -version '$version' -target-os '$target_os' -target-arch '$target_arch' -filename '$client'"
done

cp "$project_dir/packaging/usb/launchers/windows-amd64/OPENCLAW BRIDGE.exe" "$project_dir/packaging/usb/OPENCLAW BRIDGE.exe"
(cd "$project_dir/packaging/usb" && find launchers payload -type f -print0 | sort -z | xargs -0 sha256sum; sha256sum "OPENCLAW BRIDGE.exe") > "$project_dir/packaging/usb/SHA256SUMS.txt"
