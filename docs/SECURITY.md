# Security operations

## Revoke sessions and USB identities

1. Revoke the active broker session (the token hash is invalidated immediately).
2. Remove the corresponding temporary OpenClaw node role with
   `openclaw nodes remove --node <id>` once the adapter is enabled.
3. To revoke every USB, stop the broker and rotate its server-side approval
   secret. USB media contain no reusable Gateway or Tailscale credentials.

## Rotate release keys

Generate a new offline Ed25519 key, rebuild the launcher with the new public
key, sign a complete new payload and replace releases atomically. Never copy
the private key to USB or source control. Store it outside the repository with
permissions restricted to the release operator.

## SmartScreen and Authenticode

Internal Ed25519 verification prevents substituted payload execution but does
not establish Windows publisher reputation. Until an Authenticode certificate
is acquired, Windows may display SmartScreen warnings. Do not bypass them.
The build keeps the launcher as a normal PE file ready for later `signtool`
signing using an OV or EV code-signing certificate.
