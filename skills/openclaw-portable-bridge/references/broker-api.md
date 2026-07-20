# Broker adapter reference

The broker is an HTTP implementation detail intended to remain on loopback.
Production agents should call a trusted adapter or MCP server rather than hold
the administrator token directly.

## Administrative endpoints

Send `Authorization: Bearer <admin-token>` and JSON bodies.

- `POST /v1/admin/approve`: `{"requestId":"…","minutes":20}`
- `POST /v1/admin/command`: `{"requestId":"…","command":{…}}`
- `GET /v1/admin/results?id=<request-id>&consume=true`
- `POST /v1/admin/revoke`: `{"requestId":"…"}`

A command has `ID`, `Name`, optional `params`, and an RFC3339 `Deadline`.
Command IDs must be unique within the operator workflow. Set short deadlines
for inspections and bounded deadlines for asynchronous jobs.

Command submission is idempotent within a session. Reusing an ID with the same
name and parameters returns the existing state; reusing it with a different
payload returns `409`. Queue saturation also returns `409` with `queueDepth`,
`queueLimit`, `retryAfterSeconds`, and a matching `Retry-After` header.

The guest delivery flow is `poll -> ack -> execute -> result`. A polled command
has a short lease and is requeued if the guest does not acknowledge it. Results
are accepted only for an acknowledged running command with the matching ID and
name.

## Common parameter shapes

```json
{"ID":"inspect-1","Name":"system.info","Deadline":"2026-01-01T00:00:30Z"}
{"ID":"shell-1","Name":"shell.run","params":{"command":"uname -a"},"Deadline":"2026-01-01T00:00:30Z"}
{"ID":"ps-1","Name":"powershell.run","params":{"script":"Get-Date"},"Deadline":"2026-01-01T00:00:30Z"}
{"ID":"list-1","Name":"files.list","params":{"path":"/tmp","offset":0,"limit":100,"filter":""},"Deadline":"2026-01-01T00:00:30Z"}
```

Use `shell.start`, `shell.status`, and `shell.cancel` for long-running commands.
Use chunked file operations for larger transfers and verify the final SHA-256.

## Adapter design

Expose typed tools such as `list_pending`, `approve`, `reject`, `command`,
`results`, and `revoke`. Keep secrets server-side, validate every argument,
redact tokens from logs, bind request IDs to the approving conversation, and
make result consumption explicit. Do not expose a generic unauthenticated HTTP
proxy to the model.

The current MVP does not yet provide `list_pending` or `reject` as first-class
broker endpoints; an adapter must derive pending requests from its trusted
event stream, and rejection should be implemented explicitly before claiming a
complete approval UX.
