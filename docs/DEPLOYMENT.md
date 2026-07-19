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

Build the broker and generate an independent administrator token:

```sh
go build -o bin/pairing-broker ./cmd/pairing-broker
openssl rand -base64 32
```

Provide that token as `BRIDGE_ADMIN_TOKEN` through a protected environment
file and bind the broker to loopback:

```sh
BRIDGE_ADMIN_TOKEN='<random-admin-token>' \
  ./bin/pairing-broker -listen 127.0.0.1:17443 -audit ./broker-audit.jsonl
```

An example hardened user service is available at
`packaging/systemd/openclaw-portable-bridge-broker.service.example`.

## 3. HTTPS publication

Publish only `127.0.0.1:17443` through a dedicated HTTPS endpoint. Tailscale
Funnel or a hardened reverse proxy are suitable. Do not bind the broker
directly to a public interface and do not expose an OpenClaw Gateway token.

Verify the endpoint from a network outside the operator host before packaging.

## 4. Build the signed USB directory

```sh
export BRIDGE_RELEASE_KEY_FILE=/secure/path/release.key
export BRIDGE_RELEASE_PUBLIC_KEY='<public-key-from-keygen>'
./scripts/build-release.sh 0.4.2-mvp-dev
cp packaging/usb/config/bridge-public.example.json \
   packaging/usb/config/bridge-public.json
```

Edit `bridge-public.json` with the public HTTPS broker URL and a non-secret USB
identifier. Copy the entire generated `packaging/usb` directory to a dedicated
`OPENCLAW_BRIDGE` directory on the USB. Preserve unrelated files already on the
drive.

Before distribution, verify `SHA256SUMS.txt` and test that modifying either the
manifest or payload causes the launcher to refuse execution.

## 5. Guest operation

The Windows 10/11 x64 guest only needs to double-click
`OPENCLAW BRIDGE.exe`. The launcher verifies the signed payload, stages it in
`%TEMP%`, keeps consent visible, and removes only its own temporary directory
after exit. Normal operation does not install anything or require elevation.

## Current approval interface

The MVP exposes authenticated broker administration endpoints for an
operator-side adapter. A polished approval CLI, Telegram buttons, and the
official OpenClaw Node v4 adapter are not complete yet. Treat this repository
as a development MVP until those pieces and the remaining adversarial tests are
finished.
