# Architecture decision record — 2026-07-19

## Protocol assumptions

- The reference implementation was designed against OpenClaw Gateway protocol
  v4. Operators must verify compatibility with their installed version.
- Official remote nodes use the Gateway WebSocket, device pairing (`role=node`),
  then separate node command-surface approval. `system.run` also has node-local
  exec approvals. Revocation disconnects the node role.
- A guest cannot bootstrap through a tailnet it has not joined. It therefore
  needs an outbound-reachable HTTPS endpoint with no reusable tailnet secret.

## Decision

Use a minimal public HTTPS/WSS pairing broker on TCP 443 as the only guest
bootstrap surface. The portable client makes outbound-only connections. The
broker never exposes a shell and never knows the permanent Gateway token. A
local adapter on the operator-controlled host translates approved sessions
to the official Gateway Node v4 protocol. The first PoC keeps broker HTTP on
loopback only; public routing is deliberately deferred until authentication,
rate limits, replay protection, and TLS deployment tests pass.

Do not embed tsnet credentials on the USB. A deployment may publish the
loopback broker through Tailscale Funnel, a dedicated TLS reverse proxy, or a
small relay. The public repository deliberately contains no live endpoint.

## Bootstrap comparison

| Option | Decision | Reason |
|---|---|---|
| HTTPS/WSS 443 broker | chosen design | outbound-only, works through most NATs/proxies |
| Tailscale Funnel | supported publisher | low operational cost; non-standard ports may be blocked |
| Cloudflare Tunnel | fallback | strong transport, extra third-party dependency |
| VPS relay | fallback | controllable but adds host/patching burden |
| pre-provisioned tsnet key | rejected | reusable secret and awkward revocation/bootstrap |
| direct Gateway exposure | rejected | exposes high-value control plane |
