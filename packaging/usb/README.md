# USB packaging output

This directory is populated by `scripts/build-release.sh`. Copy
`config/bridge-public.example.json` to `config/bridge-public.json` and set an
HTTPS broker URL and a non-secret USB identifier for your deployment.

Generated executables, manifests, signatures, checksums, logs, and local
configuration are excluded from source control.
