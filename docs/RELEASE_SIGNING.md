# Release signing operations

Portable Bridge releases use an Ed25519 trust root. The private key must never
be committed, copied to the portable package, or printed in logs.

## Canonical operator layout

The standard build automatically reads:

```text
~/.config/openclaw-portable-bridge/signing/release.key
~/.config/openclaw-portable-bridge/signing/release.pub
```

Both files and their parent directory must remain private to the operator.
`release.key` contains the base64 private key and `release.pub` contains the
matching base64 public key. Environment variables may override these paths for
CI or an external signer, but normal operator builds must not depend on
session-only exports.

Before every release, `build-release.sh` verifies that the key pair matches.
It then signs the public USB configuration, every payload, and every manifest.
The build fails before publishing if the key is missing, mismatched, or the
package checksum inventory does not verify.

## Backup and recovery

Keep only encrypted backups outside the operator host. For an SSH Ed25519
recipient:

```sh
age -R ~/.ssh/id_ed25519.pub \
  -o /secure/backup/release.key.age \
  ~/.config/openclaw-portable-bridge/signing/release.key
```

Test recovery into a temporary private directory and compare it byte-for-byte
with the primary key. Never decrypt directly into the repository or USB.

## Rotation

Rotate only when the original key is unavailable or potentially exposed:

1. Preserve the last package signed by the old trust root for rollback/audit.
2. Generate a new key at the canonical path.
3. Produce and verify an encrypted off-host backup.
4. Rebuild the entire portable package. Do not mix old launchers, payloads,
   signatures, or signed configuration with the new trust root.
5. Verify `SHA256SUMS.txt`, signature rejection, and a clean guest launch.
6. Record the rotation reason and release version without recording keys.

Existing packages remain internally valid under their embedded old public key,
but cannot accept components signed by the new key.
