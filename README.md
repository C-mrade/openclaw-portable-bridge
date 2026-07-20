# OpenClaw Portable Bridge

Experimental, security-first bridge for connecting a guest computer to an
operator-controlled broker without installing Python, Node.js, Docker, Git,
Tailscale, or OpenClaw on the guest. Windows 10/11 x64 is the validated target;
Windows ARM64, Linux x64/ARM64, and macOS Intel/Apple Silicon builds are now
available as experimental targets.

> **Guest experience:** copy the prepared `OPENCLAW_BRIDGE` directory to a USB
> drive, launch the matching `OPENCLAW BRIDGE` binary, choose a profile, and approve the
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

It does not install services on the guest, persist after exit, bypass operating
system protections, or expose an OpenClaw Gateway token. Administrator
execution is never automatic: Developer mode must be confirmed locally and,
on Windows, each elevated command requires a separate UAC approval.

## Platform support

| Platform | Build | Runtime validation | Elevation |
| --- | --- | --- | --- |
| Windows 10/11 x64 | supported | validated | local UAC per command |
| Windows ARM64 | experimental | compile-tested | local UAC per command |
| Linux x64/ARM64 | experimental | Linux x64 smoke-tested | not exposed |
| macOS Intel/Apple Silicon | experimental | compile-tested | not exposed |

All builds are standalone Go binaries. Linux and macOS currently use a visible
terminal UI and user-level commands. macOS application bundles, notarization,
and native graphical interfaces remain future work.

## Status

This is an MVP, not a production remote-management product. The current UI is
a visible console. Native ConPTY primitives are present, but the persistent
terminal protocol and UI are not complete. A native GUI, WebSocket streaming,
Telegram approval buttons,
the official OpenClaw Node v4 adapter, Authenticode signing, and the complete
adversarial test matrix remain future work. Review
[docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) before exposing a broker publicly.
The project's production invariants and release horizons are defined in the
[Production vision](docs/PRODUCTION_VISION.md).

## Quick start for operators

There are two distinct environments:

1. **Operator/server:** builds and signs the package, runs the loopback broker,
   and publishes only that broker through HTTPS.
2. **Windows guest:** runs the finished USB package with no prerequisites.

Start with the [operator quickstart](docs/QUICKSTART.md), then use the complete
[Deployment](docs/DEPLOYMENT.md) guide for hardened or customized setups.
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
./scripts/build-release.sh 0.5.0-mvp-dev
```

Copy `packaging/usb/config/bridge-public.example.json` to
`packaging/usb/config/bridge-public.json` and replace the placeholder values
for your own deployment. Generated binaries, signatures, manifests, local
configuration, logs, and release keys are intentionally ignored by Git.

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Production vision](docs/PRODUCTION_VISION.md)
- [Threat model](docs/THREAT_MODEL.md)
- [Security operations](docs/SECURITY.md)
- [Deployment and packaging](docs/DEPLOYMENT.md)
- [Operator quickstart](docs/QUICKSTART.md)
- [Agent integration](docs/AGENT_INTEGRATION.md)
- [Usage](docs/USAGE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [MVP test status](docs/TEST_REPORT.md)
- [Changelog](CHANGELOG.md)
- [Roadmap](ROADMAP.md)

## Contributing

Contributions are welcome, including documentation improvements, platform
testing, security review, packaging, UI work, and agent integrations. Read
[CONTRIBUTING.md](CONTRIBUTING.md) before opening an issue or pull request.
Security vulnerabilities must be reported privately as described in
[SECURITY.md](SECURITY.md).

An installable agent skill is available at
[`skills/openclaw-portable-bridge`](skills/openclaw-portable-bridge/SKILL.md).

## License

MIT. See [LICENSE](LICENSE).
