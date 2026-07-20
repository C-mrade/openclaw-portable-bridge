# Changelog

## 0.5.1-mvp-dev

- Give the staged client and every child process a stable bridge-owned working
  directory so shells continue working when the USB directory disappears.
- Add idempotent command IDs and reject conflicting reuse of an existing ID.
- Add delivery leases and explicit client acknowledgement before execution;
  unacknowledged deliveries return to the queue.
- Correlate results with acknowledged running commands and reject duplicate or
  unsolicited results.
- Return structured queue depth, limit, and retry information.
- Distinguish cancellation requests from jobs that already completed.
- Configure explicit HTTP read, write, header, and idle timeouts.
- Retry transient long-poll failures with bounded exponential backoff instead
  of converting a brief disconnect into a client-initiated revocation.
- Advance the wire protocol to version 2; broker and guest must be upgraded
  together.

## 0.5.0-mvp-dev

- Translate the launcher and client console experience to English.
- Add signed launcher/client targets for Windows ARM64, Linux x64/ARM64, and
  macOS Intel/Apple Silicon.
- Add platform-aware capability profiles and native read-only inventory
  commands for Linux and macOS.
- Add an OpenClaw/Hermes-compatible operator skill and broker adapter reference.

## 0.4.2-mvp-dev

- Force UTF-8 console input, output, and pipeline encoding for Windows
  PowerShell 5.1 while retaining BOM-based script parsing.
- Preserve Unicode characters outside OEM code pages in captured output.
- Document the operator deployment path and prerequisite-free guest workflow.

## 0.4.1-mvp-dev

- Write structured PowerShell scripts with a UTF-8 BOM for Windows PowerShell
  5.1 compatibility.

## 0.4.0-mvp-dev

- Add Windows Job Object process-tree containment.
- Add structured PowerShell execution and CLIXML filtering.
- Add OEM, UTF-8, and UTF-16 output normalization.
- Add bounded chunked transfers and native ConPTY lifecycle primitives.
