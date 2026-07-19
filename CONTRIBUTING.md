# Contributing

Thank you for helping improve OpenClaw Portable Bridge. Contributions are
welcome from first-time and experienced contributors.

## Good ways to help

- reproduce and document behavior on Windows ARM64, Linux ARM64, or macOS;
- improve consent UX, accessibility, documentation, and packaging;
- add protocol tests, fuzz tests, or adversarial security tests;
- implement isolated roadmap items behind reviewed capability boundaries;
- review the threat model and report non-sensitive findings in issues.

Look for issues labelled `good first issue` or `help wanted`. Before starting a
large feature or protocol change, open a design issue so scope and security
properties can be agreed first.

## Development workflow

Requirements: Go 1.24 or newer. Docker is additionally required by the signed
multi-platform packaging script.

```sh
go test ./...
go test -race ./...
go vet ./...
```

Keep changes focused and add tests for behavior changes. Pull requests should
explain the user impact, security impact, platforms tested, and any remaining
limitations. Do not commit generated binaries, local configuration, audit
logs, tokens, private endpoints, device identifiers, or signing keys.

## Security boundaries

The project must remain visible, consent-driven, temporary, capability-scoped,
and prerequisite-free on the guest. Changes that add persistence, hidden
execution, protection bypasses, credential collection, or automatic privilege
escalation will not be accepted. Elevated Windows commands must continue to
require ordinary local UAC consent.

Report vulnerabilities privately through GitHub Security Advisories as
described in [SECURITY.md](SECURITY.md). Never place exploit details or real
deployment secrets in a public issue.

## Pull-request checklist

- [ ] Tests and documentation are updated.
- [ ] `go test ./...`, `go test -race ./...`, and `go vet ./...` pass.
- [ ] No deployment-specific or sensitive data is included.
- [ ] New protocol fields and capabilities are bounded and documented.
- [ ] Platform support claims distinguish compile testing from hardware tests.
