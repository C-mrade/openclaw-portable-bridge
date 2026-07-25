# Threat model

Assets: guest files/processes, ephemeral session keys, broker approval state,
OpenClaw Gateway authority, USB release integrity, and audit evidence.

Primary threats: stolen USB, malicious broker client, replay, token theft,
path traversal/junction escape, command confusion, oversized messages, update
substitution, and compromised relay. Controls in PoC: ephemeral Ed25519 key,
signed canonical request, 64 KiB request cap, strict JSON fields, capability
authorization fixed at pairing time, short token expiry, hashed server-side tokens,
constant-time comparison, explicit local approval, revocation, pairing rate
limits, replay rejection, capability profiles, path containment, bounded
messages/transfers, a loopback-only listener intended for a TLS reverse proxy,
signed launcher payloads, and logs without full tokens/private keys.

Developer commands run with the interactive user's authority. Administrative
commands require a separate local UAC approval and the project does not bypass
Windows security controls. Owned Windows processes are placed in kill-on-close
Job Objects so cancellation can terminate their descendants.

Residual risks: the durable broker snapshot format does not yet have stable
schema migrations; transport TLS is supplied by the deployment proxy rather
than the broker; recovered acknowledged commands remain explicitly uncertain
until their result arrives; reparse-point and time-of-check/time-of-use attacks
need a larger adversarial Windows matrix; Authenticode, the native GUI, full
ConPTY integration, conversation-bound approval, and proactive agent approval
notifications are incomplete.
Operators must keep the broker on loopback, protect its admin token, expose it
only through authenticated TLS infrastructure, and treat Developer sessions as
equivalent to temporary remote control by the approved operator.
