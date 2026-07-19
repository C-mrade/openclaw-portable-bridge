# Operator quickstart

This guide produces a signed portable package and a loopback-only broker. The
guest machine remains prerequisite-free. Commands assume Linux on the operator
host, Go 1.24+, Docker, OpenSSL, and an HTTPS publishing method.

## 1. Clone and verify

```sh
git clone https://github.com/C-mrade/openclaw-portable-bridge.git
cd openclaw-portable-bridge
go test ./...
```

## 2. Create deployment secrets

Keep both files outside the repository and outside the portable package.

```sh
mkdir -p "$HOME/.config/openclaw-portable-bridge"
go run ./cmd/release-tool -mode keygen \
  -key "$HOME/.config/openclaw-portable-bridge/release.key"
openssl rand -base64 32 > "$HOME/.config/openclaw-portable-bridge/admin-token"
chmod 600 "$HOME/.config/openclaw-portable-bridge/"*
```

Record the public key printed by `release-tool`. Never copy `release.key` or
`admin-token` to Git, the USB drive, or the guest.

## 3. Run the broker on loopback

```sh
go build -o bin/pairing-broker ./cmd/pairing-broker
BRIDGE_ADMIN_TOKEN="$(tr -d '\r\n' < "$HOME/.config/openclaw-portable-bridge/admin-token")" \
  ./bin/pairing-broker -listen 127.0.0.1:17443 -audit ./broker-audit.jsonl
```

Publish only `127.0.0.1:17443` through a dedicated HTTPS endpoint. Do not bind
the broker directly to a public interface. Confirm the endpoint is reachable
from outside the operator network before continuing.

## 4. Build the signed portable package

```sh
export BRIDGE_RELEASE_KEY_FILE="$HOME/.config/openclaw-portable-bridge/release.key"
export BRIDGE_RELEASE_PUBLIC_KEY='<public-key-printed-in-step-2>'
./scripts/build-release.sh 0.5.0-mvp-dev
cp packaging/usb/config/bridge-public.example.json \
   packaging/usb/config/bridge-public.json
```

Edit `bridge-public.json` and set your public HTTPS broker URL and a non-secret
USB identifier. Copy `packaging/usb/` to an `OPENCLAW_BRIDGE` directory on the
portable drive.

## 5. Acceptance checks

- Verify `SHA256SUMS.txt` after copying.
- Confirm a modified manifest or payload is rejected.
- Start with the Information profile and a short session.
- Confirm revocation stops command delivery.
- Confirm the launcher's temporary directory is removed after exit.
- Inspect broker and client audit logs for secrets before distribution.

Next, read [Deployment](DEPLOYMENT.md), [Security operations](SECURITY.md), and
the [Threat model](THREAT_MODEL.md). The operator-side approval CLI/MCP adapter
is planned but not part of release 0.5.0.
