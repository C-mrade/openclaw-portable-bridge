#!/usr/bin/env bash
set -euo pipefail
project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
key_file="${BRIDGE_RELEASE_KEY_FILE:?set BRIDGE_RELEASE_KEY_FILE to an Ed25519 private key outside the repository}"
public_key="${BRIDGE_RELEASE_PUBLIC_KEY:?set BRIDGE_RELEASE_PUBLIC_KEY to the matching base64 public key}"
version="${1:-0.1.0-mvp-dev}"
image="golang:1.24-bookworm"

test -f "$key_file"
mkdir -p "$project_dir/bin" "$project_dir/packaging/usb/payload/windows-amd64"
mkdir -p "$project_dir/packaging/usb/docs"
cp "$project_dir/docs/ARCHITECTURE.md" "$project_dir/packaging/usb/docs/ARCHITECTURE.md"
cp "$project_dir/docs/SECURITY.md" "$project_dir/packaging/usb/docs/SECURITY.md"
cp "$project_dir/docs/USAGE.md" "$project_dir/packaging/usb/docs/USAGE.md"
cp "$project_dir/docs/TROUBLESHOOTING.md" "$project_dir/packaging/usb/docs/TROUBLESHOOTING.md"
cp "$project_dir/docs/TEST_REPORT.md" "$project_dir/packaging/usb/docs/TEST_REPORT.md"
cp "$project_dir/docs/STATUS.md" "$project_dir/packaging/usb/docs/STATUS.md"
cp "$project_dir/CHANGELOG.md" "$project_dir/packaging/usb/docs/CHANGELOG.md"
docker run --rm -v "$project_dir:/src" -w /src "$image" sh -c \
  'export PATH=/usr/local/go/bin:$PATH; go test -buildvcs=false ./...; CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags="-s -w" -o bin/bridge-client-windows-amd64.exe ./cmd/bridge-client'
docker run --rm -v "$project_dir:/src" -v "$(dirname "$key_file"):/keys:ro" -w /src "$image" sh -c \
  "export PATH=/usr/local/go/bin:\$PATH; go run -buildvcs=false ./cmd/release-tool -key /keys/$(basename "$key_file") -payload bin/bridge-client-windows-amd64.exe -out packaging/usb/payload/windows-amd64 -version '$version'"
docker run --rm -v "$project_dir:/src" -w /src "$image" sh -c \
  "export PATH=/usr/local/go/bin:\$PATH; CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags='-s -w -X main.releasePublicKey=$public_key' -o 'packaging/usb/OPENCLAW BRIDGE.exe' ./cmd/usb-launcher"
(cd "$project_dir/packaging/usb" && sha256sum "OPENCLAW BRIDGE.exe" payload/windows-amd64/* > SHA256SUMS.txt)
