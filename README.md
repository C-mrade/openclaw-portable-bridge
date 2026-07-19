# OpenClaw Portable Bridge

Experimental, security-first bridge for connecting a Windows 10/11 x64 guest
to an operator-controlled broker without installing Python, Node.js, Docker,
Git, Tailscale, or OpenClaw on the guest.

The project currently provides:

- a standalone Go launcher that verifies an Ed25519-signed manifest and
  payload before staging the client under `%TEMP%`;
- a visible, consent-driven portable client with ephemeral session identity;
- an HTTP broker designed to remain on loopback behind a TLS reverse proxy or
  Tailscale Funnel;
- explicit pairing approval, short-lived tokens, revocation, replay
  protection, rate limiting, capability profiles, scoped file access, and
  application audit logs.

It does not install services on the guest, persist after exit, request
administrator privileges, bypass Windows protections, or expose an OpenClaw
Gateway token.

## Status

This is an MVP, not a production remote-management product. The current UI is
a visible console. Telegram approval buttons, the official OpenClaw Node v4
adapter, Authenticode signing, and the complete adversarial test matrix remain
future work. Review [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) before exposing
a broker publicly.

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
./scripts/build-release.sh 0.1.0
```

Copy `packaging/usb/config/bridge-public.example.json` to
`packaging/usb/config/bridge-public.json` and replace the placeholder values
for your own deployment. Generated binaries, signatures, manifests, local
configuration, logs, and release keys are intentionally ignored by Git.

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Threat model](docs/THREAT_MODEL.md)
- [Security operations](docs/SECURITY.md)
- [Usage](docs/USAGE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [MVP test status](docs/TEST_REPORT.md)

## License

MIT. See [LICENSE](LICENSE).
