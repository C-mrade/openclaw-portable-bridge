# OpenClaw Portable Bridge

Experimental, security-first bridge for connecting a Windows 10/11 x64 guest
to an operator-controlled broker without installing Python, Node.js, Docker,
Git, Tailscale, or OpenClaw on the guest.

> **Guest experience:** copy the prepared `OPENCLAW_BRIDGE` directory to a USB
> drive, double-click `OPENCLAW BRIDGE.exe`, choose a profile, and approve the
> pairing. The guest needs no installation, runtime, service, administrator
> setup, or Tailscale client. The operator performs the one-time broker,
> endpoint, and signing-key setup described in
> [Deployment](docs/DEPLOYMENT.md).

The project currently provides:

- a standalone Go launcher that verifies an Ed25519-signed manifest and
  payload before staging the client under `%TEMP%`;
- a visible, consent-driven portable client with ephemeral session identity;
- an HTTP broker designed to remain on loopback behind a TLS reverse proxy or
  Tailscale Funnel;
- explicit pairing approval, short-lived tokens, revocation, replay
  protection, rate limiting, capability profiles, scoped file access, and
  application audit logs;
- cancellable asynchronous shell jobs contained in Windows Job Objects,
  low-latency long polling, paginated directory listings, bounded resumable
  transfers with SHA-256, structured PowerShell execution, OEM/UTF-16 output
  decoding, and CLIXML filtering;
- an optional administrator command path that always invokes the normal local
  Windows UAC prompt.

It does not install services on the guest, persist after exit, bypass Windows
protections, or expose an OpenClaw Gateway token. Administrator execution is
never automatic: Developer mode must be confirmed locally and each elevated
command requires a separate UAC approval.

## Status

This is an MVP, not a production remote-management product. The current UI is
a visible console. Native ConPTY primitives are present, but the persistent
terminal protocol and UI are not complete. A native GUI, WebSocket streaming,
Telegram approval buttons,
the official OpenClaw Node v4 adapter, Authenticode signing, and the complete
adversarial test matrix remain future work. Review
[docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) before exposing a broker publicly.

## Quick start for operators

There are two distinct environments:

1. **Operator/server:** builds and signs the package, runs the loopback broker,
   and publishes only that broker through HTTPS.
2. **Windows guest:** runs the finished USB package with no prerequisites.

For a reproducible self-hosted setup, follow [Deployment](docs/DEPLOYMENT.md).
For normal guest use, follow [Usage](docs/USAGE.md).

## Build and test

Go 1.24 or newer is recommended.

```sh
go test ./...
go build ./cmd/pairing-broker
go build ./cmd/bridge-client
```

Create an Ed25519 release key outside the repository, then export its path and
public key before building the Windows package:

```sh
go run ./cmd/release-tool -mode keygen -key /secure/path/release.key
export BRIDGE_RELEASE_KEY_FILE=/secure/path/release.key
export BRIDGE_RELEASE_PUBLIC_KEY='<public-key-printed-by-keygen>'
./scripts/build-release.sh 0.4.2-mvp-dev
```

Copy `packaging/usb/config/bridge-public.example.json` to
`packaging/usb/config/bridge-public.json` and replace the placeholder values
for your own deployment. Generated binaries, signatures, manifests, local
configuration, logs, and release keys are intentionally ignored by Git.

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Threat model](docs/THREAT_MODEL.md)
- [Security operations](docs/SECURITY.md)
- [Deployment and packaging](docs/DEPLOYMENT.md)
- [Usage](docs/USAGE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [MVP test status](docs/TEST_REPORT.md)
- [Changelog](CHANGELOG.md)

## License

MIT. See [LICENSE](LICENSE).
