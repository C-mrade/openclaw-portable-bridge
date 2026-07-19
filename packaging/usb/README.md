# Portable packaging output

This directory is populated by `scripts/build-release.sh`. Copy
`config/bridge-public.example.json` to `config/bridge-public.json` and set an
HTTPS broker URL and a non-secret USB identifier for your deployment.

Launchers are generated under `launchers/<os>-<arch>` and signed payloads under
`payload/<os>-<arch>`. The root `OPENCLAW BRIDGE.exe` is the Windows x64
convenience launcher.

Generated executables, manifests, signatures, checksums, logs, and local
configuration are excluded from source control.
