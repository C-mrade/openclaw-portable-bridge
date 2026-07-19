# Roadmap

OpenClaw Portable Bridge is an experimental MVP. Roadmap items are ordered by
what most improves safe, understandable adoption.

## Next

- Typed operator CLI and MCP adapter for pending requests, approval, commands,
  results, and revocation.
- Telegram approval flow built on the same typed adapter.
- One-command deployment checks and package verification.
- Protocol version negotiation and compatibility tests.

## Later

- Complete persistent ConPTY transport with input, resize, acknowledgements,
  cancellation, and backpressure.
- Native visible guest UI for activity, transfers, consent, and revocation.
- macOS `.app` packaging, Developer ID signing, and notarization.
- Authenticode signing support for Windows releases.
- Representative Windows ARM64, Linux ARM64, and macOS hardware validation.

## Release-quality gates

- Complete adversarial network, authentication, command, transfer, and cleanup
  test matrix.
- Reproducible release procedure and signed source releases.
- Stable protocol and documented upgrade/rollback path.
- Independent security review before a production-stable claim.

See [docs/STATUS.md](docs/STATUS.md) for current implementation details.
