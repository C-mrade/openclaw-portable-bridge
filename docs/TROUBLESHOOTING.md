# Troubleshooting

- **SmartScreen warning:** expected without Authenticode reputation. Inspect
  hashes; do not bypass corporate policy or disable SmartScreen.
- **Manifest/payload signature failed:** do not run the client. Rebuild from a
  trusted checkout using the offline release key.
- **Broker unavailable/DNS failure:** no command capability is granted. Close
  the launcher; it removes its staging directory.
- **Pairing expired/rejected:** start again to generate a new Ed25519 identity.
- **Path outside approved directories:** select the intended directory locally;
  the broker cannot expand it remotely.
- **Temporary files remain:** the launcher prints the exact owned staging path.
  Do not remove unrelated `%TEMP%` content.

