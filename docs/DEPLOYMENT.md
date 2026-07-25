# Deployment and portable packaging

This guide separates one-time operator setup from the prerequisite-free guest
experience. Commands are examples: use your own hostname, paths, keys, and
secrets.

## 1. Operator prerequisites

The build/server machine needs Go 1.24 or Docker, an HTTPS publishing method,
and a safe location for an Ed25519 release key. None of these are required on
the Windows guest.

Generate the release key outside the checkout:

```sh
go run ./cmd/release-tool -mode keygen -key /secure/path/release.key
```

Store the printed public key. Keep the private key mode `0600`, back it up
securely, and never copy it to Git or USB.

## 2. Broker

For the standard agent installation, copy `.env.example` to `.env`, set the
administrator token and broker URL, then run:

```sh
./scripts/setup-operator.sh
```

This installs the broker, typed operator CLI, standalone MCP adapter, hardened
user service, and bundled skill. Register `bridge-mcp` with the existing
OpenClaw or Hermes instance so approval uses its existing private conversation.
No dedicated Telegram bot is required.

For OpenClaw-native proactive Telegram approval, also configure the numeric
private chat and approver IDs:

```sh
BRIDGE_APPROVAL_TARGET=123456789
BRIDGE_APPROVER_IDS=123456789
BRIDGE_GATEWAY_ENV_FILE=/home/USER/.openclaw/openclaw.env
```

The setup then installs a 10-second notifier timer and a callback plugin. The
plugin resolves approve/reject callbacks before the agent queue, binds them to
the configured private conversation, and edits the original message in place.
Restart the OpenClaw Gateway once after installation.

Build the broker and generate an independent administrator token:

```sh
go build -o bin/pairing-broker ./cmd/pairing-broker
openssl rand -base64 32
```

Provide that token as `BRIDGE_ADMIN_TOKEN` through a protected environment
file and bind the broker to loopback:

```sh
BRIDGE_ADMIN_TOKEN='<random-admin-token>' \
  ./bin/pairing-broker -listen 127.0.0.1:17443 \
    -audit ./broker-audit.jsonl -state ./broker-state.json
```

An example hardened user service is available at
`packaging/systemd/openclaw-portable-bridge-broker.service.example`.
The private state snapshot is published atomically with mode `0600`. Back it up
consistently with the audit log, never expose it through HTTPS, and treat a
corrupt or unsupported state file as a fail-closed startup error.

## 3. HTTPS publication

Publish only `127.0.0.1:17443` through a dedicated HTTPS endpoint. Tailscale
Funnel or a hardened reverse proxy are suitable. Do not bind the broker
directly to a public interface and do not expose an OpenClaw Gateway token.

Verify the endpoint from a network outside the operator host before packaging.

## 4. Build the signed portable directory

```sh
export BRIDGE_RELEASE_KEY_FILE=/secure/path/release.key
export BRIDGE_RELEASE_PUBLIC_KEY='<public-key-from-keygen>'
cp packaging/usb/config/bridge-public.example.json \
   packaging/usb/config/bridge-public.json
```

Edit `bridge-public.json` with the public HTTPS broker URL and a non-secret USB
identifier, then run:

```sh
./scripts/build-release.sh 0.6.0-alpha.1
```

The build signs that configuration and refuses to publish a partial release.
Changing it later requires rebuilding or explicitly re-signing it with the
release key. Copy the entire generated `packaging/usb` directory to a dedicated
`OPENCLAW_BRIDGE` directory on the USB. Preserve unrelated files already on the
drive.

Before distribution, verify `SHA256SUMS.txt` and test that modifying the public
configuration, manifest, or payload causes the launcher to refuse execution.

## 5. Guest operation

The build creates signed payloads and launchers for Windows amd64/arm64, Linux
amd64/arm64, and macOS amd64/arm64. Select the launcher under
`packaging/usb/launchers/<os>-<arch>` and keep the shared `payload` and `config`
directories beside the launcher directory structure. The root Windows x64
launcher is retained for the simplest USB experience.

The launcher verifies the matching signed payload, stages it under the native
temporary directory, keeps consent visible, and removes only its own temporary
directory after exit. Normal operation does not install anything or require
elevation. Linux/macOS removable media may be mounted `noexec`; in that case,
copy the signed portable directory to a user-owned local folder before running
it. Do not weaken mount security globally.

## Current approval interface

The current development release includes automatic MCP registration, proactive
OpenClaw Telegram notifications, and a direct callback handler with explicit
sender/chat binding. Hermes and other agents retain the typed MCP/CLI path but
do not yet have the OpenClaw-specific direct Telegram handler. Treat this
repository as a development release until the remaining adversarial tests and
platform-signing gates are complete.

For a short, end-to-end path with validation checkpoints, use
[QUICKSTART.md](QUICKSTART.md) first.
