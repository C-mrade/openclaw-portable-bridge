# Operator quickstart

## Agent-first setup

The normal path is to ask a trusted OpenClaw or Hermes agent to install and
configure the Bridge. The agent starts with:

```sh
git clone https://github.com/C-mrade/openclaw-portable-bridge.git
cd openclaw-portable-bridge
./scripts/bridge-bootstrap.sh discover
./scripts/bridge-bootstrap.sh plan --publisher tailscale-funnel --agent auto
```

Both commands are read-only and return JSON. The plan identifies missing
prerequisites and explicit human consent gates. After the human approves HTTPS
publication, the agent may execute the exact plan:

```sh
./scripts/bridge-bootstrap.sh apply --publisher tailscale-funnel \
  --agent auto --approve-publication
```

For an already provisioned reverse proxy or tunnel, use `--publisher existing
--public-url https://bridge.example.com`. Trusting a new endpoint also requires
the publication approval flag. `status` combines the persisted non-secret
bootstrap state with machine-readable diagnostics:

```sh
./scripts/bridge-bootstrap.sh status
```

An identical successful `apply` is a no-op. Agents can therefore safely resume
after context loss without rotating keys or rebuilding the deployment.

## Guided human fallback

The shortest safe path from a fresh clone is:

```sh
git clone https://github.com/C-mrade/openclaw-portable-bridge.git
cd openclaw-portable-bridge
./scripts/install.sh
```

The wizard validates prerequisites, generates the administrator token and
deployment-specific Ed25519 trust root, configures OpenClaw or Hermes, builds
the signed multi-platform package, verifies every checksum, and runs
`bridge-doctor`. To prepare a mounted USB in the same run:

```sh
./scripts/install.sh --usb-dir /media/USER/USB
```

For automation, all required choices can be supplied explicitly:

```sh
./scripts/install.sh --non-interactive \
  --public-url https://bridge.example.com \
  --agent openclaw \
  --approval-target 123456789 \
  --approver-ids 123456789
```

The public HTTPS endpoint must already route to the loopback broker through a
hardened reverse proxy or tunnel. The installer deliberately does not bind the
broker publicly or create a third-party relay account.

## Existing configuration path

The shortest supported operator path is:

```sh
cp .env.example .env
# Configure BRIDGE_ADMIN_TOKEN and BRIDGE_BROKER_URL.
./scripts/setup-operator.sh
bridge-operator pending
```

The setup installs the broker, typed CLI, standalone MCP adapter, user service,
and Bridge skill. It
uses an existing Go toolchain or Docker only on the operator machine. Portable
guest machines have no dependency requirement.

Register `bridge-mcp` with the existing OpenClaw or Hermes instance. Approval
then appears through that agent's existing private conversation; the Bridge
does not require or poll a separate Telegram bot. The typed
`bridge-operator` CLI remains available for diagnostics and recovery.

This guide produces a signed portable package and a loopback-only broker. The
guest machine remains prerequisite-free. The guided installer requires Linux,
OpenSSL, jq, curl, and Docker on the operator host. Go is optional because the
installer can use the pinned Go container for builds.

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
mkdir -p "$HOME/.config/openclaw-portable-bridge/signing"
chmod 700 "$HOME/.config/openclaw-portable-bridge/signing"
go run ./cmd/release-tool -mode keygen \
  -key "$HOME/.config/openclaw-portable-bridge/signing/release.key" \
  > "$HOME/.config/openclaw-portable-bridge/signing/release.pub"
openssl rand -base64 32 > "$HOME/.config/openclaw-portable-bridge/admin-token"
chmod 600 "$HOME/.config/openclaw-portable-bridge/signing/"*
```

Never copy `release.key` or `admin-token` to Git, the USB drive, or the guest.
Create and test an encrypted backup as described in
[Release signing operations](RELEASE_SIGNING.md).

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
cp packaging/usb/config/bridge-public.example.json \
   packaging/usb/config/bridge-public.json
```

Edit `bridge-public.json` and set your public HTTPS broker URL and a non-secret
USB identifier, then build:

```sh
./scripts/build-release.sh 0.6.0-beta.1
```

The build signs the public configuration, assembles every target in a clean
staging directory, and verifies the complete checksum inventory before
publishing it. Copy `packaging/usb/` to an `OPENCLAW_BRIDGE` directory on the
portable drive.

## 5. Acceptance checks

- Verify `SHA256SUMS.txt` after copying.
- Confirm a modified configuration, manifest, or payload is rejected.
- Start with the Information profile and a short session.
- Confirm revocation stops command delivery.
- Confirm the launcher's temporary directory is removed after exit.
- Inspect broker and client audit logs for secrets before distribution.

Next, read [Deployment](DEPLOYMENT.md), [Security operations](SECURITY.md), and
the [Threat model](THREAT_MODEL.md). The typed operator CLI is included; a
the native stdio MCP adapter is included.
