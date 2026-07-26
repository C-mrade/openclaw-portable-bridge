# Portable packaging output

This directory is populated by `scripts/build-release.sh`. Copy
`config/bridge-public.example.json` to `config/bridge-public.json` and set an
HTTPS broker URL and a non-secret USB identifier for your deployment.

Launchers are generated under `launchers/<os>-<arch>` and signed payloads under
`payload/<os>-<arch>`. The release root contains clearly labelled convenience
launchers for Windows x64, Linux x64, and Apple Silicon macOS. The `.exe`
suffix on the native Linux convenience launcher intentionally works around
FAT/VFAT `showexec` mounts and does not imply a Windows binary.

Generated executables, manifests, signatures, checksums, logs, and local
configuration are excluded from source control.
