# Threat model (PoC)

Assets: guest files/processes, ephemeral session keys, broker approval state,
OpenClaw Gateway authority, USB release integrity, and audit evidence.

Primary threats: stolen USB, malicious broker client, replay, token theft,
path traversal/junction escape, command confusion, oversized messages, update
substitution, and compromised relay. Controls in PoC: ephemeral Ed25519 key,
signed canonical request, 64 KiB request cap, strict JSON fields, capability
pinning to `system.info`, short token expiry, hashed server-side tokens,
constant-time comparison, explicit local approval, revocation, loopback-only
listener, and logs without full tokens/private keys.

Not yet production-ready: no durable state, no rate limiter, no TLS/WSS, no
USB launcher signature verification, no file sandbox, no OpenClaw adapter, and
no Windows GUI. The broker must not be published until these are implemented.

