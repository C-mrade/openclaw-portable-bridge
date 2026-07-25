# OpenClaw Portable Bridge approval plugin

This plugin claims Telegram callbacks in the `bridge:` namespace before they
enter the agent queue. It validates OpenClaw's sender authorization, a private
sender allowlist, a private conversation allowlist, the broker request ID, the
comparison code, and the guest-requested duration before invoking the typed
`bridge-operator` CLI.

Successful approve, reject, mismatch, and expiry outcomes edit the original
Telegram prompt in place and clear its buttons. Transient broker failures throw
without clearing the buttons so the callback can be retried.
